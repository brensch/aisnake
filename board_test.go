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

func TestApplyMove(t *testing.T) {
	testCases := []ApplyMoveTestCase{
		{
			Description: "Single snake moves up and loses health",
			InitialBoard: Board{
				Height: 5, Width: 5,
				Snakes: []Snake{
					{ID: "snake1", Health: 100, Head: Point{X: 2, Y: 2}, Body: []Point{{X: 2, Y: 2}, {X: 2, Y: 1}}},
				},
			},
			Move:       Up,
			SnakeIndex: 0,
			ExpectedBoard: Board{
				Height: 5, Width: 5,
				Snakes: []Snake{
					{ID: "snake1", Health: 99, Head: Point{X: 2, Y: 3}, Body: []Point{{X: 2, Y: 3}, {X: 2, Y: 2}}},
				},
			},
		},
		{
			Description: "Single snake eats food, grows, and restores health",
			InitialBoard: Board{
				Height: 5, Width: 5,
				Food: []Point{{X: 2, Y: 3}},
				Snakes: []Snake{
					{ID: "snake1", Health: 98, Head: Point{X: 2, Y: 2}, Body: []Point{{X: 2, Y: 2}, {X: 2, Y: 1}}},
				},
			},
			Move:       Up,
			SnakeIndex: 0,
			ExpectedBoard: Board{
				Height: 5, Width: 5,
				Food: []Point{}, // Food is consumed
				Snakes: []Snake{
					{ID: "snake1", Health: 100, Head: Point{X: 2, Y: 3}, Body: []Point{{X: 2, Y: 3}, {X: 2, Y: 2}, {X: 2, Y: 1}}},
				},
			},
		},
		{
			Description: "Snake runs into wall and dies",
			InitialBoard: Board{
				Height: 5, Width: 5,
				Snakes: []Snake{
					{ID: "snake1", Health: 100, Head: Point{X: 4, Y: 4}, Body: []Point{{X: 4, Y: 4}, {X: 3, Y: 4}}},
				},
			},
			Move:       Right,
			SnakeIndex: 0,
			ExpectedBoard: Board{
				Height: 5, Width: 5,
				Snakes: []Snake{}, // Snake dies
			},
		},
		{
			Description: "Two snakes collide head-to-head, longer one survives",
			InitialBoard: Board{
				Height: 5, Width: 5,
				Snakes: []Snake{
					{ID: "snake1", Health: 100, Head: Point{X: 2, Y: 2}, Body: []Point{{X: 2, Y: 2}, {X: 1, Y: 2}, {X: 0, Y: 2}}},
					{ID: "snake2", Health: 100, Head: Point{X: 3, Y: 2}, Body: []Point{{X: 3, Y: 2}, {X: 4, Y: 2}}},
				},
			},
			Move:       Right,
			SnakeIndex: 0,
			ExpectedBoard: Board{
				Height: 5, Width: 5,
				Snakes: []Snake{
					{ID: "snake1", Health: 99, Head: Point{X: 3, Y: 2}, Body: []Point{{X: 3, Y: 2}, {X: 2, Y: 2}, {X: 1, Y: 2}}},
					// snake2 should die
				},
			},
		},
		{
			Description: "Two snakes collide heads at 90 degrees, only one snake moves",
			InitialBoard: Board{
				Height: 5, Width: 5,
				Snakes: []Snake{
					{ID: "snake1", Health: 100, Head: Point{X: 2, Y: 2}, Body: []Point{{X: 2, Y: 2}, {X: 2, Y: 1}, {X: 2, Y: 0}}},
					{ID: "snake2", Health: 100, Head: Point{X: 3, Y: 3}, Body: []Point{{X: 3, Y: 3}, {X: 3, Y: 4}}},
				},
			},
			Move:       Right,
			SnakeIndex: 0,
			ExpectedBoard: Board{
				Height: 5, Width: 5,
				Snakes: []Snake{
					// snake1 should survive because it's longer
					{ID: "snake1", Health: 99, Head: Point{X: 3, Y: 2}, Body: []Point{{X: 3, Y: 2}, {X: 2, Y: 2}, {X: 2, Y: 1}}},
					// snake2 remains, as it has not moved yet
					{ID: "snake2", Health: 100, Head: Point{X: 3, Y: 3}, Body: []Point{{X: 3, Y: 3}, {X: 3, Y: 4}}},
				},
			},
		},
		{
			Description: "Move causes snake to collide with another snake's head (even length)",
			InitialBoard: Board{
				Height: 5, Width: 5,
				Snakes: []Snake{
					{ID: "snake1", Health: 100, Head: Point{X: 2, Y: 2}, Body: []Point{{X: 2, Y: 2}, {X: 2, Y: 1}}},
					{ID: "snake2", Health: 100, Head: Point{X: 3, Y: 2}, Body: []Point{{X: 3, Y: 2}, {X: 3, Y: 3}}},
				},
			},
			Move:       Right,
			SnakeIndex: 0,
			ExpectedBoard: Board{
				Height: 5, Width: 5,
				Snakes: []Snake{
					// snake1 and snake2 should both be removed because they are of the same length
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Description, func(t *testing.T) {
			newBoard := copyBoard(tc.InitialBoard)
			applyMove(&newBoard, tc.SnakeIndex, tc.Move)
			assert.Equal(t, tc.ExpectedBoard, newBoard, "The resulting board state does not match the expected board state")

			fmt.Println("original")
			fmt.Println(visualizeBoard(tc.InitialBoard))
			fmt.Println("expected")
			fmt.Println(visualizeBoard(tc.ExpectedBoard))
			fmt.Println("actual")
			fmt.Println(visualizeBoard(newBoard))
		})
	}
}
