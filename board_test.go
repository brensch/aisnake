package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateSafeMoves(t *testing.T) {
	testCases := []struct {
		Description   string
		Board         Board
		SnakeIndex    int
		ExpectedMoves []Direction
	}{
		{
			Description: "One snake in the middle of the board",
			Board: Board{
				Height: 5,
				Width:  5,
				Snakes: []Snake{
					{ID: "snake1", Head: Point{X: 2, Y: 2}, Body: []Point{{X: 2, Y: 2}}},
				},
			},
			SnakeIndex: 0,
			ExpectedMoves: []Direction{
				Up, Down, Left, Right,
			},
		},
		{
			Description: "One snake at the bottom-left corner of the board",
			Board: Board{
				Height: 5,
				Width:  5,
				Snakes: []Snake{
					{ID: "snake1", Head: Point{X: 0, Y: 0}, Body: []Point{{X: 0, Y: 0}}}, // Bottom-left corner
				},
			},
			SnakeIndex: 0,
			ExpectedMoves: []Direction{
				Up,    // Can move up
				Right, // Can move right
			},
		},
		{
			Description: "One multi-length snake in the middle of the board",
			Board: Board{
				Height: 5,
				Width:  5,
				Snakes: []Snake{
					{
						ID:   "snake1",
						Head: Point{X: 2, Y: 2},
						Body: []Point{
							{X: 2, Y: 2},
							{X: 2, Y: 1}, // Neck position, right behind the head
							{X: 2, Y: 0},
						},
					},
				},
			},
			SnakeIndex: 0,
			ExpectedMoves: []Direction{
				Up,    // Can move up
				Left,  // Can move left
				Right, // Can move right
				// {Down} is invalid as it would cause the snake to move back on itself
			},
		},
		{
			Description: "Snake with no safe moves in middle",
			Board: Board{
				Height: 5,
				Width:  5,
				Snakes: []Snake{
					{
						ID:   "snake1",
						Head: Point{X: 2, Y: 2},
						Body: []Point{
							{X: 2, Y: 2},
							{X: 2, Y: 3},
							{X: 3, Y: 3},
							{X: 3, Y: 2},
							{X: 3, Y: 1},
							{X: 2, Y: 1},
							{X: 1, Y: 1},
							{X: 1, Y: 2},
							{X: 1, Y: 3},
						},
					},
				},
			},
			SnakeIndex: 0,
			ExpectedMoves: []Direction{
				Up, // Forced to move up, but it collides with its own body (this represents the only possible move)
			},
		},
		{
			Description: "One snake with no safe moves, one snake with a safe move",
			Board: Board{
				Height: 7,
				Width:  7,
				Snakes: []Snake{
					{
						ID:   "snake1",
						Head: Point{X: 0, Y: 4},
						Body: []Point{
							{X: 0, Y: 4},
							{X: 0, Y: 3},
							{X: 1, Y: 3},
							{X: 1, Y: 4},
							{X: 1, Y: 5},
							{X: 0, Y: 5},
						},
					},
					{
						ID:   "snake2",
						Head: Point{X: 5, Y: 5},
						Body: []Point{{X: 5, Y: 5}}, // Can move in any direction
					},
				},
			},
			SnakeIndex: 0, // Testing moves for snake1
			ExpectedMoves: []Direction{
				Up,
			},
		},
		{
			Description: "Snake avoids moving into another snake's body",
			Board: Board{
				Height: 5,
				Width:  5,
				Snakes: []Snake{
					{
						ID:   "snake1",
						Head: Point{X: 2, Y: 2},
						Body: []Point{
							{X: 2, Y: 2},
							{X: 2, Y: 1}, // Body extends down
						},
					},
					{
						ID:   "snake2",
						Head: Point{X: 3, Y: 3},
						Body: []Point{
							{X: 3, Y: 3},
							{X: 3, Y: 2}, // Body extends left
						},
					},
				},
			},
			SnakeIndex: 0, // Testing moves for snake1
			ExpectedMoves: []Direction{
				Up, Left, Right,
				// Down should not be present since moving Down would make snake1 collide with snake2's body
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Description, func(t *testing.T) {
			moves := generateSafeMoves(tc.Board, tc.SnakeIndex)

			if tc.ExpectedMoves != nil {
				assert.ElementsMatch(t, tc.ExpectedMoves, moves, "Moves do not match expected values")
			}

			t.Log(tc.ExpectedMoves)
			t.Log(moves)

			t.Log("generated")
			for _, move := range moves {
				fmt.Println(visualizeBoard(tc.Board, WithMove(move, tc.SnakeIndex), WithNewlineCharacter("\n")))
			}
			t.Log("expected")
			for _, move := range tc.ExpectedMoves {
				fmt.Println(visualizeBoard(tc.Board, WithMove(move, tc.SnakeIndex), WithNewlineCharacter("\n")))
			}
		})
	}
}
