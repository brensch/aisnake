package main

import (
	"container/heap"
	"math"
)

// EvaluationFunc defines the function signature for evaluation modules.
// It now returns an array of scores for each snake.
type EvaluationFunc func(board Board, context *EvaluationContext) []float64

// EvaluationModule defines a struct that holds an evaluation function and its corresponding weight.
type EvaluationModule struct {
	EvalFunc EvaluationFunc
	Weight   float64
}

var (
	modules = []EvaluationModule{
		{
			EvalFunc: voronoiEvaluation,
			Weight:   6,
		},
		{
			EvalFunc: lengthEvaluation,
			Weight:   6,
		},
	}
)

// EvaluationContext holds precomputed data for evaluation functions to avoid redundant computations.
type EvaluationContext struct {
	Voronoi [][]int
}

// GenerateVoronoi generates a board ownership diagram based on a shortest path algorithm
func GenerateVoronoi(board Board) [][]int {
	// Track the best path (shortest distance) to each position
	bestPaths := make([][]dijkstraNode, board.Height)
	for i := range bestPaths {
		bestPaths[i] = make([]dijkstraNode, board.Width)
		for j := range bestPaths[i] {
			bestPaths[i][j] = dijkstraNode{Point{-1, -1}, -1, math.MaxInt32}
		}
	}

	// Priority queue (min-heap) to process nodes based on distance
	pq := &PriorityQueue{}
	heap.Init(pq)

	// Initialize the priority queue with the heads of all snakes
	for k, snake := range board.Snakes {
		if snake.Health > 0 && len(snake.Body) > 0 { // Skip dead or empty snakes
			head := snake.Head
			heap.Push(pq, dijkstraNode{head, k, 0})
			bestPaths[head.Y][head.X] = dijkstraNode{head, k, 0}
		}
	}

	// Process nodes in the priority queue
	for pq.Len() > 0 {
		node := heap.Pop(pq).(dijkstraNode)
		currentPoint := node.point

		// Get legal moves for the current point
		for _, direction := range AllDirections {
			newPoint := moveHead(currentPoint, direction)

			// Ensure new point is within bounds
			if newPoint.X >= 0 && newPoint.X < board.Width && newPoint.Y >= 0 && newPoint.Y < board.Height {
				// Check if the move is legal for the snake at snakeIndex
				if isLegalMove(board, node.snakeIndex, newPoint, node.distance) {
					// Compute the new distance to reach this point
					newDistance := node.distance + 1

					// Check the best path to this point
					bestNode := bestPaths[newPoint.Y][newPoint.X]
					if newDistance < bestNode.distance {
						// Found a shorter path; update best path
						bestPaths[newPoint.Y][newPoint.X] = dijkstraNode{newPoint, node.snakeIndex, newDistance}
						heap.Push(pq, dijkstraNode{newPoint, node.snakeIndex, newDistance})
					} else if newDistance == bestNode.distance && bestNode.snakeIndex != node.snakeIndex {
						// Tie detected; assign to no one
						bestPaths[newPoint.Y][newPoint.X].snakeIndex = -1
					}
				}
			}
		}
	}

	return dijkstraToResult(bestPaths)
}

// dijkstraNode represents a node in the Dijkstra algorithm
type dijkstraNode struct {
	point      Point
	snakeIndex int
	distance   int // Number of moves from the snake's head
}

// Priority queue for Dijkstra's algorithm
type PriorityQueue []dijkstraNode

// Implement heap.Interface for PriorityQueue
func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// Priority based on distance only
	return pq[i].distance < pq[j].distance
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *PriorityQueue) Push(x interface{}) {
	*pq = append(*pq, x.(dijkstraNode))
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[0 : n-1]
	return item
}

// isLegalMove checks if a move to a new point is legal for the snake.
func isLegalMove(board Board, snakeIndex int, newHead Point, steps int) bool {
	snake := board.Snakes[snakeIndex]

	// Check if the new head is within the board boundaries
	if newHead.X < 0 || newHead.X >= board.Width || newHead.Y < 0 || newHead.Y >= board.Height {
		return false
	}

	// Check for collisions with other snakes
	for i := range board.Snakes {
		otherSnake := board.Snakes[i]

		// Don't consider dead snakes
		if len(otherSnake.Body) == 0 || otherSnake.Health == 0 {
			continue
		}

		// Determine how much of the tail is to be removed based on steps
		stepsToRemove := steps
		if snakeIndex < i {
			stepsToRemove++ // Since the other snake moves after this one, remove one extra
		}

		// Ensure we do not remove more segments than the snake has
		if stepsToRemove < len(otherSnake.Body) {
			otherSnake.Body = otherSnake.Body[0 : len(otherSnake.Body)-stepsToRemove]
		} else {
			// If the steps exceed the length of the snake, treat it as having no body
			otherSnake.Body = []Point{}
		}

		// Check for collisions with the snake's body
		for _, segment := range otherSnake.Body {
			if newHead == segment {
				return false
			}
		}

		// Check for head-to-head collisions where the other snake is longer or equal
		if newHead == otherSnake.Head && len(otherSnake.Body) >= len(snake.Body) {
			return false
		}
	}

	// If no collision, the move is legal
	return true
}

// dijkstraToResult converts the bestPaths grid to a simple snake ownership grid (used for debugging)
func dijkstraToResult(bestPaths [][]dijkstraNode) [][]int {
	result := make([][]int, len(bestPaths))
	for i := range result {
		result[i] = make([]int, len(bestPaths[i]))
		for j := range result[i] {
			result[i][j] = bestPaths[i][j].snakeIndex
		}
	}
	return result
}

// evaluateBoard evaluates the board state and returns an array of scores for each snake.
func evaluateBoard(board Board, modules []EvaluationModule) []float64 {
	numSnakes := len(board.Snakes)
	scores := make([]float64, numSnakes)

	// Create EvaluationContext and precompute data
	context := &EvaluationContext{}
	context.Voronoi = GenerateVoronoi(board)
	// Precompute other data if necessary

	// Calculate the sum of all weights for normalization.
	totalWeight := 0.0
	for _, module := range modules {
		totalWeight += module.Weight
	}

	// For each module, get the scores, apply weight, and accumulate
	for _, module := range modules {
		moduleScores := module.EvalFunc(board, context)
		for i := 0; i < numSnakes; i++ {
			weightedScore := (module.Weight / totalWeight) * moduleScores[i]
			scores[i] += weightedScore
		}
	}

	// calculate snake deaths to save on calcs for all in each loop
	snakeDeaths := make([]bool, numSnakes)
	draw := true
	for i := 0; i < numSnakes; i++ {
		dead := isSnakeDead(board.Snakes[i])
		snakeDeaths[i] = dead
		if !dead {
			draw = false
		}
	}

	// Normalize scores between -1 and 1, handle special cases
	for i := 0; i < numSnakes; i++ {
		// if all snakes are dead, set score to 0 (draw)
		if draw {
			scores[i] = 0
			continue
		}

		// If the snake is dead, set score to minimum
		if snakeDeaths[i] {
			scores[i] = -1
			continue
		}

		// Check if all opponents are dead
		aliveOpponents := 0
		for j := 0; j < numSnakes; j++ {
			if j != i && !isSnakeDead(board.Snakes[j]) {
				aliveOpponents++
			}
		}
		if aliveOpponents == 0 {
			// All opponents are dead, set score to maximum
			scores[i] = 1
			continue
		}

		// Otherwise, normalize score between -1 and 1
		if scores[i] > 1 {
			scores[i] = 1
		} else if scores[i] < -1 {
			scores[i] = -1
		}
	}

	return scores
}

// voronoiEvaluation evaluates the board based on Voronoi control.
func voronoiEvaluation(board Board, context *EvaluationContext) []float64 {
	numSnakes := len(board.Snakes)
	scores := make([]float64, numSnakes)

	// Count the number of cells each snake controls in the Voronoi diagram.
	controlledCells := make([]float64, numSnakes)
	unclaimedCells := 0.0

	for y := 0; y < board.Height; y++ {
		for x := 0; x < board.Width; x++ {
			snakeIndex := context.Voronoi[y][x]
			if snakeIndex >= 0 && snakeIndex < numSnakes {
				controlledCells[snakeIndex]++
			} else {
				// Count unclaimed cells
				unclaimedCells++
			}
		}
	}

	// Compute the score for each snake
	for i := 0; i < numSnakes; i++ {
		if isSnakeDead(board.Snakes[i]) {
			scores[i] = -2
			continue
		}

		opponentsControlledCells := 0.0
		for j := 0; j < numSnakes; j++ {
			if j != i {
				opponentsControlledCells += controlledCells[j]
			}
		}

		// Consider unclaimed cells as neutral
		totalControlled := controlledCells[i] + opponentsControlledCells + unclaimedCells

		// Return the difference in controlled areas as a score.
		scores[i] = (controlledCells[i] - opponentsControlledCells) / totalControlled
	}

	return scores
}

// lengthEvaluation evaluates the board based on the length of each snake compared to opponents.
func lengthEvaluation(board Board, context *EvaluationContext) []float64 {
	numSnakes := len(board.Snakes)
	scores := make([]float64, numSnakes)

	for i := 0; i < numSnakes; i++ {
		rootSnake := board.Snakes[i]
		if isSnakeDead(rootSnake) {
			scores[i] = -2
			continue
		}

		rootLength := len(rootSnake.Body)
		lengthBonus := 0.0

		// Calculate length bonus/penalty.
		for j := 0; j < numSnakes; j++ {
			if j != i {
				opponent := board.Snakes[j]
				if isSnakeDead(opponent) {
					continue
				}

				opponentLength := len(opponent.Body)
				lengthDifference := rootLength - opponentLength

				if lengthDifference > 0 {
					// If root snake is longer, calculate bonus.
					if lengthDifference == 1 {
						lengthBonus += 0.5
					} else if float64(rootLength) > 1.1*float64(opponentLength) {
						// Cap bonus at 1.0 for being 10% longer.
						lengthBonus += 1.0
					} else {
						// Scale between 0.5 and 1.0 as the length difference increases up to 10% longer.
						extraLengthRatio := float64(rootLength) / float64(opponentLength)
						lengthBonus += 0.5 + 0.5*((extraLengthRatio-1.0)/0.1)
					}
				} else {
					// If root snake is shorter, calculate penalty.
					if lengthDifference == -1 {
						lengthBonus -= 0.1
					} else {
						// Scale penalty down to -1.0 for being 60% or less of the opponent's length.
						minLength := 0.6 * float64(opponentLength)
						if float64(rootLength) <= minLength {
							lengthBonus -= 1.0
						} else {
							// Scale between -0.1 and -1.0 as the root snake gets closer to 60% of the opponent's length.
							lengthBonus -= 0.1 + 0.9*((float64(opponentLength)-float64(rootLength))/(float64(opponentLength)*0.4))
						}
					}
				}
			}
		}

		// Ensure the result is between -1 and 1.
		if lengthBonus > 1.0 {
			lengthBonus = 1.0
		} else if lengthBonus < -1.0 {
			lengthBonus = -1.0
		}

		scores[i] = lengthBonus
	}

	return scores
}
