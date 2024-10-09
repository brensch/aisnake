package main

import (
	"math"
)

// NodeMaxN represents a node in the MaxN game tree.
type NodeMaxN struct {
	Board         Board
	Depth         int
	UtilityVector []float64   // Utility values for each player.
	Children      []*NodeMaxN // Child nodes in the game tree.
	PlayerIndex   int         // The index of the current player.
}

// NewNodeMaxN initializes a new NodeMaxN.
func NewNodeMaxN(board Board, depth int, playerIndex int) *NodeMaxN {
	playerCount := len(board.Snakes)
	return &NodeMaxN{
		Board:         board,
		Depth:         depth,
		UtilityVector: make([]float64, playerCount),
		Children:      []*NodeMaxN{},
		PlayerIndex:   playerIndex,
	}
}

// MaxNSearch performs the MaxN algorithm up to a specified depth.
func MaxNSearch(node *NodeMaxN, depth int) []float64 {
	// fmt.Println("searching", depth)
	// Base case: if the game is over or depth limit reached.
	if isTerminal(node.Board) || depth == 0 {
		node.UtilityVector = evaluateUtilities(node.Board)
		return node.UtilityVector
	}

	// Generate all possible joint moves (combinations of moves by all alive snakes).
	jointMoves := generateJointMoves(node.Board)

	// Initialize the best utility vector.
	bestUtility := make([]float64, len(node.Board.Snakes))
	for i := range bestUtility {
		bestUtility[i] = -math.MaxFloat64
	}

	// For each joint move, recursively evaluate the resulting game state.
	for _, moves := range jointMoves {
		// Apply the joint moves to get a new board state.
		newBoard := copyBoard(node.Board)
		applyJointMoves(&newBoard, moves)

		// Create a child node for the new board state.
		childNode := NewNodeMaxN(newBoard, depth-1, node.PlayerIndex)
		node.Children = append(node.Children, childNode)

		// Recursively perform MaxN search on the child node.
		utilityVector := MaxNSearch(childNode, depth-1)

		// Update the best utility vector for the current player.
		if utilityVector[node.PlayerIndex] > bestUtility[node.PlayerIndex] {
			bestUtility = utilityVector
		}
	}

	node.UtilityVector = bestUtility
	return bestUtility
}

// generateJointMoves generates all possible combinations of moves by all players.
func generateJointMoves(board Board) [][]Direction {
	playerMoves := make([][]Direction, len(board.Snakes))
	for i, snake := range board.Snakes {
		if isSnakeDead(snake) {
			playerMoves[i] = []Direction{Unset} // No moves for dead snakes.
		} else {
			moves := generateSafeMoves(board, i)
			if len(moves) == 0 {
				moves = AllDirections // If no safe moves, consider all directions.
			}
			playerMoves[i] = moves
		}
	}
	return cartesianProduct(playerMoves)
}

// cartesianProduct computes the cartesian product of players' moves.
func cartesianProduct(sets [][]Direction) [][]Direction {
	var result [][]Direction
	cartesianProductRecursive(sets, []Direction{}, &result)
	return result
}

func cartesianProductRecursive(sets [][]Direction, prefix []Direction, result *[][]Direction) {
	if len(sets) == 0 {
		// Make a copy of prefix to avoid overwriting
		combination := make([]Direction, len(prefix))
		copy(combination, prefix)
		*result = append(*result, combination)
		return
	}

	firstSet := sets[0]
	restSets := sets[1:]

	for _, item := range firstSet {
		newPrefix := append(prefix, item)
		cartesianProductRecursive(restSets, newPrefix, result)
	}
}

// applyJointMoves applies all players' moves simultaneously to the board.
func applyJointMoves(board *Board, moves []Direction) {
	// Create a copy of the snakes to hold their new states.
	newSnakes := make([]Snake, len(board.Snakes))
	copy(newSnakes, board.Snakes)

	// First, calculate the new heads for all snakes.
	for i, move := range moves {
		snake := &newSnakes[i]
		if isSnakeDead(*snake) || move == Unset {
			continue // Skip dead snakes or unset moves.
		}
		// Move the snake's head.
		newHead := moveHead(snake.Head, move)
		snake.Body = append([]Point{newHead}, snake.Body...)
		snake.Head = newHead
		snake.Health -= 1 // Reduce health by 1.
	}

	// Handle food consumption and tail movement.
	handleFoodAndTail(board, newSnakes)

	// Resolve collisions.
	resolveCollisionsMaxN(newSnakes)

	// Update the board with the new snakes.
	board.Snakes = newSnakes
}

// handleFoodAndTail handles food consumption and tail movement for all snakes.
func handleFoodAndTail(board *Board, snakes []Snake) {
	// Keep track of food that has been eaten.
	eatenFoodIndices := map[int]bool{}

	for i := range snakes {
		snake := &snakes[i]
		if isSnakeDead(*snake) {
			continue
		}

		ateFood := false
		for j, food := range board.Food {
			if snake.Head == food && !eatenFoodIndices[j] {
				ateFood = true
				eatenFoodIndices[j] = true
				break
			}
		}

		if ateFood {
			snake.Health = 100
			// Do not remove tail segment (snake grows).
		} else {
			// Remove last segment (snake moves without growing).
			snake.Body = snake.Body[:len(snake.Body)-1]
		}
	}

	// Remove eaten food from the board.
	newFood := []Point{}
	for i, food := range board.Food {
		if !eatenFoodIndices[i] {
			newFood = append(newFood, food)
		}
	}
	board.Food = newFood
}

// resolveCollisionsMaxN resolves any collisions after all moves have been applied.
func resolveCollisionsMaxN(snakes []Snake) {
	deadSnakes := map[int]bool{}

	// Map of positions to snakes occupying them.
	positionToSnakes := make(map[Point][]int)

	// Build the position map.
	for i, snake := range snakes {
		if isSnakeDead(snake) {
			continue
		}
		positionToSnakes[snake.Head] = append(positionToSnakes[snake.Head], i)
	}

	// Handle head-to-head collisions.
	for _, snakeIndices := range positionToSnakes {
		if len(snakeIndices) > 1 {
			// Multiple snakes occupy the same position.
			maxLength := 0
			snakesWithMaxLength := []int{}
			for _, idx := range snakeIndices {
				length := len(snakes[idx].Body)
				if length > maxLength {
					maxLength = length
					snakesWithMaxLength = []int{idx}
				} else if length == maxLength {
					snakesWithMaxLength = append(snakesWithMaxLength, idx)
				}
			}
			// Eliminate all snakes except those with maximum length.
			for _, idx := range snakeIndices {
				if !contains(snakesWithMaxLength, idx) {
					deadSnakes[idx] = true
				}
			}
			// If multiple snakes have the same maximum length, they all die.
			if len(snakesWithMaxLength) > 1 {
				for _, idx := range snakesWithMaxLength {
					deadSnakes[idx] = true
				}
			}
		}
	}

	// Handle collisions with other snakes' bodies.
	for i, snake := range snakes {
		if isSnakeDead(snake) {
			continue
		}
		for j, otherSnake := range snakes {
			if i == j || isSnakeDead(otherSnake) {
				continue
			}
			// Skip the first segment (head) for the other snake.
			for _, segment := range otherSnake.Body[1:] {
				if snake.Head == segment {
					deadSnakes[i] = true
					break
				}
			}
			if deadSnakes[i] {
				break
			}
		}
	}

	// Mark dead snakes.
	for idx := range deadSnakes {
		snakes[idx].Body = []Point{}
		snakes[idx].Health = 0
	}
}

// contains checks if a slice contains a specific integer.
func contains(slice []int, item int) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

// evaluateUtilities evaluates the utility vector for all players.
func evaluateUtilities(board Board) []float64 {
	utilities := make([]float64, len(board.Snakes))
	for i := range board.Snakes {
		utilities[i] = evaluateBoard(board, i, modules)
	}
	return utilities
}
