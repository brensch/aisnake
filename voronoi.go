package main

import (
	"container/list"
	"math"
)

// GenerateVoronoi generates a Voronoi diagram for the given board.
func GenerateVoronoi2(board Board) [][]int {
	voronoi := make([][]int, board.Height)
	for i := range voronoi {
		voronoi[i] = make([]int, board.Width)
		for j := range voronoi[i] {
			voronoi[i][j] = -1 // Initialize all positions as unassigned
		}
	}

	for y := 0; y < board.Height; y++ {
		for x := 0; x < board.Width; x++ {
			minDistance := math.MaxInt32
			closestSnake := -1

			for k, snake := range board.Snakes {
				distance := manhattanDistance(Point{x, y}, snake.Head)
				if distance < minDistance {
					minDistance = distance
					closestSnake = k
				} else if distance == minDistance {
					// In case of a tie, leave the cell unassigned (-1)
					closestSnake = -1
				}
			}

			voronoi[y][x] = closestSnake
		}
	}

	return voronoi
}

// manhattanDistance calculates the Manhattan distance between two points.
func manhattanDistance(a, b Point) int {
	return int(math.Abs(float64(a.X-b.X)) + math.Abs(float64(a.Y-b.Y)))
}

// GenerateFloodFill generates a flood-fill diagram for the given board, where each cell is filled with the closest snake according to legal moves.
func GenerateVoronoi(board Board) [][]int {
	floodFill := make([][]int, board.Height)
	for i := range floodFill {
		floodFill[i] = make([]int, board.Width)
		for j := range floodFill[i] {
			floodFill[i][j] = -1 // Initialize all positions as unassigned
		}
	}

	// Queue for flood fill, each element contains the current point, snake index, and the current depth (distance from snake head)
	queue := list.New()

	// Initialize the queue with the heads of all snakes
	for k, snake := range board.Snakes {
		if snake.Health > 0 && len(snake.Body) > 0 { // Skip dead or empty snakes
			queue.PushBack(floodFillNode{snake.Head, k, 0})
			floodFill[snake.Head.Y][snake.Head.X] = k
		}
	}

	// Perform flood fill
	for queue.Len() > 0 {
		element := queue.Front()
		node := element.Value.(floodFillNode)
		queue.Remove(element)

		snakeIndex := node.snakeIndex
		currentPoint := node.point

		// Get legal moves for the current point
		for _, direction := range AllDirections {
			newPoint := moveHead(currentPoint, direction)

			// Ensure new point is within bounds and unassigned
			if newPoint.X >= 0 && newPoint.X < board.Width && newPoint.Y >= 0 && newPoint.Y < board.Height {
				if floodFill[newPoint.Y][newPoint.X] == -1 {
					// Check if the move is legal for the snake at snakeIndex
					if isLegalMove(board, snakeIndex, newPoint) {
						floodFill[newPoint.Y][newPoint.X] = snakeIndex
						queue.PushBack(floodFillNode{newPoint, snakeIndex, node.depth + 1})
					}
				}
			}
		}
	}

	return floodFill
}

// floodFillNode represents a node in the flood-fill queue.
type floodFillNode struct {
	point      Point
	snakeIndex int
	depth      int // Depth is the number of moves from the snake's head
}

// isLegalMove checks if a move to a new point is legal for the snake.
func isLegalMove(board Board, snakeIndex int, newHead Point) bool {
	snake := board.Snakes[snakeIndex]

	// Check if the new head is within the board boundaries
	if newHead.X < 0 || newHead.X >= board.Width || newHead.Y < 0 || newHead.Y >= board.Height {
		return false
	}

	// Check for collisions with other snakes
	for i := range board.Snakes {
		otherSnake := board.Snakes[i]

		// Don't consider dead snakes
		if len(otherSnake.Body) == 0 || otherSnake.Health == 0 {
			continue
		}

		// Check for collisions with other snakes' bodies
		snakeWithoutTail := otherSnake.Body[0 : len(otherSnake.Body)-1]
		for _, segment := range snakeWithoutTail {
			if newHead == segment {
				return false
			}
		}

		// Check for head-to-head collisions where the other snake is longer or equal
		if newHead == otherSnake.Head && len(otherSnake.Body) >= len(snake.Body) {
			return false
		}
	}

	// If no collision, the move is legal
	return true
}
