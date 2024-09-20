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
			Weight:   8,
		},
		{
			EvalFunc: lengthEvaluation,
			Weight:   5,
		},
		{
			EvalFunc: luckEvaluation,
			Weight:   5,
		},
		{
			EvalFunc: trappedEvaluation,
			Weight:   5,
		},
	}
)

// EvaluationContext holds precomputed data for evaluation functions to avoid redundant computations.
type EvaluationContext struct {
	AllPaths     [][][]dijkstraNode
	LongestPaths []int
	LuckMatrix   []bool
}

// GenerateVoronoi generates a board ownership diagram based on independently computed shortest path maps for each snake.
func GenerateVoronoi(board Board) ([][][]dijkstraNode, []int) {
	allPaths := make([][][]dijkstraNode, len(board.Snakes))
	longestPaths := make([]int, len(board.Snakes))
	// Initialize distance maps for each snake
	for index, snake := range board.Snakes {
		allPaths[index] = make([][]dijkstraNode, board.Height)
		for y := range allPaths[index] {
			allPaths[index][y] = make([]dijkstraNode, board.Width)
			for x := range allPaths[index][y] {
				allPaths[index][y][x] = dijkstraNode{Point{x, y}, index, math.MaxInt32}
			}
		}
		longestPaths[index] = calculatePathsForSnake(&board, index, snake, allPaths[index])
	}

	return allPaths, longestPaths
	// return resolveOwnership(allPaths)
}

// calculatePathsForSnake calculates shortest paths from a single snake's head using Dijkstra's algorithm
func calculatePathsForSnake(board *Board, snakeIndex int, snake Snake, paths [][]dijkstraNode) int {
	pq := &PriorityQueue{}
	heap.Init(pq)
	start := snake.Head
	paths[start.Y][start.X].distance = 0
	heap.Push(pq, dijkstraNode{start, snakeIndex, 0})
	longestPath := -1

	for pq.Len() > 0 {
		// fmt.Println(snakeIndex)
		// visualisePQ(paths)
		current := heap.Pop(pq).(dijkstraNode)

		for _, direction := range AllDirections {
			nextPoint := moveHead(current.point, direction)
			if isLegalMove(*board, snakeIndex, nextPoint, current.distance+1) {
				newDistance := current.distance + 1
				if newDistance < paths[nextPoint.Y][nextPoint.X].distance {
					paths[nextPoint.Y][nextPoint.X] = dijkstraNode{nextPoint, snakeIndex, newDistance}
					heap.Push(pq, dijkstraNode{nextPoint, snakeIndex, newDistance})
				}
				if newDistance > longestPath {
					longestPath = newDistance
				}
			}
		}
	}
	return longestPath
}

func resolveOwnership(allPaths [][][]dijkstraNode) [][]int {
	height := len(allPaths[0])
	width := len(allPaths[0][0])
	ownership := make([][]int, height)
	for y := 0; y < height; y++ {
		ownership[y] = make([]int, width)
		for x := 0; x < width; x++ {
			owner := -1
			minDistance := math.MaxInt32
			tie := false // Flag to detect ties

			// Check all snakes for the shortest path to this cell
			for i, paths := range allPaths {
				if paths[y][x].distance < minDistance {
					minDistance = paths[y][x].distance
					owner = i
					tie = false // New shortest path found, no tie
				} else if paths[y][x].distance == minDistance {
					tie = true // A tie is found
				}
			}

			// If there was a tie, resolve it or leave the cell unclaimed
			if tie {
				owner = -1 // No snake claims the cell in the case of a tie
			}

			ownership[y][x] = owner
		}
	}
	return ownership
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
	}
	context.AllPaths, context.LongestPaths = GenerateVoronoi(node.Board)
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
			scores[i] = -1
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

	return scores, scoreBreakdown
}

// voronoiEvaluation evaluates the board based on Voronoi control.
func voronoiEvaluation(board Board, context *EvaluationContext) []float64 {
	numSnakes := len(board.Snakes)
	scores := make([]float64, numSnakes)

	// Count the number of cells each snake controls in the Voronoi diagram.
	controlledCells := make([]float64, numSnakes)
	unclaimedCells := 0.0

	voronoi := resolveOwnership(context.AllPaths)

	for y := 0; y < board.Height; y++ {
		for x := 0; x < board.Width; x++ {
			snakeIndex := voronoi[y][x]
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

func trappedEvaluation(board Board, context *EvaluationContext) []float64 {
	numSnakes := len(board.Snakes)
	scores := make([]float64, numSnakes)

	for i, snake := range board.Snakes {
		if context.LongestPaths[i] < len(snake.Body) {
			scores[i] = -1
		}
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

// luckEvaluation checks if the snake's move relies on luck for this branch.
// luck means another snake could have moved into our head at the same time and we both died.
func luckEvaluation(board Board, context *EvaluationContext) []float64 {
	numSnakes := len(board.Snakes)
	scores := make([]float64, numSnakes)

	for i := 0; i < numSnakes; i++ {
		// If the snake relies on luck, apply a negative score.
		if context.LuckMatrix[i] {
			scores[i] = -.9
		}
	}

	return scores
}
