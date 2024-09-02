package main

import (
	"context"
	"math"
	"math/rand"
	"sync"
)

const maxMovesToIterate = 1

// Node represents a node in the MCTS tree.
type Node struct {
	Board              Board       // The current state of the game board
	Parent             *Node       // The parent node
	Children           []*Node     // The children nodes
	Visits             int         // Number of times this node has been visited
	Score              float64     // The total score accumulated through simulations
	UntriedMoves       []Direction // Moves that haven't been tried yet for the current snake
	SnakeIndex         int         // Index of the snake whose move is being evaluated
	SnakeIndexWhoMoved int
	Move               Direction // The move that was applied to reach this node
}

// NewNode creates a new MCTS node.
func NewNode(board Board, parent *Node, snakeIndex, snakeIndexWhoMoved int, move Direction) *Node {
	return &Node{
		Board:              board,
		Parent:             parent,
		Children:           []*Node{},
		Visits:             0,
		Score:              0,
		UntriedMoves:       generateSafeMoves(board, snakeIndex),
		SnakeIndex:         snakeIndex,
		SnakeIndexWhoMoved: snakeIndexWhoMoved,
		Move:               move,
	}
}

const explorationParameter = 100

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
		// The next snake's turn, but keep the index for evaluation of this move
		nextSnakeIndex := (n.SnakeIndex + 1) % len(newBoard.Snakes)

		childNode := NewNode(newBoard, n, nextSnakeIndex, n.SnakeIndex, move)
		children = append(children, childNode)
	}

	// Clear untried moves since all have been expanded
	n.UntriedMoves = []Direction{}
	n.Children = append(n.Children, children...) // Save the expanded children to the node
	return children
}

// Update updates the node's visit count and score.
func (n *Node) Update(score float64) {
	n.Visits++
	n.Score += score
}

func MCTS(ctx context.Context, rootBoard Board, iterations, numGoroutines int) *Node {
	rootNode := NewNode(rootBoard, nil, 0, -1, Unset) // Start with the first snake, with an Unset move
	// fmt.Println("root", rootNode.SnakeIndex)
	// fmt.Println(visualizeBoard(rootNode.Board))

	if iterations < 2 {
		iterations = 2
	}

	for i := 0; i < iterations; i++ {
		select {
		case <-ctx.Done():
			return rootNode // Exit early if the context is canceled
		default:
		}

		node := rootNode

		// Selection and early expansion
		for len(node.Children) > 0 && len(node.UntriedMoves) == 0 {
			// fmt.Println("doing selection")
			node = node.SelectChild()
		}

		// Expansion
		if !boardIsTerminal(node.Board) && len(node.UntriedMoves) > 0 {
			children := node.Expand()
			// fmt.Println("expanding", children)
			if len(children) > 0 {
				node.Children = children // Save the expanded children to the node
				node = children[0]       // Proceed with one of the expanded children
			}
		}

		// Parallel Simulation (Rollout)
		results := make(chan float64, numGoroutines)
		var wg sync.WaitGroup

		// Start each rollout from the board state of the selected node
		for g := 0; g < numGoroutines; g++ {
			wg.Add(1)
			go func(node *Node) {
				defer wg.Done()

				// fmt.Println("base", node.SnakeIndex)
				// fmt.Println(visualizeBoard(node.Board))
				boardCopy := copyBoard(node.Board) // Start from the board state at the current node
				currentSnakeIndex := node.SnakeIndex
				moves := 0

				for !boardIsTerminal(boardCopy) {
					select {
					case <-ctx.Done():
						return
					default:
					}

					move := randomSafeMove(boardCopy, currentSnakeIndex)
					applyMove(&boardCopy, currentSnakeIndex, move)
					// everyone died
					if len(boardCopy.Snakes) == 0 {
						results <- 0
						return
					}

					// Update snake index for the next move
					currentSnakeIndex = (currentSnakeIndex + 1) % len(boardCopy.Snakes) // Next snake's turn
					moves++
					if moves == maxMovesToIterate {
						// fmt.Println("max iters")
						break
					}
				}

				// Evaluate from the perspective of the snake that led to this node, not the current snake
				score := evaluateBoard(boardCopy, node.SnakeIndexWhoMoved)
				results <- score

				// fmt.Println("rolled", score, boardCopy.Height*boardCopy.Width, node.SnakeIndex)
				// fmt.Println(visualizeBoard(boardCopy))
				// fmt.Println(VisualizeVoronoi(GenerateVoronoi(boardCopy), boardCopy.Snakes))

			}(node) // Pass the node to start the rollout from
		}

		// Collect and average the results
		wg.Wait()
		close(results)

		totalScore := 0.0
		count := 0

		for score := range results {
			select {
			case <-ctx.Done():
				return rootNode
			default:
			}

			totalScore += score
			count++
		}

		averageScore := totalScore / float64(count)

		// Backpropagation
		for node != nil {
			node.Update(averageScore)
			node = node.Parent
		}
	}

	return rootNode
}

func randomSafeMove(board Board, snakeIndex int) Direction {
	safeMoves := generateSafeMoves(board, snakeIndex)
	if len(safeMoves) == 0 {
		return Up // Default move if no safe moves are found (this should generally not happen)
	}

	return safeMoves[rand.Intn(len(safeMoves))] // Return a random move from the list of safe moves
}
