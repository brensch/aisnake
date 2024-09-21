package main

import (
	"container/heap"
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
			Weight:   20,
		},
		{
			EvalFunc: lengthEvaluation,
			Weight:   30,
		},
		{
			EvalFunc: luckEvaluation,
			Weight:   15,
		},
		{
			EvalFunc: otherSnakeEvaluation,
			Weight:   15,
		},
		// {
		// 	EvalFunc: trappedEvaluation,
		// 	Weight:   15,
		// },
	}
)

// EvaluationContext holds precomputed data for evaluation functions to avoid redundant computations.
type EvaluationContext struct {
	// AllPaths [][][]dijkstraNode
	Voronoi [][]int
	// LongestPaths []int //TODO: might add this back for trapped snakes
	LuckMatrix []bool
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

		// Determine tail removal based on steps
		stepsToRemove := steps
		if snakeIndex < i {
			stepsToRemove++
		}

		if stepsToRemove < len(otherSnake.Body) {
			otherSnake.Body = otherSnake.Body[0 : len(otherSnake.Body)-stepsToRemove]
		} else {
			otherSnake.Body = []Point{}
		}

		// Check for collisions
		for _, segment := range otherSnake.Body {
			if newHead == segment {
				return false
			}
		}

		// Head-to-head collision check
		if newHead == otherSnake.Head && len(otherSnake.Body) >= len(snake.Body) {
			return false
		}
	}

	return true
}

// GenerateVoronoi generates a board ownership diagram based on a shortest path algorithm.
func GenerateVoronoi(board Board) [][]int {
	bestPaths := make([][]dijkstraNode, board.Height)
	for i := range bestPaths {
		bestPaths[i] = make([]dijkstraNode, board.Width)
		for j := range bestPaths[i] {
			bestPaths[i][j] = dijkstraNode{Point{-1, -1}, -1, -1} // Initialize as unassigned
		}
	}

	pq := &PriorityQueue{}
	heap.Init(pq)

	// Initialize the priority queue with all snake heads
	for k, snake := range board.Snakes {
		if snake.Health > 0 && len(snake.Body) > 0 {
			head := snake.Head
			heap.Push(pq, dijkstraNode{head, k, 0})
			bestPaths[head.Y][head.X] = dijkstraNode{head, k, 0} // Track snake ownership
		}
	}

	// Process the priority queue
	for pq.Len() > 0 {
		node := heap.Pop(pq).(dijkstraNode)
		currentPoint := node.point

		// Explore in all directions
		for _, direction := range AllDirections {
			newPoint := moveHead(currentPoint, direction)

			// Ensure the new point is within bounds and legal for this snake
			if newPoint.X >= 0 && newPoint.X < board.Width && newPoint.Y >= 0 && newPoint.Y < board.Height &&
				isLegalMove(board, node.snakeIndex, newPoint, node.distance+1) {

				newDistance := node.distance + 1
				bestNode := bestPaths[newPoint.Y][newPoint.X]

				// Update if the new path is better
				if bestNode.snakeIndex == -1 || newDistance < bestNode.distance {
					bestPaths[newPoint.Y][newPoint.X] = dijkstraNode{newPoint, node.snakeIndex, newDistance}
					heap.Push(pq, dijkstraNode{newPoint, node.snakeIndex, newDistance})
				}
			}
		}
	}

	// Convert the result to a simple snake ownership grid
	return dijkstraToResult(bestPaths)
}

// Converts the bestPaths grid to a simple snake ownership grid
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

// evaluateBoard evaluates the board state and returns an array of scores for each snake.
func evaluateBoard(node *Node, modules []EvaluationModule) ([]float64, [][]float64) {
	numSnakes := len(node.Board.Snakes)
	scores := make([]float64, numSnakes)
	scoreBreakdown := make([][]float64, len(modules))
	for i := range scoreBreakdown {
		scoreBreakdown[i] = make([]float64, numSnakes)
	}

	// Create EvaluationContext and precompute data
	context := &EvaluationContext{
		LuckMatrix: node.LuckMatrix,
		Voronoi:    GenerateVoronoi(node.Board),
	}
	// fmt.Println(VisualizeVoronoi(context.Voronoi, node.Board.Snakes))
	// fmt.Println(visualizeBoard(node.Board))
	// Precompute other data if necessary

	// Calculate the sum of all weights for normalization.
	totalWeight := 0.0
	for _, module := range modules {
		totalWeight += module.Weight
	}

	// For each module, get the scores, apply weight, and accumulate
	for i, module := range modules {
		moduleScores := module.EvalFunc(node.Board, context)
		scoreBreakdown[i] = moduleScores
		for j := 0; j < numSnakes; j++ {
			// fmt.Println(moduleScores)
			weightedScore := (module.Weight / totalWeight) * moduleScores[j]
			scores[j] += weightedScore
		}
	}

	// calculate snake deaths to save on calcs for all in each loop
	snakeDeaths := make([]bool, numSnakes)
	draw := true
	for i := 0; i < numSnakes; i++ {
		dead := isSnakeDead(node.Board.Snakes[i])
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
			scores[i] = -4
			continue
		}

		// Check if all opponents are dead
		aliveOpponents := 0
		for j := 0; j < numSnakes; j++ {
			if j != i && !isSnakeDead(node.Board.Snakes[j]) {
				aliveOpponents++
			}
		}
		if aliveOpponents == 0 {
			// All opponents are dead, set score to maximum
			scores[i] = 4
			continue
		}

		// // Otherwise, normalize score between -1 and 1
		// if scores[i] > 1 {
		// 	scores[i] = 1
		// } else if scores[i] < -1 {
		// 	scores[i] = -1
		// }
	}

	return scores, scoreBreakdown
}

// voronoiEvaluation evaluates the board based on Voronoi control.
func voronoiEvaluation(board Board, context *EvaluationContext) []float64 {
	numSnakes := len(board.Snakes)
	scores := make([]float64, numSnakes)

	voronoiOwnership := context.Voronoi

	// Count the number of cells each snake controls in the Voronoi diagram.
	controlledCells := make([]float64, numSnakes)
	unclaimedCells := 0.0

	for y := 0; y < board.Height; y++ {
		for x := 0; x < board.Width; x++ {
			snakeIndex := voronoiOwnership[y][x]
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

// TODO: maybe add back
// func trappedEvaluation(board Board, context *EvaluationContext) []float64 {
// 	numSnakes := len(board.Snakes)
// 	scores := make([]float64, numSnakes)

// 	for i, snake := range board.Snakes {
// 		if context.LongestPaths[i] < len(snake.Body) {
// 			scores[i] = -1
// 		}
// 	}

// 	return scores
// }

const minLengthScore = -4

// lengthEvaluation evaluates the board based on the length of each snake compared to opponents.
func lengthEvaluation(board Board, context *EvaluationContext) []float64 {
	numSnakes := len(board.Snakes)
	scores := make([]float64, numSnakes)

	// Determine the length of the longest snake
	maxLength := 0
	for _, snake := range board.Snakes {
		if !isSnakeDead(snake) {
			snakeLength := len(snake.Body)
			if snakeLength > maxLength {
				maxLength = snakeLength
			}
		}
	}

	// Evaluate each snake's length compared to the longest snake
	for i, snake := range board.Snakes {
		if isSnakeDead(snake) {
			scores[i] = minLengthScore // Heavy penalty for dead snakes
			continue
		}

		snakeLength := len(snake.Body)
		if snakeLength == maxLength {
			// Longest snake gets a score of 1
			scores[i] = 1.0
		} else {
			// Shorter snakes get a proportionate penalty, capped at -2
			lengthDiff := maxLength - snakeLength
			scores[i] = calculateProportionatePenalty(lengthDiff, maxLength)
		}
	}

	return scores
}

// calculateProportionatePenalty returns a negative score proportional to how far behind
// the snake is compared to the longest snake, with a minimum of -2.
func calculateProportionatePenalty(lengthDiff, maxLength int) float64 {
	if lengthDiff <= 0 {
		return 1.0 // Should not happen, as the longest snake is handled earlier
	}

	// Calculate proportionate penalty: Linear scaling from 0 to -2
	// Use the length difference as a ratio of maxLength for scaling
	penalty := minLengthScore * (float64(lengthDiff) / float64(maxLength))

	// Ensure that the penalty doesn't go below -2
	if penalty < minLengthScore {
		penalty = minLengthScore
	}

	return penalty
}

// luckEvaluation checks if the snake's move relies on luck for this branch.
// luck means another snake could have moved into our head at the same time and we both died.
func luckEvaluation(board Board, context *EvaluationContext) []float64 {
	numSnakes := len(board.Snakes)
	scores := make([]float64, numSnakes)

	for i := 0; i < numSnakes; i++ {
		// If the snake relies on luck, apply a negative score.
		if context.LuckMatrix[i] {
			scores[i] = -4
		}
	}

	return scores
}

// luckEvaluation checks if the snake's move relies on luck for this branch.
// luck means another snake could have moved into our head at the same time and we both died.
func otherSnakeEvaluation(board Board, context *EvaluationContext) []float64 {
	scores := make([]float64, len(board.Snakes))

	aliveSnakes := 0
	for _, snake := range board.Snakes {
		if isSnakeDead(snake) {
			continue
		}
		aliveSnakes++
	}

	// count how many other snakes are alive by subtracting 1 if we are alive from total alive snakes
	for i, snake := range board.Snakes {
		snakeAliveValue := 0
		if !isSnakeDead(snake) {
			snakeAliveValue = 1
		}
		scores[i] = -float64(aliveSnakes - snakeAliveValue)
	}

	return scores
}
