package main

import (
	"testing"
)

func TestVisualizeBoard(t *testing.T) {
	// Example game state
	game := BattleSnakeGame{
		Game: Game{
			ID: "totally-unique-game-id",
			Ruleset: Ruleset{
				Name:    "standard",
				Version: "v1.1.15",
				Settings: Settings{
					FoodSpawnChance:     15,
					MinimumFood:         1,
					HazardDamagePerTurn: 14,
				},
			},
			Map:     "standard",
			Source:  "league",
			Timeout: 500,
		},
		Turn: 14,
		Board: Board{
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
					Length:  3,
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
					Length:  4,
					Shout:   "I'm not really sure...",
					Customizations: Customizations{
						Color: "#26CF04",
						Head:  "silly",
						Tail:  "curled",
					},
				},
			},
		},
		You: Snake{
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
			Length:  3,
			Shout:   "why are we shouting??",
			Customizations: Customizations{
				Color: "#FF0000",
				Head:  "pixel",
				Tail:  "pixel",
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
