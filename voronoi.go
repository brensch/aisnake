package main

import (
	"math"
)

// GenerateVoronoi generates a Voronoi diagram for the given board.
func GenerateVoronoi(board Board) [][]int {
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

// evaluateBoard evaluates the board and returns a score based on the Voronoi diagram.
func evaluateBoard(board Board) float64 {
	voronoi := GenerateVoronoi(board)
	score := 0.0

	// Count the number of cells controlled by the snake at index 0
	for y := 0; y < board.Height; y++ {
		for x := 0; x < board.Width; x++ {
			if voronoi[y][x] == 0 { // 0 represents the snake at index 0
				score += 1.0
			}
		}
	}

	return score
}
