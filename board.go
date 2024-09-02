package main

import (
	"log"
)

// Direction represents possible movement directions for a snake.
type Direction int

const (
	Unset Direction = iota // This represents an unset direction
	Up
	Down
	Left
	Right
)

// AllDirections provides a slice of all possible directions.
var AllDirections = []Direction{Up, Down, Left, Right}

// applyMove applies the move of a single snake directly to the provided board without returning a new board.
func applyMove(board *Board, snakeIndex int, direction Direction) {
	// Track the initial head position of the snake
	initialHead := board.Snakes[snakeIndex].Head

	// Calculate the new head position
	newHead := moveHead(initialHead, direction)

	// Move the snake's head and body
	snake := &board.Snakes[snakeIndex]
	snake.Body = append([]Point{newHead}, snake.Body...) // Add new head to the body
	snake.Head = newHead                                 // Update the head position

	// Decrement health and handle food consumption
	snake.Health -= 1 // Reduce health by 1

	ateFood := false
	for j, food := range board.Food {
		if snake.Head == food {
			ateFood = true
			board.Food = append(board.Food[:j], board.Food[j+1:]...) // Remove food from the board
			break
		}
	}

	// remove the last segment for the move
	snake.Body = snake.Body[:len(snake.Body)-1]
	// If the snake ate food, reset health and add an additional segment on the tail
	if ateFood {
		snake.Health = 100
		snake.Body = append(snake.Body, snake.Body[len(snake.Body)-1])
	}

	// Handle collisions
	resolveCollisions(board, snakeIndex, newHead)
}

// resolveCollisions handles collisions for the specified snake after it moves.
func resolveCollisions(board *Board, snakeIndex int, newHead Point) {
	deadSnakes := make(map[int]bool)

	// Check if the snake's new head position results in a collision with another snake's head or body
	for i := range board.Snakes {
		if i != snakeIndex {
			// Check for collision with the head of another snake
			if newHead == board.Snakes[i].Head {
				if len(board.Snakes[snakeIndex].Body) > len(board.Snakes[i].Body) {
					deadSnakes[i] = true // The other snake dies
				} else if len(board.Snakes[snakeIndex].Body) < len(board.Snakes[i].Body) {
					deadSnakes[snakeIndex] = true // The current snake dies
				} else {
					// If both are of the same length, both die
					deadSnakes[snakeIndex] = true
					deadSnakes[i] = true
				}
			} else {
				// Check for collision with the body of another snake
				for _, segment := range board.Snakes[i].Body {
					if newHead == segment {
						deadSnakes[snakeIndex] = true // Current snake dies if it collides with another snake's body
					}
				}
			}
		}
	}

	// Remove any dead snakes from the board
	removeDeadSnakes(board, deadSnakes)
}

// removeDeadSnakes removes any snakes marked as dead from the board.
func removeDeadSnakes(board *Board, deadSnakes map[int]bool) {
	liveSnakes := board.Snakes[:0]
	for i, snake := range board.Snakes {
		// Check if the snake is not marked as dead and is within the board boundaries
		if !deadSnakes[i] &&
			snake.Head.X >= 0 && snake.Head.X < board.Width &&
			snake.Head.Y >= 0 && snake.Head.Y < board.Height {
			liveSnakes = append(liveSnakes, snake)
		}
	}
	board.Snakes = liveSnakes
}

// moveHead calculates the new head position based on the direction.
func moveHead(head Point, direction Direction) Point {
	switch direction {
	case Up:
		return Point{X: head.X, Y: head.Y + 1}
	case Down:
		return Point{X: head.X, Y: head.Y - 1}
	case Left:
		return Point{X: head.X - 1, Y: head.Y}
	case Right:
		return Point{X: head.X + 1, Y: head.Y}
	default:
		return head
	}
}

// generateSafeMoves generates the set of safe moves for a given snake.
func generateSafeMoves(board Board, snakeIndex int) []Direction {
	if snakeIndex >= len(board.Snakes) {
		log.Printf("invalid snakeindex: %d. board snake length: %d\n", snakeIndex, len(board.Snakes))
		// fmt.Println(visualizeBoard(board))
		return []Direction{Up}
	}
	snake := board.Snakes[snakeIndex]
	safeMoves := []Direction{}

	for _, direction := range AllDirections {
		newHead := moveHead(snake.Head, direction)

		// Check if the new head is within the board boundaries
		if newHead.X < 0 || newHead.X >= board.Width || newHead.Y < 0 || newHead.Y >= board.Height {
			continue
		}

		// Check if the move causes the snake to move back on itself
		if len(snake.Body) > 1 {
			neck := snake.Body[1] // The segment right after the head
			if newHead == neck {
				continue
			}
		}

		// Check for collisions with other snakes
		collision := false
		for i := range board.Snakes {
			otherSnake := board.Snakes[i]

			// Check for collisions with other snakes' bodies.
			// ensure that we imagine their tail will be disappeared on the next move
			snakeWithoutTail := otherSnake.Body[0 : len(otherSnake.Body)-1]
			for _, segment := range snakeWithoutTail {
				if newHead == segment {
					collision = true
					break
				}
			}

			// Check for head-to-head collisions where the other snake is longer or equal
			if !collision && newHead == otherSnake.Head && len(otherSnake.Body) >= len(snake.Body) {
				collision = true
				break
			}

		}

		// If there's no collision, add the direction to safe moves
		if !collision {
			safeMoves = append(safeMoves, direction)
		}
	}

	// If no safe moves, default to Up
	if len(safeMoves) == 0 {
		safeMoves = append(safeMoves, Up)
	}

	return safeMoves
}

// boardIsTerminal checks if the game has ended.
func boardIsTerminal(board Board) bool {
	// Count the number of alive snakes
	aliveSnakes := 0

	for _, snake := range board.Snakes {
		if snake.Health > 0 && len(snake.Body) > 0 {
			aliveSnakes++
		}
	}

	// The game is terminal if there is only one snake left alive or no snakes are alive
	return aliveSnakes <= 1
}

// copyBoard creates and returns a deep copy of the provided board.
func copyBoard(board Board) Board {
	newBoard := Board{
		Height:  board.Height,
		Width:   board.Width,
		Food:    append([]Point(nil), board.Food...),
		Hazards: append([]Point(nil), board.Hazards...),
		Snakes:  make([]Snake, len(board.Snakes)),
	}

	// Deep copy each snake
	for i, snake := range board.Snakes {
		newSnake := Snake{
			ID:             snake.ID,
			Name:           snake.Name,
			Health:         snake.Health,
			Body:           append([]Point(nil), snake.Body...),
			Latency:        snake.Latency,
			Head:           snake.Head,
			Shout:          snake.Shout,
			Customizations: snake.Customizations,
		}
		newBoard.Snakes[i] = newSnake
	}

	return newBoard
}
