package main

// import (
// 	"context"
// 	"fmt"
// 	"math"
// 	"math/rand"
// )

// const maxMovesToIterate = 100
// const explorationParameter = 1.414

// // Node represents a node in the MCTS tree.
// type Node struct {
// 	Board              Board   // The current state of the game board
// 	Parent             *Node   // The parent node
// 	Children           []*Node // The children nodes
// 	Visits             int     // Number of times this node has been visited
// 	Score              float64 // The total score accumulated through simulations
// 	MyScore            float64
// 	UntriedMoves       []Direction // Moves that haven't been tried yet for the current snake
// 	SnakeIndex         int         // Index of the snake whose move is being evaluated
// 	SnakeIndexWhoMoved int
// 	Move               Direction // The move that was applied to reach this node
// }

// // NewNode creates a new MCTS node.
// func NewNode(board Board, parent *Node, snakeIndex, snakeIndexWhoMoved int, move Direction) *Node {
// 	return &Node{
// 		Board:              board,
// 		Parent:             parent,
// 		Children:           []*Node{},
// 		Visits:             0,
// 		Score:              0,
// 		UntriedMoves:       generateSafeMoves(board, snakeIndex),
// 		SnakeIndex:         snakeIndex,
// 		SnakeIndexWhoMoved: snakeIndexWhoMoved,
// 		Move:               move,
// 	}
// }

// func (n *Node) UCTValue(c *Node) float64 {
// 	exploitation := 0.0
// 	exploration := 0.0
// 	if c.Visits != 0 {
// 		exploitation = c.Score / float64(c.Visits)
// 		exploration = explorationParameter * math.Sqrt(math.Log(float64(n.Visits))/float64(c.Visits))
// 	}
// 	return exploitation + exploration
// }

// func (n *Node) SelectChild() *Node {
// 	var selected *Node
// 	maxUcbValue := -math.MaxFloat64

// 	for _, child := range n.Children {
// 		if child.Visits == 0 {
// 			// Prioritize unvisited children first
// 			return child
// 		}

// 		// Calculate UCB value
// 		uctValue := n.UCTValue(child)

// 		// Handle NaN values
// 		if math.IsNaN(uctValue) {
// 			fmt.Println("got nan")
// 			continue // Skip this child if UCB calculation results in NaN
// 		}

// 		// Select the child with the maximum UCB value
// 		if uctValue > maxUcbValue {
// 			selected = child
// 			maxUcbValue = uctValue
// 		}
// 	}

// 	return selected
// }

// // Expand adds new child nodes by generating all possible moves for the current snake.
// func (n *Node) Expand() []*Node {
// 	if len(n.UntriedMoves) == 0 || len(n.Board.Snakes) == 0 {
// 		return nil
// 	}

// 	children := []*Node{}
// 	for _, move := range n.UntriedMoves {
// 		newBoard := copyBoard(n.Board)
// 		applyMove(&newBoard, n.SnakeIndex, move)

// 		// Ensure there are still snakes on the board before calculating the next index
// 		if len(newBoard.Snakes) == 0 {
// 			// If no snakes are left after applying the move, return the children created so far
// 			return children
// 		}

// 		nextSnakeIndex := (n.SnakeIndex + 1) % len(newBoard.Snakes)

// 		childNode := NewNode(newBoard, n, nextSnakeIndex, n.SnakeIndex, move)
// 		children = append(children, childNode)
// 	}

// 	// Clear untried moves since all have been expanded
// 	n.UntriedMoves = []Direction{}
// 	n.Children = append(n.Children, children...) // Save the expanded children to the node
// 	return children
// }

// // Update updates the node's visit count and score.
// func (n *Node) Update(score float64) {
// 	n.Visits++
// 	n.Score += score
// }

// func MCTS(ctx context.Context, rootBoard Board, iterations, numGoroutines int) *Node {
// 	rootNode := NewNode(rootBoard, nil, 0, -1, Unset) // Start with the first snake, with an Unset move

// 	for i := 0; i < iterations; i++ {
// 		select {
// 		case <-ctx.Done():
// 			return rootNode // Exit early if the context is canceled
// 		default:
// 		}

// 		node := rootNode

// 		// Selection and early expansion
// 		for len(node.Children) > 0 && len(node.UntriedMoves) == 0 {
// 			node = node.SelectChild()
// 		}

// 		// Expansion
// 		if !boardIsTerminal(node.Board) && len(node.UntriedMoves) > 0 {
// 			children := node.Expand()
// 			if len(children) > 0 {
// 				node.Children = children
// 				node = children[0] // Proceed with one of the expanded children
// 			}
// 		}

// 		// Get the score based on the correct snake's perspective (the one who moved last)
// 		instantScore := evaluateBoard(node.Board, node.SnakeIndex)

// 		// Update the current node's MyScore with the correct score
// 		node.MyScore = instantScore

// 		// Backpropagation - Ensure we're backpropagating the score for the snake that made the move
// 		for node != nil {
// 			node.Update(instantScore) // Propagate the correct score up the tree
// 			node = node.Parent
// 		}
// 	}

// 	return rootNode
// }

// func randomSafeMove(board Board, snakeIndex int) Direction {
// 	safeMoves := generateSafeMoves(board, snakeIndex)
// 	if len(safeMoves) == 0 {
// 		return Up // Default move if no safe moves are found (this should generally not happen)
// 	}

// 	return safeMoves[rand.Intn(len(safeMoves))] // Return a random move from the list of safe moves
// }
