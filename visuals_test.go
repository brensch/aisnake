package main

import (
	"fmt"
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

	output := visualizeBoard(game, WithNewlineCharacter("\n"))

	// Define the expected output with the corrected top row
	expectedOutput := `x x x x x x x x x x x x x 
x . . . . . . . . . . . x 
x . . . . . . . . . . . x 
x . . . . . . . . . . . x 
x . . . . . . . . . . . x 
x . . 🍎 . . . . . . . . x 
x . . . . . 🍎 . . . . . x 
x . . . . . B . . . . . x 
x . . . . . b b . . . . x 
x . . . H . . b . . . . x 
x . . . . . . . . . . . x 
x A a a . . . . . . 🍎 . x 
x x x x x x x x x x x x x 
`

	fmt.Println(output)
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
		Board:  board,
		Visits: 10,
		Score:  1.5,
		// UntriedMoves: []Direction{Up},
		SnakeIndex: 0,
		Children: []*Node{
			{
				Board:  board,
				Visits: 5,
				Score:  2.0,
				// UntriedMoves: []Direction{Right},
				SnakeIndex: 0,
				Children: []*Node{
					{
						Board:  board,
						Visits: 3,
						Score:  1.0,
						// UntriedMoves: []Direction{},
						SnakeIndex: 0,
					},
				},
			},
			{
				Board:  board,
				Visits: 8,
				Score:  1.8,
				// UntriedMoves: []Direction{Up},
				SnakeIndex: 0,
			},
		},
	}

	// Call the GenerateMermaidTree function
	output := GenerateMostVisitedPathWithAlternativesMermaidTree(root)
	expectedOutput := `graph TD;
Node_0xc00011e210["Visits: 10<br/>Average Score: 0.15<br/>Untried Moves: 1<br/>a↑<br/>x x x x x <br/>x . . H x <br/>x ↑ 🍎 . x <br/>x A . . x <br/>x x x x x <br/><br/>A A A <br/>A A A <br/>A A A <br/>"]
Node_0xc00011e210 -->|UCB: 1.36| Node_0xc00011e2c0
Node_0xc00011e2c0["Visits: 5<br/>Average Score: 0.40<br/>Untried Moves: 1<br/>a→<br/>x x x x x <br/>x . . H x <br/>x . 🍎 . x <br/>x A → . x <br/>x x x x x <br/><br/>A A A <br/>A A A <br/>A A A <br/>"]
Node_0xc00011e2c0 -->|UCB: 1.37| Node_0xc00011e370
Node_0xc00011e370["Visits: 3<br/>Average Score: 0.33<br/>Untried Moves: 0<br/>A A A <br/>A A A <br/>A A A <br/>"]
Node_0xc00011e210 -->|UCB: 0.98| Node_0xc00011e420
Node_0xc00011e420["Visits: 8<br/>Average Score: 0.23<br/>Untried Moves: 1<br/>a↑<br/>x x x x x <br/>x . . H x <br/>x ↑ 🍎 . x <br/>x A . . x <br/>x x x x x <br/><br/>A A A <br/>A A A <br/>A A A <br/>"]
`

	fmt.Println(output)
	assert.Equal(t, expectedOutput, output)
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
	expectedOutput := `. B B B B 
A . B B B 
A A . B B 
A A A . B 
A A A A . 
`

	// Compare the output with the expected output using testify's assert.Equal
	assert.Equal(t, expectedOutput, output, "The Voronoi visualization output does not match the expected output")
}
