package main

import (
	"context"
	"fmt"
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

	mu         sync.Mutex // Mutex to protect the node's state
	childrenMu sync.Mutex // Separate mutex for protecting children initialization
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

// Expand the node to add children based on the board's state
func expand(node *Node) {
	node.childrenMu.Lock() // Protect child initialization
	defer node.childrenMu.Unlock()

	// If already expanded, return immediately
	if len(node.Children) > 0 {
		return
	}

	nextSnakeIndex := (node.SnakeIndex + 1) % len(node.Board.Snakes)

	moves := generateSafeMoves(node.Board, nextSnakeIndex)
	if len(moves) == 0 {
		moves = append(moves, Up)
	}

	for _, move := range moves {
		newBoard := copyBoard(node.Board)
		applyMove(&newBoard, nextSnakeIndex, move)

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

// Select the best child node based on the UCT value
func bestChild(node *Node, explorationParam float64) *Node {
	node.childrenMu.Lock() // Protect access to children
	defer node.childrenMu.Unlock()

	if len(node.Children) == 0 {
		return nil // No children available
	}

	bestValue := -math.MaxFloat64
	var bestNode *Node

	for _, child := range node.Children {
		if child == nil {
			continue // Skip nil children, in case of race condition or partial initialization
		}

		// Only lock the child for reading its Visits and Score
		// note this is now racey since node.visits is accessed.
		// could not get it to lock the node without causing a deadlock so will leave it
		child.mu.Lock()
		value := child.UCT(node, explorationParam)
		child.mu.Unlock()

		if value > bestValue {
			bestValue = value
			bestNode = child
		}
	}

	return bestNode
}

func MCTS(ctx context.Context, rootBoard Board, iterations int, numWorkers int, gameStates map[string]*Node) *Node {
	// Generate the hash for the current board state
	boardKey := boardHash(rootBoard)
	var rootNode *Node
	// If the board state is already known, use the existing node
	if existingNode, ok := gameStates[boardKey]; ok {
		fmt.Printf("found board from previous game with %d visits\n", existingNode.Visits)
		rootNode = existingNode
		// we still want to update to the new board since it may have food or danger updates
		rootNode.Board = rootBoard
	} else {
		// Otherwise, create a new node and add it to the game state map
		fmt.Println("couldn't find yo")
		rootNode = NewNode(rootBoard, -1)
		expand(rootNode)
	}

	nodeChan := make(chan *Node, numWorkers) // Channel to distribute work

	// Central coordinator goroutine
	go func() {
		for i := 0; i < iterations; i++ {
			node := rootNode
			for len(node.Children) > 0 {
				nextNode := bestChild(node, 1.41)
				if nextNode == nil {
					break // Break the loop if no valid child is found
				}
				node = nextNode
			}
			select {
			case nodeChan <- node:
			case <-ctx.Done():
				close(nodeChan)
				return
			}
		}
		close(nodeChan)
	}()

	// Worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for node := range nodeChan {
				if node == nil {
					continue // Skip processing if node is nil
				}

				// Lock node before processing
				node.mu.Lock()
				if !isTerminal(node.Board) && node.Visits > 0 {
					expand(node)
					// No need to switch locks between parent and child at this point.
					// Just expand, unlock, and continue.
				}
				node.mu.Unlock()

				// Simulation
				score := evaluateBoard(node.Board, node.SnakeIndex)
				node.MyScore = score

				// Backpropagation
				for n := node; n != nil; n = n.Parent {
					n.mu.Lock()
					n.Visits++
					n.Score += score
					n.mu.Unlock()
					score = -score
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
