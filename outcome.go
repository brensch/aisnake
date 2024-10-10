package main

import "fmt"

type GameOutcome int

const (
	Win GameOutcome = iota
	Draw
	Loss
)

// describeGameOutcome returns both the enum (GameOutcome) and a descriptive string.
func describeGameOutcome(game BattleSnakeGame) (GameOutcome, string) {
	// Check if you lost by colliding with a wall
	if game.You.Head.X < 0 || game.You.Head.X >= game.Board.Width || game.You.Head.Y < 0 || game.You.Head.Y >= game.Board.Height {
		return Loss, "You crashed into a wall"
	}

	// Check if you lost by colliding with another snake
	for _, snake := range game.Board.Snakes {
		if snake.ID != game.You.ID {
			for _, segment := range snake.Body {
				if game.You.Head == segment {
					return Loss, fmt.Sprintf("You lost by colliding with %s.", snake.Name)
				}
			}
		} else {
			// check for collisions with ourself
			for _, segment := range snake.Body[1 : len(snake.Body)-1] {
				if game.You.Head == segment {
					return Loss, "You ran into yourself"

				}
			}
		}
	}

	// Check if you lost by starving
	if game.You.Health <= 0 {
		return Loss, "You lost by starving to death."
	}

	// Check if all snakes died (a draw)
	livingSnakes := 0
	for _, snake := range game.Board.Snakes {
		if snake.Health > 0 {
			livingSnakes++
		}
	}
	if livingSnakes == 0 {
		return Draw, "All snakes died"
	}

	// Check if you won because all other snakes starved or collided
	if len(game.Board.Snakes) == 1 && game.Board.Snakes[0].ID == game.You.ID {
		// If only your snake remains, it means you won
		return Win, "You won."
	}

	// if we didn't win or draw
	return Loss, "You Lost."
}

func getColorForOutcome(outcome GameOutcome) int {
	switch outcome {
	case Win:
		return 0x00FF00 // Green
	case Draw:
		return 0xFFFF00 // Yellow
	case Loss:
		return 0xFF0000 // Red
	default:
		return 0x0099ff // Default blue color for Discord
	}
}
