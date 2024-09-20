package main

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type VoronoiTestCase struct {
	Board       string
	Expected    [][]int
	Description string
}

var voronoiTestCases = []VoronoiTestCase{
	// {
	// 	Description: "Test Case 1: Snakes with the same position",
	// 	Width:       5,
	// 	Height:      5,
	// 	Snakes: []Point{
	// 		{X: 3, Y: 1},
	// 		{X: 3, Y: 1},
	// 	},
	// 	Expected: [][]int{
	// 		{-1, -1, -1, -1, -1},
	// 		{-1, -1, -1, -1, -1},
	// 		{-1, -1, -1, -1, -1},
	// 		{-1, -1, -1, -1, -1},
	// 		{-1, -1, -1, -1, -1},
	// 	},
	// },
	// {
	// 	Description: "Test Case 2: Multiple snakes with different positions",
	// 	Width:       7,
	// 	Height:      7,
	// 	Snakes: []Point{
	// 		{X: 4, Y: 0},
	// 		{X: 3, Y: 4},
	// 		{X: 5, Y: 5},
	// 	},
	// 	Expected: [][]int{
	// 		{0, 0, 0, 0, 0, 0, 0},
	// 		{0, 0, 0, 0, 0, 0, 0},
	// 		{1, 1, 1, 1, 0, -1, -1},
	// 		{1, 1, 1, 1, 1, 2, 2},
	// 		{1, 1, 1, 1, 1, 2, 2},
	// 		{1, 1, 1, 1, 2, 2, 2},
	// 		{1, 1, 1, 1, 2, 2, 2},
	// 	},
	// },
	// {
	// 	Description: "Test Case 3: Large grid with multiple snakes",
	// 	Width:       10,
	// 	Height:      10,
	// 	Snakes: []Point{
	// 		{X: 9, Y: 4},
	// 		{X: 8, Y: 4},
	// 		{X: 5, Y: 3},
	// 		{X: 2, Y: 0},
	// 	},
	// 	Expected: [][]int{
	// 		{3, 3, 3, 3, 3, -1, -1, -1, 1, 0},
	// 		{3, 3, 3, 3, -1, 2, 2, -1, 1, 0},
	// 		{3, 3, 3, -1, 2, 2, 2, -1, 1, 0},
	// 		{-1, -1, -1, 2, 2, 2, 2, -1, 1, 0},
	// 		{-1, -1, -1, 2, 2, 2, -1, 1, 1, 0},
	// 		{-1, -1, -1, 2, 2, 2, -1, 1, 1, 0},
	// 		{-1, -1, -1, 2, 2, 2, -1, 1, 1, 0},
	// 		{-1, -1, -1, 2, 2, 2, -1, 1, 1, 0},
	// 		{-1, -1, -1, 2, 2, 2, -1, 1, 1, 0},
	// 		{-1, -1, -1, 2, 2, 2, -1, 1, 1, 0},
	// 	},
	// },
	// {
	// 	Description: "Test Case 4: Three snakes on a small grid",
	// 	Width:       6,
	// 	Height:      6,
	// 	Snakes: []Point{
	// 		{X: 4, Y: 2},
	// 		{X: 4, Y: 4},
	// 		{X: 1, Y: 5},
	// 	},
	// 	Expected: [][]int{
	// 		{-1, -1, 0, 0, 0, 0},
	// 		{-1, -1, 0, 0, 0, 0},
	// 		{-1, -1, 0, 0, 0, 0},
	// 		{2, 2, -1, -1, -1, -1},
	// 		{2, 2, -1, 1, 1, 1},
	// 		{2, 2, 2, -1, 1, 1},
	// 	},
	// },
	// {
	// 	Description: "Test Case 5: Large grid with four snakes",
	// 	Width:       8,
	// 	Height:      8,
	// 	Snakes: []Point{
	// 		{X: 4, Y: 5},
	// 		{X: 4, Y: 6},
	// 		{X: 5, Y: 2},
	// 		{X: 2, Y: 0},
	// 	},
	// 	Expected: [][]int{
	// 		{3, 3, 3, 3, 3, 2, 2, 2},
	// 		{3, 3, 3, 3, 2, 2, 2, 2},
	// 		{3, 3, 3, 2, 2, 2, 2, 2},
	// 		{3, 3, 3, -1, -1, 2, 2, 2},
	// 		{0, 0, 0, 0, 0, -1, -1, -1},
	// 		{0, 0, 0, 0, 0, 0, 0, 0},
	// 		{1, 1, 1, 1, 1, 1, 1, 1},
	// 		{1, 1, 1, 1, 1, 1, 1, 1},
	// 	},
	// },
	// {
	// 	Description: "Test Case 6: Large grid with evenly spaced snakes",
	// 	Width:       10,
	// 	Height:      10,
	// 	Snakes: []Point{
	// 		{X: 1, Y: 1},
	// 		{X: 8, Y: 1},
	// 		{X: 1, Y: 8},
	// 		{X: 8, Y: 8},
	// 	},
	// 	Expected: [][]int{
	// 		{0, 0, 0, 0, 0, 1, 1, 1, 1, 1},
	// 		{0, 0, 0, 0, 0, 1, 1, 1, 1, 1},
	// 		{0, 0, 0, 0, 0, 1, 1, 1, 1, 1},
	// 		{0, 0, 0, 0, 0, 1, 1, 1, 1, 1},
	// 		{0, 0, 0, 0, 0, 1, 1, 1, 1, 1},
	// 		{2, 2, 2, 2, 2, 3, 3, 3, 3, 3},
	// 		{2, 2, 2, 2, 2, 3, 3, 3, 3, 3},
	// 		{2, 2, 2, 2, 2, 3, 3, 3, 3, 3},
	// 		{2, 2, 2, 2, 2, 3, 3, 3, 3, 3},
	// 		{2, 2, 2, 2, 2, 3, 3, 3, 3, 3},
	// 	},
	// },
	// {
	// 	Description: "Test Case 7: Small grid with close snakes",
	// 	Width:       3,
	// 	Height:      3,
	// 	Snakes: []Point{
	// 		{X: 0, Y: 0},
	// 		{X: 2, Y: 2},
	// 	},
	// 	Expected: [][]int{
	// 		{0, 0, -1},
	// 		{0, -1, 1},
	// 		{-1, 1, 1},
	// 	},
	// },
	// {
	// 	Description: "Test Case 8: Single snake in the middle of a grid",
	// 	Width:       7,
	// 	Height:      7,
	// 	Snakes: []Point{
	// 		{X: 3, Y: 3},
	// 	},
	// 	Expected: [][]int{
	// 		{0, 0, 0, 0, 0, 0, 0},
	// 		{0, 0, 0, 0, 0, 0, 0},
	// 		{0, 0, 0, 0, 0, 0, 0},
	// 		{0, 0, 0, 0, 0, 0, 0},
	// 		{0, 0, 0, 0, 0, 0, 0},
	// 		{0, 0, 0, 0, 0, 0, 0},
	// 		{0, 0, 0, 0, 0, 0, 0},
	// 	},
	// },
	// {
	// 	Description: "Test Case 9: Diagonal snakes",
	// 	Width:       5,
	// 	Height:      5,
	// 	Snakes: []Point{
	// 		{X: 0, Y: 0},
	// 		{X: 4, Y: 4},
	// 	},
	// 	Expected: [][]int{
	// 		{0, 0, 0, 0, -1},
	// 		{0, 0, 0, -1, 1},
	// 		{0, 0, -1, 1, 1},
	// 		{0, -1, 1, 1, 1},
	// 		{-1, 1, 1, 1, 1},
	// 	},
	// },
	{
		Description: "Test Case 10: Large grid with clustered snakes",
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
		Board: `{"height":11,"width":11,"food":[{"x":5,"y":5}],"hazards":null,"snakes":[{"id":"9e49d817-9ae9-4cbc-8d70-c2e93912dd54","name":"mcts","health":90,"body":[{"x":7,"y":5},{"x":6,"y":5},{"x":6,"y":4},{"x":5,"y":4}],"latency":"396","head":{"x":7,"y":5},"shout":"","customizations":{"color":"#00ff00","head":"replit-mark","tail":"replit-notmark"}},{"id":"2ab74633-92cc-4ee2-8a26-0c92118bf1fc","name":"me2","health":90,"body":[{"x":5,"y":7},{"x":5,"y":6},{"x":4,"y":6},{"x":4,"y":5}],"latency":"396","head":{"x":5,"y":7},"shout":"","customizations":{"color":"#00ff00","head":"replit-mark","tail":"replit-notmark"}}]}`,
	},
}

func TestVoronoi(t *testing.T) {
	for _, testCase := range voronoiTestCases {
		t.Run(testCase.Description, func(t *testing.T) {

			var board Board
			err := json.Unmarshal([]byte(testCase.Board), &board)
			assert.NoError(t, err)

			paths, _ := GenerateVoronoi(board)

			fmt.Println(VisualizeVoronoi(resolveOwnership(paths), board.Snakes))
			fmt.Println(visualizeBoard(board))

			// assert.Equal(t, testCase.Expected, voronoi, "Voronoi diagram did not match expected for test case: %+v", testCase)
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
		_, _ = GenerateVoronoi(board)
	}
}

// func TestIsLegalMove(t *testing.T) {
// 	testCases := []struct {
// 		Description  string
// 		InitialBoard string
// 		SnakeIndex   int
// 		NewHead      Point
// 		Expected     bool
// 	}{
// 		{
// 			Description:  "should be a legal move within the board boundaries and no collisions",
// 			InitialBoard: `{"height":11,"width":11,"food":[{"x":0,"y":2},{"x":0,"y":4},{"x":7,"y":0},{"x":6,"y":10}],"hazards":null,"snakes":[{"id":"8d1de07d-92cf-4ac9-a23e-45aeb8bc14c1","name":"mcts","health":58,"body":[{"x":9,"y":1},{"x":10,"y":1},{"x":10,"y":2},{"x":9,"y":2},{"x":9,"y":3}],"latency":"406","head":{"x":9,"y":1},"shout":"","customizations":{"color":"#888888","head":"default","tail":"default"}},{"id":"a6afe25e-c5fc-450a-b9f1-40f638fe8be0","name":"soba","health":87,"body":[{"x":10,"y":3},{"x":10,"y":4},{"x":10,"y":5},{"x":10,"y":6},{"x":9,"y":6},{"x":8,"y":6}],"latency":"401","head":{"x":10,"y":3},"shout":"","customizations":{"color":"#118645","head":"replit-mark","tail":"replit-notmark"}}]}`,
// 			SnakeIndex:   1,
// 			NewHead:      Point{X: 9, Y: 3},
// 			Expected:     false,
// 		},
// 	}

// 	for _, tc := range testCases {
// 		t.Run(tc.Description, func(t *testing.T) {
// 			var board Board
// 			assert.NoError(t, json.Unmarshal([]byte(tc.InitialBoard), &board))

// 			result := isLegalMove(board, tc.SnakeIndex, tc.NewHead)
// 			assert.Equal(t, tc.Expected, result)
// 		})
// 	}
// }

func TestEvaluate(t *testing.T) {
	testCases := []struct {
		Description  string
		InitialBoard string
		SnakeIndex   int
	}{
		// {
		// 	Description:  "should score win as 1",
		// 	InitialBoard: `{"height":11,"width":11,"food":[{"x":0,"y":2},{"x":0,"y":4},{"x":7,"y":0},{"x":6,"y":10}],"hazards":null,"snakes":[{"id":"8d1de07d-92cf-4ac9-a23e-45aeb8bc14c1","name":"mcts","health":0,"body":[],"latency":"406","head":{"x":10,"y":5},"shout":"","customizations":{"color":"#888888","head":"default","tail":"default"}},{"id":"a6afe25e-c5fc-450a-b9f1-40f638fe8be0","name":"soba","health":86,"body":[{"x":7,"y":5},{"x":8,"y":5},{"x":9,"y":5},{"x":10,"y":5},{"x":10,"y":6},{"x":9,"y":6}],"latency":"401","head":{"x":7,"y":5},"shout":"","customizations":{"color":"#118645","head":"replit-mark","tail":"replit-notmark"}}]}`,
		// 	SnakeIndex:   1,
		// },
		// {
		// 	Description:  "should score loss as negative",
		// 	InitialBoard: `{"height":11,"width":11,"food":[{"x":10,"y":0},{"x":10,"y":3},{"x":8,"y":1},{"x":9,"y":0},{"x":3,"y":1},{"x":4,"y":2},{"x":8,"y":4},{"x":3,"y":0},{"x":9,"y":5}],"hazards":null,"snakes":[{"id":"bbc27600-9763-4cce-954a-b3d6fa0d58de","name":"mcts","health":0,"body":[],"latency":"451","head":{"x":3,"y":9},"shout":"","customizations":{"color":"#888888","head":"default","tail":"default"}},{"id":"a34717ee-ee2f-472e-ba78-a99e446a310a","name":"soba","health":91,"body":[{"x":4,"y":7},{"x":5,"y":7},{"x":6,"y":7},{"x":7,"y":7},{"x":8,"y":7},{"x":9,"y":7},{"x":10,"y":7},{"x":10,"y":8},{"x":10,"y":9},{"x":10,"y":10},{"x":9,"y":10},{"x":9,"y":9},{"x":9,"y":8},{"x":8,"y":8},{"x":7,"y":8},{"x":6,"y":8},{"x":6,"y":9},{"x":5,"y":9},{"x":5,"y":8},{"x":4,"y":8},{"x":4,"y":9},{"x":3,"y":9}],"latency":"401","head":{"x":4,"y":7},"shout":"","customizations":{"color":"#118645","head":"replit-mark","tail":"replit-notmark"}}]}`,
		// 	SnakeIndex:   0,
		// },
		// {
		// 	Description:  "should score loss as negative",
		// 	InitialBoard: `{"height":11,"width":11,"food":[{"x":1,"y":7},{"x":5,"y":4},{"x":6,"y":6},{"x":1,"y":4},{"x":4,"y":2},{"x":5,"y":5},{"x":9,"y":10},{"x":9,"y":9},{"x":9,"y":8}],"hazards":null,"snakes":[{"id":"gs_dytFDvX4qKGTytgV9yRctBH9","name":"Gregory Megory","health":100,"body":[{"x":6,"y":8},{"x":6,"y":9},{"x":6,"y":10},{"x":5,"y":10},{"x":4,"y":10},{"x":3,"y":10},{"x":2,"y":10},{"x":2,"y":9},{"x":1,"y":9},{"x":1,"y":10},{"x":0,"y":10},{"x":0,"y":9},{"x":0,"y":8},{"x":0,"y":7},{"x":0,"y":6},{"x":0,"y":5},{"x":0,"y":4},{"x":0,"y":3},{"x":0,"y":2},{"x":0,"y":1},{"x":1,"y":1},{"x":1,"y":0},{"x":2,"y":0},{"x":2,"y":1},{"x":2,"y":1}],"latency":"413","head":{"x":6,"y":8},"shout":"This is a nice move.","customizations":{"color":"","head":"","tail":""}},{"id":"gs_bH8QtHgCxFdD3cgdPRy8MxfS","name":"Gregory-Degory","health":98,"body":[{"x":7,"y":6},{"x":7,"y":5},{"x":7,"y":4},{"x":8,"y":4},{"x":9,"y":4},{"x":9,"y":3},{"x":9,"y":2},{"x":10,"y":2},{"x":10,"y":1},{"x":10,"y":0},{"x":9,"y":0},{"x":8,"y":0},{"x":7,"y":0},{"x":7,"y":1},{"x":6,"y":1},{"x":6,"y":2},{"x":5,"y":2},{"x":5,"y":1},{"x":4,"y":1},{"x":4,"y":0},{"x":3,"y":0},{"x":3,"y":1},{"x":3,"y":2},{"x":3,"y":3},{"x":3,"y":4},{"x":3,"y":5},{"x":3,"y":6},{"x":3,"y":7},{"x":4,"y":7},{"x":5,"y":7}],"latency":"416","head":{"x":7,"y":6},"shout":"This is a nice move.","customizations":{"color":"","head":"","tail":""}}]}`,
		// 	SnakeIndex:   0,
		// },

		// {
		// 	Description:  "trapped",
		// 	InitialBoard: `{"height":11,"width":11,"food":[{"x":1,"y":7},{"x":5,"y":4},{"x":6,"y":6},{"x":1,"y":4},{"x":4,"y":2},{"x":5,"y":5},{"x":9,"y":10},{"x":9,"y":8}],"hazards":null,"snakes":[{"id":"gs_dytFDvX4qKGTytgV9yRctBH9","name":"Gregory Megory","health":100,"body":[{"x":9,"y":9},{"x":8,"y":9},{"x":7,"y":9},{"x":6,"y":9},{"x":6,"y":10},{"x":5,"y":10},{"x":4,"y":10},{"x":3,"y":10},{"x":2,"y":10},{"x":2,"y":9},{"x":1,"y":9},{"x":1,"y":10},{"x":0,"y":10},{"x":0,"y":9},{"x":0,"y":8},{"x":0,"y":7},{"x":0,"y":6},{"x":0,"y":5},{"x":0,"y":4},{"x":0,"y":3},{"x":0,"y":2},{"x":0,"y":1},{"x":1,"y":1},{"x":1,"y":0},{"x":1,"y":0}],"latency":"413","head":{"x":9,"y":9},"shout":"This is a nice move.","customizations":{"color":"","head":"","tail":""}},{"id":"gs_bH8QtHgCxFdD3cgdPRy8MxfS","name":"Gregory-Degory","health":100,"body":[{"x":6,"y":8},{"x":7,"y":8},{"x":7,"y":7},{"x":7,"y":6},{"x":7,"y":5},{"x":7,"y":4},{"x":8,"y":4},{"x":9,"y":4},{"x":9,"y":3},{"x":9,"y":2},{"x":10,"y":2},{"x":10,"y":1},{"x":10,"y":0},{"x":9,"y":0},{"x":8,"y":0},{"x":7,"y":0},{"x":7,"y":1},{"x":6,"y":1},{"x":6,"y":2},{"x":5,"y":2},{"x":5,"y":1},{"x":4,"y":1},{"x":4,"y":0},{"x":3,"y":0},{"x":3,"y":1},{"x":3,"y":2},{"x":3,"y":3},{"x":3,"y":4},{"x":3,"y":5},{"x":3,"y":6},{"x":3,"y":6}],"latency":"416","head":{"x":6,"y":8},"shout":"This is a nice move.","customizations":{"color":"","head":"","tail":""}}]}`,
		// 	SnakeIndex:   0,
		// },
		{
			Description:  "trapped",
			InitialBoard: `{"height":11,"width":11,"food":[{"x":1,"y":7},{"x":6,"y":8},{"x":5,"y":4},{"x":6,"y":6},{"x":1,"y":4},{"x":4,"y":2},{"x":5,"y":5},{"x":9,"y":10},{"x":9,"y":9},{"x":9,"y":8}],"hazards":null,"snakes":[{"id":"gs_dytFDvX4qKGTytgV9yRctBH9","name":"Gregory Megory","health":65,"body":[{"x":8,"y":10},{"x":8,"y":9},{"x":7,"y":9},{"x":6,"y":9},{"x":6,"y":10},{"x":5,"y":10},{"x":4,"y":10},{"x":3,"y":10},{"x":2,"y":10},{"x":2,"y":9},{"x":1,"y":9},{"x":1,"y":10},{"x":0,"y":10},{"x":0,"y":9},{"x":0,"y":8},{"x":0,"y":7},{"x":0,"y":6},{"x":0,"y":5},{"x":0,"y":4},{"x":0,"y":3},{"x":0,"y":2},{"x":0,"y":1},{"x":1,"y":1},{"x":1,"y":0}],"latency":"413","head":{"x":8,"y":10},"shout":"This is a nice move.","customizations":{"color":"","head":"","tail":""}},{"id":"gs_bH8QtHgCxFdD3cgdPRy8MxfS","name":"Gregory-Degory","health":96,"body":[{"x":7,"y":8},{"x":7,"y":7},{"x":7,"y":6},{"x":7,"y":5},{"x":7,"y":4},{"x":8,"y":4},{"x":9,"y":4},{"x":9,"y":3},{"x":9,"y":2},{"x":10,"y":2},{"x":10,"y":1},{"x":10,"y":0},{"x":9,"y":0},{"x":8,"y":0},{"x":7,"y":0},{"x":7,"y":1},{"x":6,"y":1},{"x":6,"y":2},{"x":5,"y":2},{"x":5,"y":1},{"x":4,"y":1},{"x":4,"y":0},{"x":3,"y":0},{"x":3,"y":1},{"x":3,"y":2},{"x":3,"y":3},{"x":3,"y":4},{"x":3,"y":5},{"x":3,"y":6},{"x":3,"y":7}],"latency":"416","head":{"x":7,"y":8},"shout":"This is a nice move.","customizations":{"color":"","head":"","tail":""}}]}`,
			SnakeIndex:   0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Description, func(t *testing.T) {
			var board Board
			assert.NoError(t, json.Unmarshal([]byte(tc.InitialBoard), &board))

			result := evaluateBoard(&Node{Board: board, LuckMatrix: make([]bool, len(board.Snakes))}, modules)
			t.Log(result)
		})
	}
}
