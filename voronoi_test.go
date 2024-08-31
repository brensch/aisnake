package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type VoronoiTestCase struct {
	Width       int
	Height      int
	Snakes      []Point
	Expected    [][]int
	Description string
}

var voronoiTestCases = []VoronoiTestCase{
	{
		Description: "Test Case 1: Snakes with the same position",
		Width:       5,
		Height:      5,
		Snakes: []Point{
			{X: 3, Y: 1},
			{X: 3, Y: 1},
		},
		Expected: [][]int{
			{-1, -1, -1, -1, -1},
			{-1, -1, -1, -1, -1},
			{-1, -1, -1, -1, -1},
			{-1, -1, -1, -1, -1},
			{-1, -1, -1, -1, -1},
		},
	},
	{
		Description: "Test Case 2: Multiple snakes with different positions",
		Width:       7,
		Height:      7,
		Snakes: []Point{
			{X: 4, Y: 0},
			{X: 3, Y: 4},
			{X: 5, Y: 5},
		},
		Expected: [][]int{
			{0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0},
			{1, 1, 1, 1, 0, -1, -1},
			{1, 1, 1, 1, 1, 2, 2},
			{1, 1, 1, 1, 1, 2, 2},
			{1, 1, 1, 1, 2, 2, 2},
			{1, 1, 1, 1, 2, 2, 2},
		},
	},
	{
		Description: "Test Case 3: Large grid with multiple snakes",
		Width:       10,
		Height:      10,
		Snakes: []Point{
			{X: 9, Y: 4},
			{X: 8, Y: 4},
			{X: 5, Y: 3},
			{X: 2, Y: 0},
		},
		Expected: [][]int{
			{3, 3, 3, 3, 3, -1, -1, -1, 1, 0},
			{3, 3, 3, 3, -1, 2, 2, -1, 1, 0},
			{3, 3, 3, -1, 2, 2, 2, -1, 1, 0},
			{-1, -1, -1, 2, 2, 2, 2, -1, 1, 0},
			{-1, -1, -1, 2, 2, 2, -1, 1, 1, 0},
			{-1, -1, -1, 2, 2, 2, -1, 1, 1, 0},
			{-1, -1, -1, 2, 2, 2, -1, 1, 1, 0},
			{-1, -1, -1, 2, 2, 2, -1, 1, 1, 0},
			{-1, -1, -1, 2, 2, 2, -1, 1, 1, 0},
			{-1, -1, -1, 2, 2, 2, -1, 1, 1, 0},
		},
	},
	{
		Description: "Test Case 4: Three snakes on a small grid",
		Width:       6,
		Height:      6,
		Snakes: []Point{
			{X: 4, Y: 2},
			{X: 4, Y: 4},
			{X: 1, Y: 5},
		},
		Expected: [][]int{
			{-1, -1, 0, 0, 0, 0},
			{-1, -1, 0, 0, 0, 0},
			{-1, -1, 0, 0, 0, 0},
			{2, 2, -1, -1, -1, -1},
			{2, 2, -1, 1, 1, 1},
			{2, 2, 2, -1, 1, 1},
		},
	},
	{
		Description: "Test Case 5: Large grid with four snakes",
		Width:       8,
		Height:      8,
		Snakes: []Point{
			{X: 4, Y: 5},
			{X: 4, Y: 6},
			{X: 5, Y: 2},
			{X: 2, Y: 0},
		},
		Expected: [][]int{
			{3, 3, 3, 3, 3, 2, 2, 2},
			{3, 3, 3, 3, 2, 2, 2, 2},
			{3, 3, 3, 2, 2, 2, 2, 2},
			{3, 3, 3, -1, -1, 2, 2, 2},
			{0, 0, 0, 0, 0, -1, -1, -1},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{1, 1, 1, 1, 1, 1, 1, 1},
			{1, 1, 1, 1, 1, 1, 1, 1},
		},
	},
	{
		Description: "Test Case 6: Large grid with evenly spaced snakes",
		Width:       10,
		Height:      10,
		Snakes: []Point{
			{X: 1, Y: 1},
			{X: 8, Y: 1},
			{X: 1, Y: 8},
			{X: 8, Y: 8},
		},
		Expected: [][]int{
			{0, 0, 0, 0, 0, 1, 1, 1, 1, 1},
			{0, 0, 0, 0, 0, 1, 1, 1, 1, 1},
			{0, 0, 0, 0, 0, 1, 1, 1, 1, 1},
			{0, 0, 0, 0, 0, 1, 1, 1, 1, 1},
			{0, 0, 0, 0, 0, 1, 1, 1, 1, 1},
			{2, 2, 2, 2, 2, 3, 3, 3, 3, 3},
			{2, 2, 2, 2, 2, 3, 3, 3, 3, 3},
			{2, 2, 2, 2, 2, 3, 3, 3, 3, 3},
			{2, 2, 2, 2, 2, 3, 3, 3, 3, 3},
			{2, 2, 2, 2, 2, 3, 3, 3, 3, 3},
		},
	},
	{
		Description: "Test Case 7: Small grid with close snakes",
		Width:       3,
		Height:      3,
		Snakes: []Point{
			{X: 0, Y: 0},
			{X: 2, Y: 2},
		},
		Expected: [][]int{
			{0, 0, -1},
			{0, -1, 1},
			{-1, 1, 1},
		},
	},
	{
		Description: "Test Case 8: Single snake in the middle of a grid",
		Width:       7,
		Height:      7,
		Snakes: []Point{
			{X: 3, Y: 3},
		},
		Expected: [][]int{
			{0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0},
		},
	},
	{
		Description: "Test Case 9: Diagonal snakes",
		Width:       5,
		Height:      5,
		Snakes: []Point{
			{X: 0, Y: 0},
			{X: 4, Y: 4},
		},
		Expected: [][]int{
			{0, 0, 0, 0, -1},
			{0, 0, 0, -1, 1},
			{0, 0, -1, 1, 1},
			{0, -1, 1, 1, 1},
			{-1, 1, 1, 1, 1},
		},
	},
	{
		Description: "Test Case 10: Large grid with clustered snakes",
		Width:       12,
		Height:      12,
		Snakes: []Point{
			{X: 5, Y: 5},
			{X: 6, Y: 6},
			{X: 5, Y: 6},
			{X: 6, Y: 5},
		},
		Expected: [][]int{
			{0, 0, 0, 0, 0, 0, 3, 3, 3, 3, 3, 3},
			{0, 0, 0, 0, 0, 0, 3, 3, 3, 3, 3, 3},
			{0, 0, 0, 0, 0, 0, 3, 3, 3, 3, 3, 3},
			{0, 0, 0, 0, 0, 0, 3, 3, 3, 3, 3, 3},
			{0, 0, 0, 0, 0, 0, 3, 3, 3, 3, 3, 3},
			{0, 0, 0, 0, 0, 0, 3, 3, 3, 3, 3, 3},
			{2, 2, 2, 2, 2, 2, 1, 1, 1, 1, 1, 1},
			{2, 2, 2, 2, 2, 2, 1, 1, 1, 1, 1, 1},
			{2, 2, 2, 2, 2, 2, 1, 1, 1, 1, 1, 1},
			{2, 2, 2, 2, 2, 2, 1, 1, 1, 1, 1, 1},
			{2, 2, 2, 2, 2, 2, 1, 1, 1, 1, 1, 1},
			{2, 2, 2, 2, 2, 2, 1, 1, 1, 1, 1, 1},
		},
	},
}

func TestVoronoi(t *testing.T) {
	for _, testCase := range voronoiTestCases {
		t.Run(testCase.Description, func(t *testing.T) {

			board := Board{
				Height: testCase.Height,
				Width:  testCase.Width,
				Snakes: []Snake{},
			}

			for i, snakePos := range testCase.Snakes {
				board.Snakes = append(board.Snakes, Snake{
					ID:   fmt.Sprintf("snake-%d", i+1),
					Head: snakePos,
				})
			}

			voronoi := GenerateVoronoi(board)

			assert.Equal(t, testCase.Expected, voronoi, "Voronoi diagram did not match expected for test case: %+v", testCase)
		})
	}
}

func BenchmarkGenerateVoronoi(b *testing.B) {
	// Set up an 11x11 grid with some snakes
	board := Board{
		Height: 11,
		Width:  11,
		Snakes: []Snake{
			{ID: "snake1", Head: Point{X: 1, Y: 1}},
			{ID: "snake2", Head: Point{X: 9, Y: 1}},
			{ID: "snake3", Head: Point{X: 1, Y: 9}},
			{ID: "snake4", Head: Point{X: 9, Y: 9}},
		},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = GenerateVoronoi(board)
	}
}
