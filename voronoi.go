package main

import (
	"container/heap"
)

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

type dijkstraNode struct {
	point       Point
	snakeIndex  int
	distance    int // Number of moves from the snake's head
	snakeLength int // Length of the snake
}

// Priority queue for Dijkstra's algorithm
type PriorityQueue []dijkstraNode

// Implement heap.Interface for PriorityQueue
func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// Priority based on distance first, and snake length for tie-breaking
	if pq[i].distance == pq[j].distance {
		return pq[i].snakeLength > pq[j].snakeLength
	}
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

// GenerateVoronoi generates a board ownership diagram based on a shortest path algorithm
func GenerateVoronoi(board Board) [][]int {
	// Track the best path (shortest distance and longest snake) to each position
	bestPaths := make([][]dijkstraNode, board.Height)
	for i := range bestPaths {
		bestPaths[i] = make([]dijkstraNode, board.Width)
		for j := range bestPaths[i] {
			bestPaths[i][j] = dijkstraNode{Point{-1, -1}, -1, -1, -1} // Initialize all positions as unassigned
		}
	}

	// Priority queue (min-heap) to process nodes based on distance
	pq := &PriorityQueue{}
	heap.Init(pq)

	// Initialize the priority queue with the heads of all snakes
	for k, snake := range board.Snakes {
		if snake.Health > 0 && len(snake.Body) > 0 { // Skip dead or empty snakes
			head := snake.Head
			heap.Push(pq, dijkstraNode{head, k, 0, len(snake.Body)})
			bestPaths[head.Y][head.X] = dijkstraNode{head, k, 0, len(snake.Body)} // Record snake index, distance, and snake length
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

					// Check if this path is better (shorter distance or same distance but longer snake)
					bestNode := bestPaths[newPoint.Y][newPoint.X]
					if bestNode.snakeIndex == -1 || newDistance < bestNode.distance ||
						(newDistance == bestNode.distance && node.snakeLength > bestNode.snakeLength) {

						// Update with the better path
						bestPaths[newPoint.Y][newPoint.X] = dijkstraNode{newPoint, node.snakeIndex, newDistance, node.snakeLength}
						heap.Push(pq, dijkstraNode{newPoint, node.snakeIndex, newDistance, node.snakeLength})
					}
				}
			}
		}
	}

	return dijkstraToResult(bestPaths)
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
