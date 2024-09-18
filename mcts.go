package main

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"
	"unsafe"
)

// Node represents a node in the MCTS tree.
type Node struct {
	Board      Board
	SnakeIndex int // The index of the snake whose turn it is at this node.
	Parent     *Node
	Children   []*Node
	Visits     int64
	Score      float64 // Cumulative score from simulations.
	// MyScore         []float64 // The initial evaluation score of this node.
	MyScore atomic.Value // Will store []float64

	UnexpandedMoves []Direction

	mutex sync.Mutex
}

// NewNode initializes a new Node and generates possible moves.
func NewNode(board Board, snakeIndex int, parent *Node) *Node {
	node := &Node{
		Board:           copyBoard(board), //TODO: explore if this is necessary
		SnakeIndex:      snakeIndex,
		Parent:          parent,
		Children:        make([]*Node, 0),
		Visits:          0,
		Score:           0,
		UnexpandedMoves: nil,
	}

	// If the node is terminal, there are no moves to expand.
	if isTerminal(board) {
		return node
	}

	// Compute the next snake's index.
	nextSnakeIndex := (snakeIndex + 1) % len(board.Snakes)
	originalNextSnake := nextSnakeIndex

	// do not generate nodes for dead snakes
	for {
		if !isSnakeDead(board.Snakes[nextSnakeIndex]) {
			break
		}
		nextSnakeIndex = (nextSnakeIndex + 1) % len(board.Snakes)
		// this means we've checked all the snakes
		if nextSnakeIndex == originalNextSnake {
			return node
		}

	}

	// Generate possible moves for the next snake.
	moves := generateSafeMoves(board, nextSnakeIndex)
	if len(moves) == 0 {
		// If no safe moves, include all possible moves.
		moves = []Direction{Up, Down, Left, Right}
	}

	node.UnexpandedMoves = moves
	return node
}

// isTerminal checks if the game has reached a terminal state.
func isTerminal(board Board) bool {
	aliveSnakesCount := 0
	for _, snake := range board.Snakes {
		if !isSnakeDead(snake) {
			aliveSnakesCount++
		}
	}
	return aliveSnakesCount <= 1
}

// isSnakeDead checks if a snake is dead.
func isSnakeDead(snake Snake) bool {
	return len(snake.Body) == 0 || snake.Health <= 0
}

// UCT calculates the Upper Confidence Bound for Trees (UCT) value.
func (n *Node) UCT(explorationParam float64) float64 {
	visits := atomic.LoadInt64(&n.Visits)
	if visits == 0 {
		return math.MaxFloat64
	}

	parentVisits := atomic.LoadInt64(&n.Parent.Visits)
	exploitation := n.Score / float64(visits)
	exploration := explorationParam * math.Sqrt(math.Log(float64(parentVisits))/float64(visits))

	return exploitation + exploration
}

// bestChild selects the best child node based on the UCT value.
func bestChild(node *Node, explorationParam float64) *Node {
	if len(node.Children) == 0 {
		return nil // No children available.
	}

	bestValue := -math.MaxFloat64
	var bestNodes []*Node

	for _, child := range node.Children {
		if child == nil {
			continue // Skip nil children.
		}

		value := child.UCT(explorationParam)

		if value > bestValue {
			bestValue = value
			bestNodes = []*Node{child}
		} else if value == bestValue {
			bestNodes = append(bestNodes, child)
		}
	}

	// Return the first among the best nodes (can be randomized if desired).
	if len(bestNodes) > 0 {
		return bestNodes[0]
	}
	return nil
}

// MCTS performs the Monte Carlo Tree Search with concurrency.
func MCTS(ctx context.Context, gameID string, rootBoard Board, iterations int, numWorkers int, gameStates map[string]*Node) *Node {
	// Generate the hash for the current board state.
	boardKey := boardHash(rootBoard)
	var rootNode *Node
	// If the board state is already known, use the existing node.
	if existingNode, ok := gameStates[boardKey]; ok {
		slog.Info("board cache lookup", "hit", true, "cache_size", len(gameStates), "visits", existingNode.Visits)
		rootNode = existingNode
	} else {
		slog.Info("board cache lookup", "hit", false, "cache_size", len(gameStates))
		// Initialize rootNode with the current snake's index (e.g., -1 for the initial state).
		rootNode = NewNode(rootBoard, -1, nil)
	}

	for i := 0; i < numWorkers; i++ {
		go worker(ctx, rootNode)
	}

	<-ctx.Done()

	return rootNode
}

func worker(ctx context.Context, rootNode *Node) {
	for {
		// Check if the context is done.
		select {
		case <-ctx.Done():
			return
		default:
			// Continue execution.
		}

		node := selectNode(ctx, rootNode)

		// If context was cancelled during selection.
		if node == nil || ctx.Err() != nil {
			return
		}

		// Simulation.
		var scores []float64
		if atomic.LoadInt64(&node.Visits) == 0 {
			// Evaluate from the perspective of the root snake.
			scores = evaluateBoard(node.Board, modules)
			if len(scores) == 0 {
				fmt.Println(visualizeBoard(node.Board))
				panic(node)
			}
			// Atomically store the initial evaluation score.
			node.MyScore.Store(scores)
			atomic.AddInt64(&node.Visits, 1)
			atomicAddFloat64(&node.Score, scores[node.SnakeIndex])
		} else {
			// Node has been visited before; use existing MyScore.
			scoresInterface := node.MyScore.Load()
			// this indicates the node has not finished computing its scores.
			// seems like this means i'm not locking correctly, but not sure it's worth fixing.
			// played around with various different locking strategies but they all end up slower.
			if scoresInterface == nil {
				continue
			}
			scores = scoresInterface.([]float64)

			// Update visits and score atomically.
			atomicAddFloat64(&node.Score, scores[node.SnakeIndex])
			atomic.AddInt64(&node.Visits, 1)
		}

		// Backpropagation.
		n := node.Parent
		for n != nil {
			if ctx.Err() != nil {
				return
			}
			atomic.AddInt64(&n.Visits, 1)

			if n.SnakeIndex == -1 {
				break
			}
			// Flip the score to represent the opponent's perspective.
			score := scores[n.SnakeIndex]

			// Update score and visits atomically.
			atomicAddFloat64(&n.Score, score)
			n = n.Parent
		}
	}
}

// selectNode traverses the tree, expanding nodes as needed.
func selectNode(ctx context.Context, rootNode *Node) *Node {
	node := rootNode

	for {
		// Check for context cancellation.
		select {
		case <-ctx.Done():
			return nil
		default:
			// Continue execution.
		}

		node.mutex.Lock()
		// If there are unexpanded moves, expand one.
		if len(node.UnexpandedMoves) > 0 {
			// Pop a move from UnexpandedMoves.
			move := node.UnexpandedMoves[0]
			node.UnexpandedMoves = node.UnexpandedMoves[1:]
			node.mutex.Unlock()

			// Create child node.
			newBoard := copyBoard(node.Board)
			nextSnakeIndex := (node.SnakeIndex + 1) % len(node.Board.Snakes)
			applyMove(&newBoard, nextSnakeIndex, move)

			child := NewNode(newBoard, nextSnakeIndex, node)

			// Append the child to node.Children.
			node.mutex.Lock()
			node.Children = append(node.Children, child)
			node.mutex.Unlock()

			return child
		}
		// No unexpanded moves.
		node.mutex.Unlock()

		// If the node is a leaf node (no children), return it.
		node.mutex.Lock()
		if len(node.Children) == 0 {
			node.mutex.Unlock()
			return node
		}
		node.mutex.Unlock()

		// Node is expanded and has children.
		// Select the best child.
		bestChildNode := bestChild(node, 1.41)
		if bestChildNode == nil {
			// No valid child found.
			return node
		}

		// Move to the best child.
		node = bestChildNode
	}
}

// atomicAddFloat64 performs an atomic addition on a float64 variable.
func atomicAddFloat64(addr *float64, delta float64) {
	for {
		old := atomic.LoadUint64((*uint64)(unsafe.Pointer(addr)))
		newVal := math.Float64frombits(old) + delta
		newBits := math.Float64bits(newVal)
		if atomic.CompareAndSwapUint64((*uint64)(unsafe.Pointer(addr)), old, newBits) {
			return
		}
	}
}
