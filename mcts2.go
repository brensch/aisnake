package main

import (
	"context"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"
	"unsafe"
)

// Node represents a node in the MCTS tree.
type Node struct {
	Board           Board
	SnakeIndex      int // The index of the snake whose turn it is at this node.
	Parent          *Node
	Children        []*Node
	Visits          int64
	Score           float64 // Cumulative score from simulations.
	MyScore         float64 // The initial evaluation score of this node.
	UnexpandedMoves []Direction

	mutex sync.Mutex
}

// NewNode initializes a new Node and generates possible moves.
func NewNode(board Board, snakeIndex int, parent *Node) *Node {
	node := &Node{
		Board:           board,
		SnakeIndex:      snakeIndex,
		Parent:          parent,
		Children:        make([]*Node, 0),
		Visits:          0,
		Score:           0,
		MyScore:         0,
		UnexpandedMoves: nil,
	}

	// If the node is terminal, there are no moves to expand.
	if isTerminal(board) {
		return node
	}

	// Compute the next snake's index.
	nextSnakeIndex := (snakeIndex + 1) % len(board.Snakes)

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

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			worker(ctx, rootNode)
		}(i)
	}
	wg.Wait()

	return rootNode
}

// worker performs MCTS iterations, managing synchronization appropriately.
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
		if node == nil {
			return
		}

		// Simulation.
		var score float64
		if atomic.LoadInt64(&node.Visits) == 0 {
			// Evaluate from the perspective of the root snake.
			score = evaluateBoard(node.Board, node.SnakeIndex, modules)

			// Update node's own score and visits atomically.
			atomic.AddInt64(&node.Visits, 1)
			atomicAddFloat64(&node.Score, score)
			node.MyScore = score // Save the initial evaluation score.
		} else {
			// Node has been visited before; use existing MyScore.
			score = node.MyScore

			// Update visits and score atomically.
			atomicAddFloat64(&node.Score, score)
			atomic.AddInt64(&node.Visits, 1)
		}

		// Backpropagation.
		n := node.Parent
		for n != nil {
			// Flip the score to represent the opponent's perspective.
			score = -score

			// Update score and visits atomically.
			atomic.AddInt64(&n.Visits, 1)
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

// EvaluationFunc defines the function signature for evaluation modules.
type EvaluationFunc func(board Board, rootSnakeIndex int) float64

// EvaluationModule defines a struct that holds an evaluation function and its corresponding weight.
type EvaluationModule struct {
	EvalFunc EvaluationFunc
	Weight   float64
}

var (
	modules = []EvaluationModule{
		{
			EvalFunc: voronoiEvaluation,
			Weight:   6,
		},
		{
			EvalFunc: lengthEvaluation,
			Weight:   2,
		},
	}
)

// evaluateBoard evaluates the board state from the perspective of the root snake.
func evaluateBoard(board Board, rootSnakeIndex int, modules []EvaluationModule) float64 {
	if rootSnakeIndex < 0 || rootSnakeIndex >= len(board.Snakes) {
		// Invalid snake index.
		return 0
	}

	rootSnake := board.Snakes[rootSnakeIndex]

	// If the root snake is dead, return an extreme negative score.
	if isSnakeDead(rootSnake) {
		return -2
	}

	// Check if all opponents are dead.
	aliveOpponents := 0
	for i, snake := range board.Snakes {
		if i != rootSnakeIndex && !isSnakeDead(snake) {
			aliveOpponents++
		}
	}

	// If all opponents are dead, return an extreme positive score.
	if aliveOpponents == 0 {
		return 2
	}

	// Calculate the sum of all weights for normalization.
	totalWeight := 0.0
	for _, module := range modules {
		totalWeight += module.Weight
	}

	// Accumulate weighted evaluations from each module.
	totalScore := 0.0
	for _, module := range modules {
		moduleScore := module.EvalFunc(board, rootSnakeIndex)
		weightedScore := (module.Weight / totalWeight) * moduleScore
		totalScore += weightedScore
	}

	// Return the final score normalized between -1 and 1.
	if totalScore > 1 {
		return 1
	} else if totalScore < -1 {
		return -1
	}

	return totalScore
}

// voronoiEvaluation evaluates the board based on Voronoi control.
func voronoiEvaluation(board Board, rootSnakeIndex int) float64 {
	voronoi := GenerateVoronoi(board)
	totalCells := float64(board.Width * board.Height)
	rootControlledCells := 0.0
	opponentsControlledCells := 0.0

	// Count the number of cells each snake controls in the Voronoi diagram.
	for y := 0; y < board.Height; y++ {
		for x := 0; x < board.Width; x++ {
			if voronoi[y][x] == rootSnakeIndex {
				rootControlledCells++
			} else if voronoi[y][x] != -1 {
				opponentsControlledCells++
			}
		}
	}

	// Return the difference in controlled areas as a score.
	return (rootControlledCells - opponentsControlledCells) / totalCells
}

// lengthEvaluation evaluates the board based on the length of the root snake compared to opponents.
func lengthEvaluation(board Board, rootSnakeIndex int) float64 {
	rootSnake := board.Snakes[rootSnakeIndex]
	lengthBonus := 0.0

	// Calculate length bonus/penalty.
	for i, opponent := range board.Snakes {
		if i != rootSnakeIndex && !isSnakeDead(opponent) {
			lengthDifference := len(rootSnake.Body) - len(opponent.Body)

			if lengthDifference >= 2 {
				// Bonus for being longer.
				lengthBonus += 0.3 * float64(lengthDifference)
			} else {
				// Penalty for being shorter.
				lengthBonus += 0.1 * float64(lengthDifference)
			}
		}
	}

	return lengthBonus
}
