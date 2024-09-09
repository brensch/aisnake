package main

import (
	"container/heap"
	"container/list"
	"math"
)

// GenerateVoronoi generates a Voronoi diagram for the given board.
func GenerateVoronoi2(board Board) [][]int {
	voronoi := make([][]int, board.Height)
	for i := range voronoi {
		voronoi[i] = make([]int, board.Width)
		for j := range voronoi[i] {
			voronoi[i][j] = -1 // Initialize all positions as unassigned
		}
	}

	for y := 0; y < board.Height; y++ {
		for x := 0; x < board.Width; x++ {
			minDistance := math.MaxInt32
			closestSnake := -1

			for k, snake := range board.Snakes {
				distance := manhattanDistance(Point{x, y}, snake.Head)
				if distance < minDistance {
					minDistance = distance
					closestSnake = k
				} else if distance == minDistance {
					// In case of a tie, leave the cell unassigned (-1)
					closestSnake = -1
				}
			}

			voronoi[y][x] = closestSnake
		}
	}

	return voronoi
}

// manhattanDistance calculates the Manhattan distance between two points.
func manhattanDistance(a, b Point) int {
	return int(math.Abs(float64(a.X-b.X)) + math.Abs(float64(a.Y-b.Y)))
}

// GenerateVoronoi generates a flood-fill diagram for the given board,
// where each cell is filled with the closest snake according to legal moves,
// and when ties in distance occur, the longer snake wins control.
func GenerateVoronoiFlood(board Board) [][]int {
	// Flood fill structure that now contains both the snake index and the snake's length when they reach the point
	floodFill := make([][]floodFillCell, board.Height)
	for i := range floodFill {
		floodFill[i] = make([]floodFillCell, board.Width)
		for j := range floodFill[i] {
			floodFill[i][j] = floodFillCell{-1, -1, -1} // Initialize all positions as unassigned
		}
	}

	// Queue for flood fill, each element contains the current point, snake index, current depth (distance), and snake length
	queue := list.New()

	// Initialize the queue with the heads of all snakes
	for k, snake := range board.Snakes {
		if snake.Health > 0 && len(snake.Body) > 0 { // Skip dead or empty snakes
			queue.PushBack(floodFillNode{snake.Head, k, 0, len(snake.Body)})
			floodFill[snake.Head.Y][snake.Head.X] = floodFillCell{k, 0, len(snake.Body)} // Snake index, distance, and snake length
		}
	}

	// Perform flood fill
	for queue.Len() > 0 {
		element := queue.Front()
		node := element.Value.(floodFillNode)
		queue.Remove(element)

		snakeIndex := node.snakeIndex
		currentPoint := node.point
		currentLength := node.snakeLength

		// Get legal moves for the current point
		for _, direction := range AllDirections {
			newPoint := moveHead(currentPoint, direction)

			// Ensure new point is within bounds and unassigned
			if newPoint.X >= 0 && newPoint.X < board.Width && newPoint.Y >= 0 && newPoint.Y < board.Height {
				if floodFill[newPoint.Y][newPoint.X].snakeIndex == -1 {
					// Check if the move is legal for the snake at snakeIndex
					if isLegalMove(board, snakeIndex, newPoint) {
						// Assign this point to the current snake
						floodFill[newPoint.Y][newPoint.X] = floodFillCell{snakeIndex, node.depth + 1, currentLength}
						queue.PushBack(floodFillNode{newPoint, snakeIndex, node.depth + 1, currentLength})
					}
				} else {
					// Handle ties - if the current snake can reach this point with the same distance but is longer
					existingCell := floodFill[newPoint.Y][newPoint.X]
					if existingCell.distance == node.depth+1 && currentLength > existingCell.snakeLength {
						floodFill[newPoint.Y][newPoint.X] = floodFillCell{snakeIndex, node.depth + 1, currentLength}
					}
				}
			}
		}
	}

	return floodFillToResult(floodFill)
}

// floodFillNode represents a node in the flood-fill queue.
type floodFillNode struct {
	point       Point
	snakeIndex  int
	depth       int // Depth is the number of moves from the snake's head
	snakeLength int // The length of the snake when it reaches this point
}

// floodFillCell represents the ownership of a cell in the flood fill process.
type floodFillCell struct {
	snakeIndex  int // Index of the snake controlling this cell
	distance    int // Distance from the snake's head
	snakeLength int // Length of the snake when it reaches this point
}

// floodFillToResult converts the flood fill grid to a simple snake ownership grid (used for debugging)
func floodFillToResult(floodFill [][]floodFillCell) [][]int {
	result := make([][]int, len(floodFill))
	for i := range result {
		result[i] = make([]int, len(floodFill[i]))
		for j := range result[i] {
			result[i][j] = floodFill[i][j].snakeIndex
		}
	}
	return result
}

// isLegalMove checks if a move to a new point is legal for the snake.
func isLegalMove(board Board, snakeIndex int, newHead Point) bool {
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

		// remove the snake's tail if it went before us (because of turn based approximation of simultaneous moves)
		if snakeIndex < i {
			otherSnake.Body = otherSnake.Body[0 : len(otherSnake.Body)-1]
		}
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
				if isLegalMove(board, node.snakeIndex, newPoint) {
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
