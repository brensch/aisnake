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
