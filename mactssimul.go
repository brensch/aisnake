package main

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
)

// MultiNode represents a node in the MCTS tree for multiple snakes.
type MultiNode struct {
	Board                Board
	Parent               *MultiNode
	Children             []*MultiNode
	Visits               int64
	Scores               []float64 // Cumulative scores for each snake.
	MyScore              []float64 // Initial evaluation scores for each snake.
	UnexpandedMoveCombos [][]Direction
	MoveCombo            []Direction // The move combination that led to this node.

	mutex sync.Mutex
}

func (m *MultiNode) GetVisits() int64 {
	return m.Visits
}

func (m *MultiNode) UCTer() float64 {
	return MultiUCT(m)
}

func (m *MultiNode) GetChildren() []GenericNode {
	children := make([]GenericNode, len(m.Children))
	for _, child := range m.Children {
		children = append(children, child)
	}
	return children
}

func (m *MultiNode) Visualise() string {
	node := m
	if node == nil {
		return ""
	}

	nodeID := fmt.Sprintf("Node_%p", node)
	var scores []float64
	for _, score := range node.Scores {
		scores = append(scores, score/float64(node.Visits))
	}
	// Using <br/> instead of \n to create HTML-based line breaks that D3 can interpret
	nodeLabel := fmt.Sprintf("%s\nVisits: %d\nScores: %v\nMy Score: %+v\n\n\n",
		nodeID, node.Visits, scores, node.MyScore)
	voronoi := GenerateVoronoi(node.Board)
	controlledPositions := make([]int, len(node.Board.Snakes))
	for _, row := range voronoi {
		for _, owner := range row {
			if owner >= 0 && owner < len(controlledPositions) {
				controlledPositions[owner]++
			}
		}
	}
	for i, count := range controlledPositions {
		nodeLabel += fmt.Sprintf("Snake %c: %d cells, %d len\n", 'A'+i, count, len(node.Board.Snakes[i].Body))
	}
	// Add the board state visualization
	boardVisualization := visualizeBoard(node.Board, WithNewlineCharacter("\n"))
	nodeLabel += boardVisualization

	// Add controlled positions from the Voronoi diagram

	nodeVoronoiVisualization := VisualizeVoronoi(voronoi, node.Board.Snakes, WithNewlineCharacter("\n"))
	nodeLabel += "\n" + nodeVoronoiVisualization

	// Return the node label with HTML line breaks
	return nodeLabel

}

func (m *MultiNode) GetBoard() Board {
	return m.Board
}

// MultiNewNode initializes a new MultiNode and generates possible move combinations.
func MultiNewNode(board Board, parent *MultiNode, moveCombo []Direction) *MultiNode {
	node := &MultiNode{
		Board:                copyBoard(board),
		Parent:               parent,
		Children:             make([]*MultiNode, 0),
		Visits:               0,
		Scores:               make([]float64, len(board.Snakes)),
		MyScore:              nil,
		UnexpandedMoveCombos: nil,
		MoveCombo:            moveCombo,
	}

	// If the node is terminal, there are no moves to expand.
	if isTerminal(board) {
		return node
	}

	// For each alive snake, generate possible moves.
	possibleMoves := make([][]Direction, len(board.Snakes))
	allSnakesDead := true
	for i, snake := range board.Snakes {
		if isSnakeDead(snake) {
			possibleMoves[i] = []Direction{}
			continue
		}
		allSnakesDead = false
		moves := generateSafeMoves(board, i)
		if len(moves) == 0 {
			// If no safe moves, include all possible moves.
			moves = []Direction{Up, Down, Left, Right}
		}
		possibleMoves[i] = moves
	}

	if allSnakesDead {
		return node
	}

	// Generate all combinations of moves.
	moveCombos := generateMoveCombinations(possibleMoves)

	node.UnexpandedMoveCombos = moveCombos

	return node
}

// generateMoveCombinations generates all possible combinations of moves for the snakes.
func generateMoveCombinations(possibleMoves [][]Direction) [][]Direction {
	var results [][]Direction
	current := make([]Direction, len(possibleMoves))
	generateMoveCombinationsRecursive(possibleMoves, 0, current, &results)
	return results
}

func generateMoveCombinationsRecursive(possibleMoves [][]Direction, index int, current []Direction, results *[][]Direction) {
	if index == len(possibleMoves) {
		combo := make([]Direction, len(current))
		copy(combo, current)
		*results = append(*results, combo)
		return
	}

	if len(possibleMoves[index]) == 0 {
		// If no possible moves for this snake, set to NoMove.
		current[index] = NoMove
		generateMoveCombinationsRecursive(possibleMoves, index+1, current, results)
	} else {
		for _, move := range possibleMoves[index] {
			current[index] = move
			generateMoveCombinationsRecursive(possibleMoves, index+1, current, results)
		}
	}
}

// MultiMCTS performs the Monte Carlo Tree Search with concurrency for multiple snakes.
func MultiMCTS(ctx context.Context, gameID string, rootBoard Board, iterations int, numWorkers int, gameStates map[string]*MultiNode) *MultiNode {
	// Generate the hash for the current board state.
	// boardKey := boardHash(rootBoard)
	// var rootNode *MultiNode
	// // If the board state is already known, use the existing node.
	// if existingNode, ok := gameStates[boardKey]; ok {
	// 	slog.Info("board cache lookup", "hit", true, "cache_size", len(gameStates), "visits", existingNode.Visits)
	// 	rootNode = existingNode
	// } else {
	// 	slog.Info("board cache lookup", "hit", false, "cache_size", len(gameStates))
	// 	// Initialize rootNode.
	// 	rootNode = MultiNewNode(rootBoard, nil, []Direction{})
	// }
	rootNode := MultiNewNode(rootBoard, nil, []Direction{})

	for i := 0; i < numWorkers; i++ {
		go MultiWorker(ctx, rootNode)
	}

	<-ctx.Done()

	return rootNode
}

func MultiWorker(ctx context.Context, rootNode *MultiNode) {
	for {
		// Check if the context is done.
		select {
		case <-ctx.Done():
			return
		default:
			// Continue execution.
		}

		node := MultiSelectNode(ctx, rootNode)

		// If context was cancelled during selection.
		if node == nil || ctx.Err() != nil {
			return
		}

		// Simulation.
		var scores []float64
		if atomic.LoadInt64(&node.Visits) == 0 {
			// Evaluate the board.
			scores = evaluateBoard(node.Board, modules)
			if len(scores) == 0 {
				fmt.Println(visualizeBoard(node.Board))
				panic(node)
			}
			// Store the initial evaluation score.
			node.MyScore = scores
			atomic.AddInt64(&node.Visits, 1)
			for i := range node.Scores {
				atomicAddFloat64(&node.Scores[i], scores[i])
			}
		} else {
			// Node has been visited before; use existing MyScore.
			scores = node.MyScore
			if len(scores) == 0 {
				continue
			}

			// Update visits and scores atomically.
			for i := range node.Scores {
				atomicAddFloat64(&node.Scores[i], scores[i])
			}
			atomic.AddInt64(&node.Visits, 1)
		}

		// Backpropagation.
		n := node.Parent
		for n != nil {
			if ctx.Err() != nil {
				return
			}
			atomic.AddInt64(&n.Visits, 1)

			// Update scores and visits atomically.
			for i := range n.Scores {
				atomicAddFloat64(&n.Scores[i], scores[i])
			}
			n = n.Parent
		}
	}
}

// MultiSelectNode traverses the tree, expanding nodes as needed for multiple snakes.
func MultiSelectNode(ctx context.Context, rootNode *MultiNode) *MultiNode {
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
		// If there are unexpanded move combinations, expand one.
		if len(node.UnexpandedMoveCombos) > 0 {
			// Pop a move combo from UnexpandedMoveCombos.
			moveCombo := node.UnexpandedMoveCombos[0]
			node.UnexpandedMoveCombos = node.UnexpandedMoveCombos[1:]
			node.mutex.Unlock()

			// Create child node.
			newBoard := copyBoard(node.Board)
			applyMoves(&newBoard, moveCombo)

			child := MultiNewNode(newBoard, node, moveCombo)

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
		bestChildNode := MultiBestChild(node)
		if bestChildNode == nil {
			// No valid child found.
			return node
		}

		// Move to the best child.
		node = bestChildNode
	}
}

// MultiBestChild selects the best child node based on a Nash equilibrium placeholder.
func MultiBestChild(node *MultiNode) *MultiNode {
	if len(node.Children) == 0 {
		return nil // No children available.
	}

	bestValue := -math.MaxFloat64
	var bestNodes []*MultiNode

	for _, child := range node.Children {
		if child == nil {
			continue // Skip nil children.
		}

		// Placeholder for Nash equilibrium computation.
		value := MultiUCT(child)

		if value > bestValue {
			bestValue = value
			bestNodes = []*MultiNode{child}
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

// MultiUCT calculates a placeholder value for the Nash equilibrium.
func MultiUCT(n *MultiNode) float64 {
	visits := atomic.LoadInt64(&n.Visits)
	if visits == 0 {
		return math.MaxFloat64
	}

	parentVisits := atomic.LoadInt64(&n.Parent.Visits)
	exploitation := 0.0
	for _, score := range n.Scores {
		exploitation += score / float64(visits)
	}
	// Average exploitation over all snakes.
	exploitation /= float64(len(n.Scores))

	exploration := 1.41 * math.Sqrt(math.Log(float64(parentVisits))/float64(visits))

	return exploitation + exploration
}

// applyMoves applies a set of moves to the board for all snakes.
func applyMoves(board *Board, moves []Direction) {
	// moves[i] is the move for snake i.
	// Apply moves to all snakes.
	for i, move := range moves {
		if isSnakeDead(board.Snakes[i]) {
			continue
		}
		if move == NoMove {
			// No move for this snake.
			continue
		}
		// Apply the move for snake i.
		applyMove(board, i, move)
	}

	// Update the board state after all moves have been applied.
	updateBoard(board)
}

// updateBoard updates the board state after moves have been applied.
func updateBoard(board *Board) {
	// Implement collision detection, health loss, and other game rules.
	// This is a placeholder and should be filled with actual game logic.
}

func MultiDetermineBestMove(node *MultiNode, mySnakeIndex int) string {
	var bestChild *MultiNode
	maxVisits := int64(-1)

	for _, child := range node.Children {
		if child != nil && child.Visits > maxVisits {
			bestChild = child
			maxVisits = child.Visits
		}
	}

	if bestChild != nil && len(bestChild.MoveCombo) > mySnakeIndex {
		myMove := bestChild.MoveCombo[mySnakeIndex]
		if myMove == NoMove {
			// Our snake didn't move, choose a random move
			moves := []string{"up", "down", "left", "right"}
			return moves[rand.Intn(len(moves))]
		}
		bestMove := directionToString(myMove)
		return bestMove
	}

	// If no best child or move found, return a random move
	moves := []string{"up", "down", "left", "right"}
	return moves[rand.Intn(len(moves))]
}

func directionToString(dir Direction) string {
	switch dir {
	case Up:
		return "up"
	case Down:
		return "down"
	case Left:
		return "left"
	case Right:
		return "right"
	default:
		return ""
	}
}
