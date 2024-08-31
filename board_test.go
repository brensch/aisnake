package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateAllMoves(t *testing.T) {
	testCases := []struct {
		Description   string
		Board         Board
		ExpectedMoves []Move
	}{
		{
			Description: "No snakes on the board",
			Board: Board{
				Height: 5,
				Width:  5,
				Snakes: []Snake{},
			},
			ExpectedMoves: []Move{},
		},
		{
			Description: "One snake in the middle of the board",
			Board: Board{
				Height: 5,
				Width:  5,
				Snakes: []Snake{
					{ID: "snake1", Head: Point{X: 2, Y: 2}, Body: []Point{{X: 2, Y: 2}}},
				},
			},
			ExpectedMoves: []Move{
				{Up},
				{Down},
				{Left},
				{Right},
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
			ExpectedMoves: []Move{
				{Up},    // Can move up
				{Right}, // Can move right
			},
		},
		{
			Description: "Two snakes on the board with valid moves only",
			Board: Board{
				Height: 5,
				Width:  5,
				Snakes: []Snake{
					{ID: "snake1", Head: Point{X: 1, Y: 1}, Body: []Point{{X: 1, Y: 1}}}, // Near bottom-left corner
					{ID: "snake2", Head: Point{X: 3, Y: 3}, Body: []Point{{X: 3, Y: 3}}}, // Near center
				},
			},
			ExpectedMoves: []Move{
				{Up, Up}, {Up, Down}, {Up, Left}, {Up, Right},
				{Right, Up}, {Right, Down}, {Right, Left}, {Right, Right},
				{Left, Up}, {Left, Down}, {Left, Left}, {Left, Right},
				{Down, Up}, {Down, Down}, {Down, Left}, {Down, Right},
			},
		},
		{
			Description: "Snake on the board edge with other snakes",
			Board: Board{
				Height: 5,
				Width:  5,
				Snakes: []Snake{
					{ID: "snake1", Head: Point{X: 4, Y: 4}, Body: []Point{{X: 4, Y: 4}}}, // Top-right corner
					{ID: "snake2", Head: Point{X: 2, Y: 2}, Body: []Point{{X: 2, Y: 2}}}, // Center
				},
			},
			ExpectedMoves: []Move{
				{Down, Up}, {Down, Down}, {Down, Left}, {Down, Right},
				{Left, Up}, {Left, Down}, {Left, Left}, {Left, Right},
			},
		},
		{
			Description: "Two snakes on the board with corner positions",
			Board: Board{
				Height: 5,
				Width:  5,
				Snakes: []Snake{
					{ID: "snake1", Head: Point{X: 0, Y: 0}, Body: []Point{{X: 0, Y: 0}}}, // Bottom-left corner
					{ID: "snake2", Head: Point{X: 4, Y: 4}, Body: []Point{{X: 4, Y: 4}}}, // Top-right corner
				},
			},
			ExpectedMoves: []Move{
				{Up, Down}, {Up, Left},
				{Right, Down}, {Right, Left},
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
			ExpectedMoves: []Move{
				{Up},    // Can move up
				{Left},  // Can move left
				{Right}, // Can move right
				// {Down} is invalid as it would cause the snake to move back on itself
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
			ExpectedMoves: []Move{
				{Up},    // Can move up
				{Left},  // Can move left
				{Right}, // Can move right
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
			ExpectedMoves: []Move{
				{Up}, // Forced to move up, but it collides with its own body (this represents the only possible move)
			},
		},
		{
			Description: "Snake with no safe moves at top of board",
			Board: Board{
				Height: 5,
				Width:  5,
				Snakes: []Snake{
					{
						ID:   "snake1",
						Head: Point{X: 0, Y: 4},
						Body: []Point{
							{X: 0, Y: 4},
							{X: 0, Y: 3},
							{X: 1, Y: 3},
							{X: 1, Y: 4},
							{X: 2, Y: 4},
						},
					},
				},
			},
			ExpectedMoves: []Move{
				{Up}, // Forced to move up, but it collides with its own body (this represents the only possible move)
			},
		},
		{
			Description: "Chase your tail out",
			Board: Board{
				Height: 5,
				Width:  5,
				Snakes: []Snake{
					{
						ID:   "snake1",
						Head: Point{X: 0, Y: 4},
						Body: []Point{
							{X: 0, Y: 4},
							{X: 0, Y: 3},
							{X: 1, Y: 3},
							{X: 1, Y: 4},
						},
					},
				},
			},
			ExpectedMoves: []Move{
				{Right}, // Forced to move right to get out through its tail
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
							{X: 2, Y: 4},
						},
					},
					{
						ID:   "snake2",
						Head: Point{X: 5, Y: 5},
						Body: []Point{{X: 5, Y: 5}}, // Can move in any direction
					},
				},
			},
			ExpectedMoves: []Move{
				{Up, Up}, {Up, Down}, {Up, Left}, {Up, Right},
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
							{X: 2, Y: 1}, // Body extends up
							{X: 2, Y: 0},
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
			ExpectedMoves: []Move{
				{Up, Up}, {Up, Left}, {Up, Right},
				{Left, Up}, {Left, Left}, {Left, Right},
				{Right, Up}, {Right, Left}, {Right, Right},
				// {Down, _} should not be present since moving Down would make snake1 collide with snake2's body
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Description, func(t *testing.T) {
			moves := generateAllMoves(tc.Board)

			if tc.ExpectedMoves != nil {
				assert.ElementsMatch(t, tc.ExpectedMoves, moves, "Moves do not match expected values")
			}

			t.Log(tc.ExpectedMoves)
			t.Log(moves)

			t.Log("generated")
			for _, move := range moves {
				fmt.Println(visualizeBoard(tc.Board, WithMove(move)))
			}
			t.Log("expected")
			for _, move := range tc.ExpectedMoves {
				fmt.Println(visualizeBoard(tc.Board, WithMove(move)))
			}
		})
	}
}
