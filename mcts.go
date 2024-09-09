package main

import (
	"context"
	"math"
	"sync"
)

type Node struct {
	Board      Board
	SnakeIndex int
	Parent     *Node
	Children   []*Node
	Visits     int
	Score      float64
	MyScore    float64

	mu sync.Mutex
}

func NewNode(board Board, snakeIndex int) *Node {
	return &Node{
		Board:      board,
		SnakeIndex: snakeIndex,
		Children:   make([]*Node, 0), // Ensuring Children slice is initialized
		Visits:     0,
		Score:      0,
	}
}

func expand(node *Node) {

	nextSnakeIndex := (node.SnakeIndex + 1) % len(node.Board.Snakes)

	moves := generateSafeMoves(node.Board, nextSnakeIndex)
	// cannot have 0 or it will seem like nothing bad happens at the end. need to see the death.
	if len(moves) == 0 {
		moves = append(moves, Up)
	}

	for _, move := range moves {
		newBoard := copyBoard(node.Board)
		applyMove(&newBoard, nextSnakeIndex, move)

		// Fix the SnakeIndex to reflect the snake who is deciding the next move
		child := NewNode(newBoard, nextSnakeIndex)
		child.Parent = node
		node.Children = append(node.Children, child)
	}
}

func isTerminal(board Board) bool {
	aliveSnakesCount := 0
	for _, snake := range board.Snakes {
		if !isSnakeDead(snake) {
			aliveSnakesCount++
		}
	}
	return aliveSnakesCount <= 1
}

// isSnakeDead checks if a snake is dead by looking at its health and body length.
func isSnakeDead(snake Snake) bool {
	return len(snake.Body) == 0 || snake.Health <= 0
}

func (n *Node) UCT(parent *Node, explorationParam float64) float64 {
	if n.Visits == 0 {
		return math.MaxFloat64
	}

	exploitation := n.Score / float64(n.Visits)
	exploration := explorationParam * math.Sqrt(math.Log(float64(parent.Visits))/float64(n.Visits))

	return exploitation + exploration
}

func bestChild(node *Node, explorationParam float64) *Node {
	bestValue := -math.MaxFloat64
	var bestNode *Node

	for _, child := range node.Children {
		value := child.UCT(node, explorationParam)
		if value > bestValue {
			bestValue = value
			bestNode = child
		}
	}

	return bestNode
}

func MCTS(ctx context.Context, rootBoard Board, iterationsPerThread, numThreads int) *Node {
	rootNode := NewNode(rootBoard, -1)
	expand(rootNode)

	var wg sync.WaitGroup
	wg.Add(numThreads)
	for i := 0; i < numThreads; i++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iterationsPerThread; i++ {
				select {
				case <-ctx.Done():
					return
				default:
					node := rootNode

					// Selection
					for len(node.Children) > 0 {
						node = bestChild(node, 1.41)
					}

					// Expansion
					if !isTerminal(node.Board) && node.Visits > 0 {
						expand(node)
						if len(node.Children) > 0 {
							node = node.Children[0] // Select the first child for simulation
						}
					}

					// Simulation
					score := evaluateBoard(node.Board, node.SnakeIndex)
					node.MyScore = score
					// score := simulate(node.Board, node.SnakeIndex)

					// Backpropagation using parent pointers, ensuring we don't hit nil
					for n := node; n != nil; n = n.Parent {
						n.Visits++
						n.Score += score
						score = -score
					}
				}
			}
		}()

	}
	wg.Wait()
	return rootNode
}

func evaluateBoard(board Board, snakeIndex int) float64 {

	// Voronoi evaluation: Calculate the area controlled by each snake
	voronoi := GenerateVoronoi(board)
	totalCells := float64(board.Width * board.Height)
	controlledCells := 0.0
	opponentsCells := 0.0

	// Count the number of cells each snake controls in the Voronoi diagram
	for y := 0; y < board.Height; y++ {
		for x := 0; x < board.Width; x++ {
			if voronoi[y][x] == snakeIndex {
				controlledCells++
			} else if voronoi[y][x] != -1 {
				opponentsCells++
			}
		}
	}

	// Return a score between -1 and 1 based on the difference in controlled areas
	return (controlledCells - opponentsCells) / totalCells
}
