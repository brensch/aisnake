package main

import (
	"fmt"
	"strings"
)

func visualizeBoard(game Board) string {
	var sb strings.Builder

	// Create a 2D slice to represent the board
	board := make([][]rune, game.Height)
	for i := range board {
		board[i] = make([]rune, game.Width)
		for j := range board[i] {
			board[i][j] = '.' // Initialize all positions as empty
		}
	}

	// Function to adjust the Y coordinate to match the expected orientation
	adjustY := func(y int) int {
		return game.Height - 1 - y
	}

	// Helper function to check for out-of-bounds errors
	checkOOB := func(x, y int) bool {
		return x >= 0 && x < game.Width && y >= 0 && y < game.Height
	}

	// Place food on the board
	for _, food := range game.Food {
		if checkOOB(food.X, food.Y) {
			board[adjustY(food.Y)][food.X] = 'F'
		} else {
			sb.WriteString(fmt.Sprintf("Food OOB at (%d, %d)\n", food.X, food.Y))
		}
	}

	// Place hazards on the board
	for _, hazard := range game.Hazards {
		if checkOOB(hazard.X, hazard.Y) {
			board[adjustY(hazard.Y)][hazard.X] = 'H'
		} else {
			sb.WriteString(fmt.Sprintf("Hazard OOB at (%d, %d)\n", hazard.X, hazard.Y))
		}
	}

	// Place snakes on the board
	for _, snake := range game.Snakes {
		// Place snake head first to ensure it isn't overwritten
		if checkOOB(snake.Head.X, snake.Head.Y) {
			board[adjustY(snake.Head.Y)][snake.Head.X] = 'S'
		} else {
			sb.WriteString(fmt.Sprintf("Snake head OOB at (%d, %d)\n", snake.Head.X, snake.Head.Y))
		}
		// Place snake body
		for _, part := range snake.Body {
			if part != snake.Head { // Skip the head position
				if checkOOB(part.X, part.Y) {
					board[adjustY(part.Y)][part.X] = 'B'
				} else {
					sb.WriteString(fmt.Sprintf("Snake body OOB at (%d, %d)\n", part.X, part.Y))
				}
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
