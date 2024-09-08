package main

import (
	"context"
	"math"
	"math/rand"
)

type Node struct {
	Board      Board
	SnakeIndex int
	Parent     *Node
	Children   []*Node
	Visits     int
	Score      float64
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

func expand(node *Node) {
	if isTerminal(node.Board) {
		// Assign a losing score if the current node is a terminal state
		if isSnakeDead(node.Board.Snakes[node.SnakeIndex]) {
			node.Score = -1.0 // Losing score (negative) - the snake is dead
		} else {
			node.Score = 1.0 // Winning score (positive) - the snake is alive and no other snakes are left
		}
		return
	}

	moves := generateSafeMoves(node.Board, node.SnakeIndex)
	for _, move := range moves {
		newBoard := copyBoard(node.Board)
		applyMove(&newBoard, node.SnakeIndex, move)

		// Fix the SnakeIndex to reflect the snake who is deciding the next move
		nextSnakeIndex := (node.SnakeIndex + 1) % len(node.Board.Snakes)
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
	if n.Visits == 0 {
		return math.Inf(1)
	}

	exploitation := n.Score / float64(n.Visits)
	exploration := explorationParam * math.Sqrt(math.Log(float64(parent.Visits))/float64(n.Visits))

	// Penalize states where the current snake is dead
	if isSnakeDead(n.Board.Snakes[n.SnakeIndex]) {
		return exploitation - 10 // Strong penalty (negative) to discourage exploring dead states
	}

	return exploitation + exploration
}

func bestChild(node *Node, explorationParam float64) *Node {
	bestValue := -math.Inf(1)
	var bestNode *Node

	for _, child := range node.Children {
		value := child.UCT(node, explorationParam)
		if value > bestValue {
			bestValue = value
			bestNode = child
		}
	}

	return bestNode
}

func MCTS(ctx context.Context, rootBoard Board, iterations int) *Node {
	rootNode := NewNode(rootBoard, 0)
	expand(rootNode)

	for i := 0; i < iterations; i++ {
		select {
		case <-ctx.Done():
			return rootNode
		default:
			node := rootNode

			// Selection
			for len(node.Children) > 0 {
				node = bestChild(node, 1.41)
			}

			// Expansion
			if !isTerminal(node.Board) && node.Visits > 0 {
				expand(node)
				if len(node.Children) > 0 {
					node = node.Children[0] // Select the first child for simulation
				}
			}

			// Simulation
			score := simulate(node.Board, node.SnakeIndex)

			// Backpropagation using parent pointers, ensuring we don't hit nil
			score = -score
			for n := node; n != nil; n = n.Parent {
				n.Visits++
				n.Score += score
				// No need to flip score for this scenario
			}
		}
	}
	return rootNode
}

func simulate(board Board, startingSnakeIndex int) float64 {
	currentBoard := copyBoard(board)
	currentSnakeIndex := startingSnakeIndex
	depth := 0
	maxDepth := 100 // Prevent infinite loops

	for !isTerminal(currentBoard) && depth < maxDepth {
		moves := generateSafeMoves(currentBoard, currentSnakeIndex)
		if len(moves) == 0 {
			break
		}
		move := moves[rand.Intn(len(moves))]
		applyMove(&currentBoard, currentSnakeIndex, move)
		currentSnakeIndex = (currentSnakeIndex + 1) % len(currentBoard.Snakes)
		depth++
	}

	return evaluateBoard(currentBoard, startingSnakeIndex)
}

func evaluateBoard(board Board, snakeIndex int) float64 {
	// If the current snake is dead, it's losing
	if isSnakeDead(board.Snakes[snakeIndex]) {
		return -1.0 // Losing
	}

	// Voronoi evaluation: Calculate the area controlled by each snake
	voronoi := GenerateVoronoi(board)
	// snakeAreas := make([]float64, len(board.Snakes))
	totalCells := float64(board.Width * board.Height)
	controlledCells := 0.0

	// Count the number of cells each snake controls in the Voronoi diagram
	for y := 0; y < board.Height; y++ {
		for x := 0; x < board.Width; x++ {
			if voronoi[y][x] == snakeIndex {
				controlledCells++
			}
			// owner := voronoi[y][x]
			// if owner >= 0 && owner < len(snakeAreas) {
			// 	snakeAreas[owner]++
			// }
		}
	}
	return controlledCells / totalCells

	// // Normalize the areas (optional, but could be useful for debugging)
	// for i := range snakeAreas {
	// 	snakeAreas[i] = snakeAreas[i] / totalCells
	// }

	// // Get the area controlled by the current snake
	// currentSnakeArea := snakeAreas[snakeIndex]

	// // Check if the current snake controls more area than all others
	// for i, area := range snakeAreas {
	// 	if i != snakeIndex && !isSnakeDead(board.Snakes[i]) {
	// 		if area > currentSnakeArea {
	// 			return -1.0 // Losing: Another snake controls more area
	// 		}
	// 	}
	// }

	// // Check if the current snake controls the same area as any other snake
	// for i, area := range snakeAreas {
	// 	if i != snakeIndex && !isSnakeDead(board.Snakes[i]) {
	// 		if math.Abs(area-currentSnakeArea) < 1e-5 {
	// 			return 0.0 // Draw: Areas are equal (or close enough to call it a draw)
	// 		}
	// 	}
	// }

	// // If no other snake controls more area and areas are not equal, the current snake is winning
	// return 1.0 // Winning
}
