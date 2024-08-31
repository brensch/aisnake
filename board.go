package main

// applyMoves applies the moves directly to the provided board without returning a new board.
func applyMoves(board *Board, moves Move) {
	// First, create a map to track the initial head positions
	initialHeads := make(map[int]Point)
	for i, snake := range board.Snakes {
		initialHeads[i] = snake.Head
	}

	// Track new head positions
	newHeads := make(map[Point][]int)

	// Apply each snake's move and calculate new head positions
	for i := range board.Snakes {
		direction := moves[i]
		newHead := moveHead(board.Snakes[i].Head, direction)
		newHeads[newHead] = append(newHeads[newHead], i)

		// Move the snake's head and body
		snake := &board.Snakes[i]
		snake.Body = append([]Point{newHead}, snake.Body...) // Add new head to the body
		snake.Head = newHead                                 // Update the head position
	}

	// After all moves, decrement health and handle food consumption
	for i := range board.Snakes {
		snake := &board.Snakes[i]
		snake.Health -= 1 // Reduce health by 1

		// Check if the snake eats food
		ateFood := false
		for j, food := range board.Food {
			if snake.Head == food {
				ateFood = true
				board.Food = append(board.Food[:j], board.Food[j+1:]...) // Remove food from the board
				break
			}
		}

		// If the snake ate food, reset health and add back the last tail segment
		if ateFood {
			snake.Health = 100
		} else {
			// If no food was eaten, remove the last segment (shrink the tail)
			snake.Body = snake.Body[:len(snake.Body)-1]
		}
	}

	// Handle head-on collisions and general collisions
	deadSnakes := make(map[int]bool)

	// Resolve head-on collisions (if two snakes move into each other's heads)
	for i := range board.Snakes {
		for j := i + 1; j < len(board.Snakes); j++ {
			if initialHeads[i] == board.Snakes[j].Head && initialHeads[j] == board.Snakes[i].Head {
				// Head-on collision detected between snake i and snake j
				if len(board.Snakes[i].Body) > len(board.Snakes[j].Body) {
					deadSnakes[j] = true // Snake j dies
				} else if len(board.Snakes[i].Body) < len(board.Snakes[j].Body) {
					deadSnakes[i] = true // Snake i dies
				} else {
					// If both are of the same length, both die
					deadSnakes[i] = true
					deadSnakes[j] = true
				}
			}
		}
	}

	// Resolve collisions after moves (multiple snakes moving to the same position)
	for _, indices := range newHeads {
		if len(indices) > 1 {
			// Multiple snakes have moved to the same position
			maxLength := 0
			for _, index := range indices {
				if len(board.Snakes[index].Body) > maxLength {
					maxLength = len(board.Snakes[index].Body)
					continue
				}
				// increase the max so everyone dies if both are at the same length
				if len(board.Snakes[index].Body) == maxLength {
					maxLength = len(board.Snakes[index].Body) + 1
				}
			}
			// Only the longest snake(s) survive; if there's a tie, all snakes die
			for _, index := range indices {
				if len(board.Snakes[index].Body) < maxLength {
					deadSnakes[index] = true // Mark the snake as dead
				}
			}
		}
	}

	// Remove any dead snakes from the board
	liveSnakes := board.Snakes[:0]
	for i, snake := range board.Snakes {
		if !deadSnakes[i] && snake.Head.X >= 0 && snake.Head.X < board.Width && snake.Head.Y >= 0 && snake.Head.Y < board.Height {
			liveSnakes = append(liveSnakes, snake)
		}
	}
	board.Snakes = liveSnakes
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

// // safeMove generates a safe move for a single snake.
// func safeMove(board Board, snake Snake) Direction {
// 	// Collect possible safe directions
// 	var safeDirections []Direction

// 	for _, dir := range AllDirections {
// 		newHead := moveHead(snake.Head, dir)

// 		// Check if the new head position is within the board boundaries
// 		if newHead.X < 0 || newHead.X >= board.Width || newHead.Y < 0 || newHead.Y >= board.Height {
// 			continue
// 		}

// 		// Check if the new head position is on the snake's own body (excluding the tail)
// 		collidesWithSelf := false
// 		for _, part := range snake.Body[:len(snake.Body)-1] { // Exclude tail
// 			if newHead == part {
// 				collidesWithSelf = true
// 				break
// 			}
// 		}
// 		if collidesWithSelf {
// 			continue
// 		}

// 		// Check if the new head position is on any other snake's body
// 		collidesWithOtherSnake := false
// 		for _, otherSnake := range board.Snakes {
// 			for _, part := range otherSnake.Body {
// 				if newHead == part {
// 					collidesWithOtherSnake = true
// 					break
// 				}
// 			}
// 			if collidesWithOtherSnake {
// 				break
// 			}
// 		}
// 		if collidesWithOtherSnake {
// 			continue
// 		}

// 		// If the direction is safe, add it to the list
// 		safeDirections = append(safeDirections, dir)
// 	}

// 	// If there are safe directions, choose one at random
// 	if len(safeDirections) > 0 {
// 		return safeDirections[rand.Intn(len(safeDirections))]
// 	}

// 	// If no safe directions are found, make a random move as a fallback (risky)
// 	return AllDirections[rand.Intn(len(AllDirections))]
// }

// randomMove generates a safe move for all players.
func randomMove(board Board) Move {
	moves := generateAllMoves(board)
	return moves[0]
}

func generateAllMoves(board Board) []Move {
	numSnakes := len(board.Snakes)

	if numSnakes == 0 {
		return nil
	}

	// Step 1: Prepare a modified version of the board where each snake's tail is temporarily removed
	modifiedSnakes := make([]Snake, numSnakes)
	for i, snake := range board.Snakes {
		modifiedSnakes[i] = Snake{
			Head: snake.Head,
			Body: append([]Point{}, snake.Body...),
		}
		if len(modifiedSnakes[i].Body) > 1 {
			modifiedSnakes[i].Body = modifiedSnakes[i].Body[:len(modifiedSnakes[i].Body)-1] // Chop off the tail
		}
	}
	modifiedBoard := Board{
		Width:  board.Width,
		Height: board.Height,
		Snakes: modifiedSnakes,
	}

	// Step 2: Generate the set of safe moves for each snake
	safeMovesPerSnake := make([][]Direction, numSnakes)

	for i := 0; i < numSnakes; i++ {
		snake := board.Snakes[i]
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
			for j := 0; j < numSnakes; j++ {
				otherSnake := modifiedBoard.Snakes[j]

				// Check for collisions with other snakes' bodies
				for _, segment := range otherSnake.Body {
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

		safeMovesPerSnake[i] = safeMoves
	}

	// Step 3: Generate all permutations of the safe moves
	return generatePermutations(safeMovesPerSnake)
}

// Helper function to generate all permutations of moves
func generatePermutations(safeMovesPerSnake [][]Direction) []Move {
	if len(safeMovesPerSnake) == 0 {
		return []Move{}
	}

	var helper func([][]Direction, int, Move, *[]Move)
	helper = func(arrays [][]Direction, depth int, current Move, result *[]Move) {
		if depth == len(arrays) {
			// Copy the current move and append to result
			moveCopy := append(Move(nil), current...)
			*result = append(*result, moveCopy)
			return
		}
		for _, dir := range arrays[depth] {
			helper(arrays, depth+1, append(current, dir), result)
		}
	}

	var result []Move
	helper(safeMovesPerSnake, 0, Move{}, &result)

	return result
}
