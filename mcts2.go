package main

import (
	"context"
	"log/slog"
	"math"
	"sync"
)

// Node represents a node in the MCTS tree.
type Node struct {
	Board      Board
	SnakeIndex int // The index of the snake whose turn it is at this node.
	Parent     *Node
	Children   []*Node
	Visits     int64
	Score      float64 // Cumulative score from simulations.
	MyScore    float64 // The initial evaluation score of this node.

	mutex       sync.Mutex
	cond        *sync.Cond
	isExpanding bool
	isExpanded  bool
}

// NewNode initializes a new Node with synchronization primitives.
func NewNode(board Board, snakeIndex int, parent *Node) *Node {
	node := &Node{
		Board:       board,
		SnakeIndex:  snakeIndex,
		Parent:      parent,
		Children:    make([]*Node, 0),
		Visits:      0,
		Score:       0,
		MyScore:     0, // Initialize MyScore to zero.
		isExpanding: false,
		isExpanded:  false,
	}
	node.cond = sync.NewCond(&node.mutex)
	return node
}

// expand adds children to the node based on the board's state.
func expand(node *Node) {
	// If the node is terminal, it can't be expanded.
	if isTerminal(node.Board) {
		node.mutex.Lock()
		node.isExpanding = false
		node.isExpanded = true
		node.cond.Broadcast()
		node.mutex.Unlock()
		return
	}

	// Prepare to expand without holding the lock.
	nextSnakeIndex := (node.SnakeIndex + 1) % len(node.Board.Snakes)

	// Generate moves for the current snake.
	moves := generateSafeMoves(node.Board, nextSnakeIndex)
	if len(moves) == 0 {
		// If no safe moves, include all possible moves.
		moves = []Direction{Up, Down, Left, Right}
	}

	newChildren := make([]*Node, 0, len(moves))
	for _, move := range moves {
		newBoard := copyBoard(node.Board)
		applyMove(&newBoard, nextSnakeIndex, move)

		child := NewNode(newBoard, nextSnakeIndex, node)
		newChildren = append(newChildren, child)
	}

	// Lock the node to update shared state.
	node.mutex.Lock()
	defer func() {
		node.isExpanding = false
		node.isExpanded = true
		node.cond.Broadcast()
		node.mutex.Unlock()
	}()

	node.Children = newChildren
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

// isSnakeDead checks if a snake is dead by looking at its health and body length.
func isSnakeDead(snake Snake) bool {
	return len(snake.Body) == 0 || snake.Health <= 0
}

// UCT calculates the Upper Confidence Bound for Trees (UCT) value.
func (n *Node) UCT(explorationParam float64) float64 {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if n.Visits == 0 {
		return math.MaxFloat64
	}

	exploitation := n.Score / float64(n.Visits)
	exploration := explorationParam * math.Sqrt(math.Log(float64(n.Parent.Visits))/float64(n.Visits))

	return exploitation + exploration
}

// bestChild selects the best child node based on the UCT value.
func bestChild(node *Node, explorationParam float64) *Node {
	// No need to lock node here as UCT function handles locking.
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

	// Randomly select among the best nodes (optional, for tie-breaking).
	// Here, we'll just return the first one.
	if len(bestNodes) > 0 {
		return bestNodes[0]
	}
	return nil
}

// SharedData holds shared information for workers.
type SharedData struct {
	rootNode *Node
	ctx      context.Context
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
		// Otherwise, create a new node and add it to the game state map.
		slog.Info("board cache lookup", "hit", false, "cache_size", len(gameStates))
		// Initialize rootNode with the current snake's index (e.g., 0 for our AI snake).
		rootNode = NewNode(rootBoard, -1, nil)
		// We don't expand the root node here; expansion is handled during selection.
	}

	sharedData := &SharedData{
		rootNode: rootNode,
		ctx:      ctx,
	}

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			worker(sharedData)
		}(i)
	}
	wg.Wait()

	return rootNode
}

// worker performs MCTS iterations, managing synchronization appropriately.
func worker(sharedData *SharedData) {
	for {
		// Check if the context is done.
		select {
		case <-sharedData.ctx.Done():
			return
		default:
			// Continue execution.
		}

		node := selectNode(sharedData.ctx, sharedData.rootNode)

		// If context was cancelled during selection.
		if node == nil {
			return
		}

		// Simulation.
		var score float64
		if node.Visits == 0 {
			// Evaluate from the perspective of the root snake.
			score = evaluateBoard(node.Board, node.SnakeIndex)

			// Lock the node to update its own score and visits.
			node.mutex.Lock()
			node.MyScore = score // Save the initial evaluation score.
			node.Score += score  // Update cumulative score.
			node.Visits++        // Increment visit count.
			node.mutex.Unlock()
		} else {
			// Node has been visited before; use existing MyScore.
			score = node.MyScore

			// Lock the node to update visits.
			node.mutex.Lock()
			node.Visits++
			node.mutex.Unlock()
		}

		// Backpropagation.
		n := node.Parent
		for n != nil {
			// Flip the score to represent the opponent's perspective.
			score = -score

			// Lock each node during backpropagation to update score and visits.
			n.mutex.Lock()
			n.Score += score
			n.Visits++
			n.mutex.Unlock()
			n = n.Parent
		}
	}
}

// selectNode traverses the tree, handling synchronization and avoiding unnecessary blocking.
func selectNode(ctx context.Context, rootNode *Node) *Node {
	node := rootNode

	// We will traverse the tree without holding locks except when necessary.
	for {
		// Check for context cancellation.
		select {
		case <-ctx.Done():
			return nil
		default:
			// Continue execution.
		}

		node.mutex.Lock()

		// If node is being expanded by another worker, avoid waiting.
		if node.isExpanding {
			node.mutex.Unlock()
			// Instead of waiting, start over from the root node.
			node = rootNode
			continue
		}

		// If node is not fully expanded, expand it.
		if !node.isExpanded {
			// Mark the node as expanding.
			node.isExpanding = true
			node.mutex.Unlock()

			// Expand the node without holding the lock.
			expand(node)

			// After expansion, start over from the root node.
			node = rootNode
			continue
		}

		// If the node is a leaf node (no children), return it.
		if len(node.Children) == 0 {
			node.mutex.Unlock()
			return node
		}

		// Node is expanded and has children.
		// Select the best child.
		bestChildNode := bestChild(node, 1.41)
		if bestChildNode == nil {
			// No valid child found.
			node.mutex.Unlock()
			return node
		}

		// Move to the best child without holding the parent lock.
		node.mutex.Unlock()
		node = bestChildNode
	}
}

// evaluateBoard evaluates the board state from the perspective of the root snake.
func evaluateBoard(board Board, rootSnakeIndex int) float64 {
	if rootSnakeIndex < 0 || rootSnakeIndex >= len(board.Snakes) {
		// Invalid snake index.
		return 0
	}

	rootSnake := board.Snakes[rootSnakeIndex]

	if isSnakeDead(rootSnake) {
		return -1
	}

	// Check if all opponents are dead.
	aliveOpponents := 0
	for i, snake := range board.Snakes {
		if i != rootSnakeIndex && !isSnakeDead(snake) {
			aliveOpponents++
		}
	}
	if aliveOpponents == 0 {
		return 1
	}

	// Voronoi evaluation: Calculate the area controlled by each snake.
	voronoi := GenerateVoronoi(board)
	totalCells := float64(board.Width * board.Height)
	rootControlledCells := 0.0
	opponentsControlledCells := 0.0
	lengthBonus := 0.0

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

	// Return a score based on the difference in controlled areas and length bonus.
	return ((rootControlledCells - opponentsControlledCells) / totalCells) + lengthBonus
}
