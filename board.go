package main

import "fmt"

// Direction represents possible movement directions for a snake.
type Direction int

const (
	Unset Direction = iota // This represents an unset direction
	Up
	Down
	Left
	Right
	NoMove
)

// AllDirections provides a slice of all possible directions.
var AllDirections = []Direction{Up, Down, Left, Right}

// applyMove applies the move of a single snake directly to the provided board without returning a new board.
func applyMove(board *Board, snakeIndex int, direction Direction) {
	snake := &board.Snakes[snakeIndex]
	if len(snake.Body) == 0 {
		return
	}
	// Track the initial head position of the snake
	initialHead := board.Snakes[snakeIndex].Head

	// Calculate the new head position
	newHead := moveHead(initialHead, direction)

	// Move the snake's head and body
	snake.Body = append([]Point{newHead}, snake.Body...) // Add new head to the body
	snake.Head = newHead                                 // Update the head position

	// Check if the snake went out of bounds
	if !isPointInsideBoard(board, newHead) {
		// Mark the snake as dead
		markDeadSnake(board, snakeIndex)
		return
	}

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

	if len(snake.Body) == 1 {
		fmt.Println("got 0 snake", snakeIndex)
		fmt.Println(visualizeBoard(*board))
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

	// First check if the new head has moved onto any other snake's head
	for i := range board.Snakes {
		if i != snakeIndex && board.Snakes[i].Health > 0 { // Skip dead snakes
			// Check for head-to-head collision
			if newHead == board.Snakes[i].Head {
				// Kill the shorter snake; if equal length, both die
				// we truncated the snake at snakeindex, so add 1 to it
				usLength := len(board.Snakes[snakeIndex].Body) + 1
				if len(board.Snakes[i].Body) == usLength {
					deadSnakes[snakeIndex] = true
					deadSnakes[i] = true
					break
				}
				if len(board.Snakes[i].Body) > usLength {
					deadSnakes[snakeIndex] = true
					break
				}
				deadSnakes[i] = true
			}
		}
	}

	// After head collisions are resolved, check if the new head overlaps any snake's body
	for i := range board.Snakes {
		if board.Snakes[i].Health > 0 { // Skip dead snakes
			// Adjust body length if the other snake has not yet moved
			body := board.Snakes[i].Body
			if i > snakeIndex {
				if len(body) > 0 {
					body = body[:len(body)-1] // Remove last segment (tail)
				}
			}
			for _, segment := range body[1:] { // Exclude the head
				if newHead == segment {
					// If the new head overlaps any body part, kill the snake
					deadSnakes[snakeIndex] = true
					break
				}
			}
		}
	}

	// TODO: this may be needed
	// // New Logic: Check possible moves of snakes that move before us
	// for i := 0; i < snakeIndex; i++ {
	// 	opponent := board.Snakes[i]
	// 	if opponent.Health <= 0 {
	// 		continue // Skip dead snakes
	// 	}

	// 	// Ensure the opponent has at least two segments
	// 	if len(opponent.Body) < 2 {
	// 		continue
	// 	}

	// 	neck := opponent.Body[1]

	// 	// Generate possible positions the opponent could have moved to from their neck
	// 	possiblePositions := []Point{
	// 		{X: neck.X + 1, Y: neck.Y},
	// 		{X: neck.X - 1, Y: neck.Y},
	// 		{X: neck.X, Y: neck.Y + 1},
	// 		{X: neck.X, Y: neck.Y - 1},
	// 	}

	// 	// Check if any of these positions match our head position
	// 	for _, pos := range possiblePositions {
	// 		// Ensure the position is within board boundaries
	// 		if pos.X < 0 || pos.X >= board.Width || pos.Y < 0 || pos.Y >= board.Height {
	// 			continue
	// 		}

	// 		// If the position is our head position
	// 		if pos == newHead {
	// 			// If the opponent is longer or equal in length, we die
	// 			if len(opponent.Body) >= len(board.Snakes[snakeIndex].Body) {
	// 				deadSnakes[snakeIndex] = true
	// 				break
	// 			}
	// 			// If we are longer, the opponent dies (but since they move before us, they have already moved)
	// 			// In this context, we cannot mark them as dead here
	// 		}
	// 	}
	// }

	// Mark dead snakes
	markDeadSnakes(board, deadSnakes)
}

// markDeadSnakes marks snakes as dead by clearing their body and setting health to 0.
func markDeadSnakes(board *Board, deadSnakes map[int]bool) {
	for i := range board.Snakes {
		if deadSnakes[i] {
			board.Snakes[i].Body = []Point{} // Clear the body to mark the snake as dead
			board.Snakes[i].Health = 0       // Set health to 0 to indicate death
		}
	}
}

// markDeadSnake marks a specific snake as dead by clearing its body and setting health to 0.
func markDeadSnake(board *Board, snakeIndex int) {
	board.Snakes[snakeIndex].Body = []Point{} // Clear the body to mark the snake as dead
	board.Snakes[snakeIndex].Health = 0       // Set health to 0 to indicate death
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

// Helper function to get all possible moves a snake can make
func getPossibleMoves(snake Snake) []Point {
	head := snake.Body[0]
	moves := []Point{
		{X: head.X, Y: head.Y + 1}, // Move up
		{X: head.X, Y: head.Y - 1}, // Move down
		{X: head.X + 1, Y: head.Y}, // Move right
		{X: head.X - 1, Y: head.Y}, // Move left
	}
	return moves
}

// Mark danger zones around snakes that are yet to move in this round
// Only snakes after the current snake in the turn order are considered dangerous.
// The dangerZones grid contains the minimum length required to win a head-to-head collision.
func markDangerZones(board *Board, snakeIndex int) [][]int {
	// Initialize the danger zones grid
	dangerZones := make([][]int, board.Height)
	for i := range dangerZones {
		dangerZones[i] = make([]int, board.Width)
	}

	// Mark potential dangerous squares for snakes that have not yet moved in this round
	for i := snakeIndex + 1; i < len(board.Snakes); i++ {
		snake := board.Snakes[i]
		if isSnakeDead(snake) {
			continue
		}
		possibleMoves := getPossibleMoves(snake)
		for _, move := range possibleMoves {
			if isPointInsideBoard(board, move) && !isOccupied(board, move, snakeIndex) {
				// Mark the danger zone with the length of the threatening snake
				dangerZones[move.Y][move.X] = len(snake.Body)
			}
		}
	}
	return dangerZones
}

// Generate safe moves (directions), not counting heads, and ignoring tails of snakes that have moved after it.
// needs to generate move in the board to avoid panics.
func generateSafeMoves(board Board, snakeIndex int) []Direction {
	snake := board.Snakes[snakeIndex]
	if isSnakeDead(snake) {
		return nil
	}

	head := snake.Body[0]
	var neck Point
	// TODO: could remove the length check here since we should be guaranteed nonzero length.
	if len(snake.Body) > 1 {
		neck = snake.Body[1]
	}

	possibleDirections := []Direction{Up, Down, Left, Right}
	safeMoves := []Direction{}
	backupMoves := []Direction{}

	for _, direction := range possibleDirections {
		nextMove := moveInDirection(head, direction)

		// Check if the move is within the board boundaries
		if !isPointInsideBoard(&board, nextMove) {
			continue // Move is out of bounds
		}

		// Check if the move is into the snake's own neck
		if nextMove == neck {
			continue // Move is into the snake's own neck
		}

		// backups are moves that stay inbounds and aren't our neck
		backupMoves = append(backupMoves, direction)

		// don't collide with bodies of other snakes
		foundCollision := false
		for i, snake := range board.Snakes {

			if len(snake.Body) == 0 {
				continue
			}

			lengthToCheck := len(snake.Body)
			if i > snakeIndex {
				lengthToCheck--
			}

			// don't include the head ever. don't include the tail if the snake has not moved yet.
			for _, body := range snake.Body[1:lengthToCheck] {
				if nextMove != body {
					continue
				}
				foundCollision = true
				break
			}
			if foundCollision {
				break
			}
		}
		if foundCollision {
			continue
		}

		// Otherwise, it's a safe move
		safeMoves = append(safeMoves, direction)
	}

	if len(safeMoves) == 0 {
		return backupMoves
	}

	return safeMoves
}

// Check if the point is within the board boundaries
func isPointInsideBoard(board *Board, point Point) bool {
	return point.X >= 0 && point.X < board.Width && point.Y >= 0 && point.Y < board.Height
}

// Check if a point is safe for a given snake to move its head to
func isOccupied(board *Board, point Point, snakeIndex int) bool {
	for i, snake := range board.Snakes {
		snakeLength := len(snake.Body)
		if snakeLength == 0 {
			continue
		}
		body := snake.Body
		if i == snakeIndex {
			// Don't count our own tail since it will disappear if we are the ones moving
			if snakeLength > 1 {
				body = body[:snakeLength-1]
			} else {
				body = []Point{}
			}
		} else if i > snakeIndex {
			// The snake hasn't moved yet; we need to remove its last segment (tail)
			if snakeLength > 1 {
				body = body[:snakeLength-1]
			} else {
				body = []Point{}
			}
		}
		// Now check if point is in the adjusted body
		for _, bodyPart := range body {
			if bodyPart.X == point.X && bodyPart.Y == point.Y {
				return true
			}
		}
	}
	return false
}

// Helper function to map a direction to a point
func moveInDirection(head Point, direction Direction) Point {
	switch direction {
	case Up:
		return Point{X: head.X, Y: head.Y + 1}
	case Down:
		return Point{X: head.X, Y: head.Y - 1}
	case Right:
		return Point{X: head.X + 1, Y: head.Y}
	case Left:
		return Point{X: head.X - 1, Y: head.Y}
	default:
		return head
	}
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
