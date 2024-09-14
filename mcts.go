package main

import (
	"context"
	"log/slog"
	"math"
	"sync"
)

type Node struct {
	Board      Board
	SnakeIndex int
	Parent     *Node
	Children   []*Node
	Visits     int
	Score      float64
	MyScore    float64

	mu         sync.RWMutex // Read-Write Mutex to protect the node's state
	childrenMu sync.RWMutex // Read-Write Mutex for protecting children initialization
}

func NewNode(board Board, snakeIndex int) *Node {
	return &Node{
		Board:      board,
		SnakeIndex: snakeIndex,
		Children:   make([]*Node, 0), // Ensuring Children slice is initialized
		Visits:     0,
		Score:      0,
	}
}

// Expand the node to add children based on the board's state
func expand(node *Node) {
	node.childrenMu.Lock() // Protect child initialization
	defer node.childrenMu.Unlock()

	// If already expanded, return immediately
	if len(node.Children) > 0 {
		return
	}

	nextSnakeIndex := (node.SnakeIndex + 1) % len(node.Board.Snakes)

	moves := generateSafeMoves(node.Board, nextSnakeIndex)
	if len(moves) == 0 {
		moves = append(moves, Up)
	}

	for _, move := range moves {
		newBoard := copyBoard(node.Board)
		applyMove(&newBoard, nextSnakeIndex, move)

		child := NewNode(newBoard, nextSnakeIndex)
		child.Parent = node
		node.Children = append(node.Children, child)
	}
}

func isTerminal(board Board) bool {
	aliveSnakesCount := 0
	for _, snake := range board.Snakes {
		if !isSnakeDead(snake) {
			aliveSnakesCount++
		}
	}
	return aliveSnakesCount <= 1
}

// isSnakeDead checks if a snake is dead by looking at its health and body length.
func isSnakeDead(snake Snake) bool {
	return len(snake.Body) == 0 || snake.Health <= 0
}

func (n *Node) UCT(parent *Node, explorationParam float64) float64 {
	// Acquire locks in a consistent order to prevent deadlocks
	if n == parent {
		// Avoid double-locking if n and parent are the same node
		n.mu.RLock()
		defer n.mu.RUnlock()
	} else if n.SnakeIndex < parent.SnakeIndex {
		n.mu.RLock()
		parent.mu.RLock()
		defer n.mu.RUnlock()
		defer parent.mu.RUnlock()
	} else {
		parent.mu.RLock()
		n.mu.RLock()
		defer n.mu.RUnlock()
		defer parent.mu.RUnlock()
	}

	if n.Visits == 0 {
		return math.MaxFloat64
	}

	exploitation := n.Score / float64(n.Visits)
	exploration := explorationParam * math.Sqrt(math.Log(float64(parent.Visits))/float64(n.Visits))

	return exploitation + exploration
}

// Select the best child node based on the UCT value
func bestChild(node *Node, explorationParam float64) *Node {
	node.childrenMu.RLock() // Protect access to children
	defer node.childrenMu.RUnlock()

	if len(node.Children) == 0 {
		return nil // No children available
	}

	bestValue := -math.MaxFloat64
	var bestNode *Node

	for _, child := range node.Children {
		if child == nil {
			continue // Skip nil children, in case of race condition or partial initialization
		}

		// Lock the child and parent for reading Visits and Score
		value := child.UCT(node, explorationParam)

		if value > bestValue {
			bestValue = value
			bestNode = child
		}
	}

	return bestNode
}

func MCTS(ctx context.Context, gameID string, rootBoard Board, iterations int, numWorkers int, gameStates map[string]*Node) *Node {
	// Generate the hash for the current board state
	boardKey := boardHash(rootBoard)
	var rootNode *Node
	// If the board state is already known, use the existing node
	if existingNode, ok := gameStates[boardKey]; ok {
		slog.Info("board cache lookup", "hit", true, "cache_size", len(gameStates), "visits", existingNode.Visits)
		rootNode = existingNode
	} else {
		// Otherwise, create a new node and add it to the game state map
		slog.Info("board cache lookup", "hit", false, "cache_size", len(gameStates))
		rootNode = NewNode(rootBoard, -1)
		expand(rootNode)
	}

	nodeChan := make(chan *Node, numWorkers) // Channel to distribute work

	// Central coordinator goroutine
	go func() {
		for i := 0; i < iterations; i++ {
			node := rootNode
			for {
				node.childrenMu.RLock()
				hasChildren := len(node.Children) > 0
				node.childrenMu.RUnlock()
				if !hasChildren {
					break
				}
				nextNode := bestChild(node, 1.41)
				if nextNode == nil {
					break // Break the loop if no valid child is found
				}
				node = nextNode
			}
			select {
			case nodeChan <- node:
			case <-ctx.Done():
				close(nodeChan)
				return
			}
		}
		close(nodeChan)
	}()

	// Worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for node := range nodeChan {
				if node == nil {
					continue // Skip processing if node is nil
				}

				// Lock node before processing
				node.mu.RLock()
				isTerminalNode := isTerminal(node.Board)
				visits := node.Visits
				node.mu.RUnlock()

				if !isTerminalNode && visits > 0 {
					expand(node)
				}

				// Simulation
				var score float64
				node.mu.Lock()
				if node.Visits == 0 {
					score = evaluateBoard(node.Board, node.SnakeIndex)
					node.MyScore = score
				} else {
					score = node.MyScore
				}
				node.mu.Unlock()

				// Backpropagation
				for n := node; n != nil; n = n.Parent {
					n.mu.Lock()
					n.Visits++
					n.Score += score
					n.mu.Unlock()
					score = -score
				}
			}
		}()
	}

	wg.Wait()
	return rootNode
}

func evaluateBoard(board Board, snakeIndex int) float64 {
	if isSnakeDead(board.Snakes[snakeIndex]) {
		return -2
	}

	// TODO: won't work for multiplayer
	if isSnakeDead(board.Snakes[(snakeIndex+1)%len(board.Snakes)]) {
		return 2
	}

	// Voronoi evaluation: Calculate the area controlled by each snake
	// note, not actually voronoi. brendonoi let's say.
	voronoi := GenerateVoronoi(board)
	totalCells := float64(board.Width * board.Height)
	controlledCells := 0.0
	opponentsCells := 0.0
	lengthBonus := 0.0

	// Count the number of cells each snake controls in the Voronoi diagram
	for y := 0; y < board.Height; y++ {
		for x := 0; x < board.Width; x++ {
			if voronoi[y][x] == snakeIndex {
				controlledCells++
			} else if voronoi[y][x] != -1 {
				opponentsCells++
			}
		}
	}

	// Calculate capped length bonus/penalty
	// TODO: this is probably not going to work for multiplayer
	mySnake := board.Snakes[snakeIndex]
	for i, opponent := range board.Snakes {
		if i != snakeIndex && !isSnakeDead(opponent) {
			lengthDifference := len(mySnake.Body) - len(opponent.Body)

			if lengthDifference >= 2 {
				// Cap the bonus for being at least 1 longer than the opponent
				lengthBonus += 0.3 * float64(lengthDifference) // Higher reward, but capped
			} else {
				// Penalize for being shorter
				lengthBonus += 0.1 * float64(lengthDifference)
			}
		}
	}

	// Return a score between -1 and 1 based on the difference in controlled areas,
	// with a capped length bonus/penalty.
	return ((controlledCells - opponentsCells) / totalCells) + lengthBonus
}
