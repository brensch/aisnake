package main

import (
	"strings"
)

// visualizeBoard returns a string representation of the game board.
func visualizeBoard(game BattleSnakeGame) string {
	var sb strings.Builder

	// Create a 2D slice to represent the board
	board := make([][]rune, game.Board.Height)
	for i := range board {
		board[i] = make([]rune, game.Board.Width)
		for j := range board[i] {
			board[i][j] = '.' // Initialize all positions as empty
		}
	}

	// Function to adjust the Y coordinate to match the expected orientation
	adjustY := func(y int) int {
		return game.Board.Height - 1 - y
	}

	// Place food on the board
	for _, food := range game.Board.Food {
		board[adjustY(food.Y)][food.X] = 'F'
	}

	// Place hazards on the board
	for _, hazard := range game.Board.Hazards {
		board[adjustY(hazard.Y)][hazard.X] = 'H'
	}

	// Place snakes on the board
	for _, snake := range game.Board.Snakes {
		// Place snake head first to ensure it isn't overwritten
		board[adjustY(snake.Head.Y)][snake.Head.X] = 'S'
		// Place snake body
		for _, part := range snake.Body {
			if part != snake.Head { // Skip the head position
				board[adjustY(part.Y)][part.X] = 'B'
			}
		}
	}

	// Build the string representation of the board
	for _, row := range board {
		for _, cell := range row {
			sb.WriteRune(cell)
			sb.WriteRune(' ')
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
