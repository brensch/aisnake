package main

import (
	"context"
	"math"
	"sync"
)

const maxMovesToIterate = 100

// Node represents a node in the MCTS tree.
type Node struct {
	Board        Board       // The current state of the game board
	Parent       *Node       // The parent node
	Children     []*Node     // The children nodes
	Visits       int         // Number of times this node has been visited
	Score        float64     // The total score accumulated through simulations
	UntriedMoves []Direction // Moves that haven't been tried yet for the current snake
	SnakeIndex   int         // Index of the snake whose move is being evaluated
	Move         Direction   // The move that was applied to reach this node
}

// NewNode creates a new MCTS node.
func NewNode(board Board, parent *Node, snakeIndex int, move Direction) *Node {
	return &Node{
		Board:        board,
		Parent:       parent,
		Children:     []*Node{},
		Visits:       0,
		Score:        0,
		UntriedMoves: generateSafeMoves(board, snakeIndex),
		SnakeIndex:   snakeIndex,
		Move:         move,
	}
}

const explorationParameter = 1.414

func (n *Node) UCTValue(c *Node) float64 {
	exploitation := c.Score / float64(c.Visits)
	exploration := explorationParameter * math.Sqrt(math.Log(float64(n.Visits))/float64(c.Visits))
	return exploitation + exploration
}

func (n *Node) SelectChild() *Node {
	var selected *Node
	maxUcbValue := -math.MaxFloat64

	for _, child := range n.Children {
		if child.Visits == 0 {
			// Prioritize unvisited children first
			return child
		}

		// Calculate UCB value
		uctValue := n.UCTValue(child)

		// Handle NaN values
		if math.IsNaN(uctValue) {
			continue // Skip this child if UCB calculation results in NaN
		}

		// Select the child with the maximum UCB value
		if uctValue > maxUcbValue {
			selected = child
			maxUcbValue = uctValue
		}
	}

	return selected
}

// Expand adds new child nodes by generating all possible moves for the current snake.
func (n *Node) Expand() []*Node {
	if len(n.UntriedMoves) == 0 {
		return nil
	}

	children := []*Node{}
	for _, move := range n.UntriedMoves {
		newBoard := copyBoard(n.Board)
		applyMove(&newBoard, n.SnakeIndex, move)

		nextSnakeIndex := (n.SnakeIndex + 1) % len(newBoard.Snakes) // Determine the next snake's turn

		childNode := NewNode(newBoard, n, nextSnakeIndex, move)
		n.Children = append(n.Children, childNode)
		children = append(children, childNode)
	}

	// Clear untried moves since all have been expanded
	n.UntriedMoves = []Direction{}
	return children
}

// Update updates the node's visit count and score.
func (n *Node) Update(score float64) {
	n.Visits++
	n.Score += score
}

// MCTS runs the Monte Carlo Tree Search algorithm with context for cancellation.
func MCTS(ctx context.Context, rootBoard Board, iterations, numGoroutines int) *Node {
	rootNode := NewNode(rootBoard, nil, 0, -1) // Start with the first snake

	// We need to do at least 2 iterations, or we will have no children
	if iterations < 2 {
		iterations = 2
	}

	for i := 0; i < iterations; i++ {
		// Check if the context has been cancelled
		select {
		case <-ctx.Done():
			return rootNode // Return the current state of the tree if cancelled
		default:
		}

		node := rootNode
		board := rootBoard

		// Selection with early expansion
		for len(node.Children) > 0 && len(node.UntriedMoves) == 0 {
			node = node.SelectChild()
			board = node.Board
		}

		// Expansion: Expand all untried moves before any simulation
		if !boardIsTerminal(board) && len(node.UntriedMoves) > 0 {
			children := node.Expand()
			if len(children) > 0 {
				// Continue with a randomly selected child from the expanded nodes
				node = children[0] // Adjust this selection logic if needed
				board = node.Board
			}
		}

		// Parallel Simulation (rollout)
		results := make(chan float64, numGoroutines)
		var wg sync.WaitGroup

		for g := 0; g < numGoroutines; g++ {
			wg.Add(1)
			go func(boardCopy Board, startSnakeIndex int) {
				defer wg.Done()

				moves := 0
				currentSnakeIndex := startSnakeIndex

				for !boardIsTerminal(boardCopy) {
					// Check if the context has been cancelled during the simulation
					select {
					case <-ctx.Done():
						return
					default:
					}

					move := randomSafeMove(boardCopy, currentSnakeIndex)
					applyMove(&boardCopy, currentSnakeIndex, move)

					currentSnakeIndex = (currentSnakeIndex + 1) % len(boardCopy.Snakes) // Move to the next snake
					moves++
					if moves == maxMovesToIterate {
						break
					}
				}

				// Evaluate the final board state
				score := evaluateBoard(boardCopy)
				results <- score

			}(copyBoard(board), node.SnakeIndex) // Pass a copy of the board and the starting snake index to each goroutine
		}

		// Wait for all rollouts to complete
		go func() {
			wg.Wait()
			close(results)
		}()

		// Aggregate the results
		totalScore := 0.0
		count := 0

		for score := range results {
			// Check if the context has been cancelled during result aggregation
			select {
			case <-ctx.Done():
				return rootNode // Return the current state of the tree if cancelled
			default:
			}

			totalScore += score
			count++
		}

		// Calculate the average score
		averageScore := totalScore / float64(count)

		// Backpropagation: Update the node with the average score
		for node != nil {
			node.Update(averageScore)
			node = node.Parent
		}
	}

	return rootNode
}

// randomSafeMove generates a random safe move for the given snake.
func randomSafeMove(board Board, snakeIndex int) Direction {
	safeMoves := generateSafeMoves(board, snakeIndex)
	return safeMoves[0] // Return the first safe move; you might want to randomize this selection
}
