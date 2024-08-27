package main

import (
	"fmt"
	"math"
	"strings"
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

// VisualizeVoronoi visualizes the Voronoi diagram on the console.
func VisualizeVoronoi(voronoi [][]int, snakes []Snake) string {
	var sb strings.Builder

	for y := 0; y < len(voronoi); y++ {
		for x := 0; x < len(voronoi[y]); x++ {
			owner := voronoi[y][x]
			if owner == -1 {
				sb.WriteString(". ") // Unassigned cells
			} else {
				sb.WriteString(fmt.Sprintf("%c ", 'A'+owner)) // Each snake gets a unique letter
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
