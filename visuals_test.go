package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVisualizeBoard(t *testing.T) {
	// Example game state
	game := Board{
		Height: 11,
		Width:  11,
		Food: []Point{
			{X: 5, Y: 5},
			{X: 9, Y: 0},
			{X: 2, Y: 6},
		},
		Hazards: []Point{
			{X: 3, Y: 2},
		},
		Snakes: []Snake{
			{
				ID:     "snake-508e96ac-94ad-11ea-bb37",
				Name:   "My Snake",
				Health: 54,
				Body: []Point{
					{X: 0, Y: 0},
					{X: 1, Y: 0},
					{X: 2, Y: 0},
				},
				Latency: "111",
				Head:    Point{X: 0, Y: 0},
				Shout:   "why are we shouting??",
				Customizations: Customizations{
					Color: "#FF0000",
					Head:  "pixel",
					Tail:  "pixel",
				},
			},
			{
				ID:     "snake-b67f4906-94ae-11ea-bb37",
				Name:   "Another Snake",
				Health: 16,
				Body: []Point{
					{X: 5, Y: 4},
					{X: 5, Y: 3},
					{X: 6, Y: 3},
					{X: 6, Y: 2},
				},
				Latency: "222",
				Head:    Point{X: 5, Y: 4},
				Shout:   "I'm not really sure...",
				Customizations: Customizations{
					Color: "#26CF04",
					Head:  "silly",
					Tail:  "curled",
				},
			},
		},
	}

	// Call visualizeBoard and capture the output
	output := visualizeBoard(game)

	// Define the expected output with the corrected top row
	expectedOutput := `. . . . . . . . . . . 
. . . . . . . . . . . 
. . . . . . . . . . . 
. . . . . . . . . . . 
. . F . . . . . . . . 
. . . . . F . . . . . 
. . . . . S . . . . . 
. . . . . B B . . . . 
. . . H . . B . . . . 
. . . . . . . . . . . 
S B B . . . . . . F . 
`

	// Compare the output with the expected output
	if output != expectedOutput {
		t.Errorf("Expected output:\n%s\nBut got:\n%s", expectedOutput, output)
	}
}

func TestGenerateMermaidTree(t *testing.T) {
	// Define a simple board for the test
	board := Board{
		Height:  3,
		Width:   3,
		Food:    []Point{{X: 1, Y: 1}},
		Hazards: []Point{{X: 2, Y: 2}},
		Snakes: []Snake{
			{
				ID:   "snake1",
				Head: Point{X: 0, Y: 0},
				Body: []Point{{X: 0, Y: 0}},
			},
		},
	}

	// Create a simple tree for the test
	root := &Node{
		Board:        board,
		Visits:       10,
		Score:        1.5,
		UntriedMoves: []Move{{}},
		Children: []*Node{
			{
				Board:  board,
				Visits: 5,
				Score:  2.0,
				Children: []*Node{
					{
						Board:  board,
						Visits: 3,
						Score:  1.0,
					},
				},
			},
			{
				Board:  board,
				Visits: 8,
				Score:  1.8,
			},
		},
	}

	// Call the GenerateMermaidTree function

	output := GenerateMermaidTree(root, 0)
	expectedOutput := `graph TD;
Node_0xc0000a0e60["Visits: 10<br/>Score: 1.50<br/>Untried Moves: 1<br/>. . H <br/>. F . <br/>S . . <br/>"]
Node_0xc0000a0e60 --> Node_0xc0000a0f00
Node_0xc0000a0f00["Visits: 5<br/>Score: 2.00<br/>Untried Moves: 0<br/>. . H <br/>. F . <br/>S . . <br/>"]
Node_0xc0000a0f00 --> Node_0xc0000a0fa0
Node_0xc0000a0fa0["Visits: 3<br/>Score: 1.00<br/>Untried Moves: 0<br/>. . H <br/>. F . <br/>S . . <br/>"]
Node_0xc0000a0e60 --> Node_0xc0000a1040
Node_0xc0000a1040["Visits: 8<br/>Score: 1.80<br/>Untried Moves: 0<br/>. . H <br/>. F . <br/>S . . <br/>"]
`

	assert.Equal(t, output, expectedOutput)
}

func TestVisualizeVoronoi(t *testing.T) {
	// Set up a simple board with two snakes
	board := Board{
		Height: 5,
		Width:  5,
		Snakes: []Snake{
			{ID: "snake1", Head: Point{X: 0, Y: 0}},
			{ID: "snake2", Head: Point{X: 4, Y: 4}},
		},
	}

	// Generate the Voronoi diagram
	voronoi := GenerateVoronoi(board)

	// Generate the visualization
	output := VisualizeVoronoi(voronoi, board.Snakes)

	// Define the expected output
	expectedOutput := `A A A A . 
A A A . B 
A A . B B 
A . B B B 
. B B B B 
`

	// Compare the output with the expected output using testify's assert.Equal
	assert.Equal(t, expectedOutput, output, "The Voronoi visualization output does not match the expected output")
}
