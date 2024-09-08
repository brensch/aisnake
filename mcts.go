package main

import (
	"context"
	"math"
)

// Methods such as copyBoard, applyMove, generateSafeMoves, and evaluateBoard
// must be correctly defined elsewhere.

type Node struct {
	Board      Board
	SnakeIndex int
	Parent     *Node
	Children   []*Node
	Visits     int
	Score      float64
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
	totalSnakes := len(node.Board.Snakes)
	currentSnakeIndex := node.SnakeIndex
	nextSnakeIndex := (currentSnakeIndex + 1) % totalSnakes
	moves := generateSafeMoves(node.Board, currentSnakeIndex)

	for _, move := range moves {
		newBoard := copyBoard(node.Board)
		applyMove(&newBoard, currentSnakeIndex, move)
		// Create node for next snake's turn but score from current snake's perspective
		child := NewNode(newBoard, nextSnakeIndex)
		child.Parent = node
		node.Children = append(node.Children, child)
	}
}

func (n *Node) UCT(parent *Node, explorationParam float64) float64 {
	exploitation := 0.0
	if n.Visits > 0 { // Check to prevent division by zero
		exploitation = n.Score / float64(n.Visits)
	}
	parentVisits := float64(parent.Visits)
	if parentVisits == 0 {
		parentVisits = 1 // Prevent division by zero in log calculation
	}
	exploration := 0.0
	if n.Visits > 0 {
		exploration = math.Sqrt(2 * math.Log(parentVisits) / float64(n.Visits))
	} else {
		exploration = math.MaxFloat64 // Encourage exploration of unvisited nodes
	}
	return exploitation + explorationParam*exploration

}

func bestChild(node *Node, explorationParam float64) *Node {
	if len(node.Children) == 0 {
		return nil
	}
	best := node.Children[0] // Initialize with the first child
	bestUCB := -math.MaxFloat64
	for _, child := range node.Children {
		ucb := child.UCT(node, explorationParam)
		if ucb > bestUCB {
			bestUCB = ucb
			best = child
		}
	}
	return best
}

func MCTS(ctx context.Context, rootBoard Board, iterations int) *Node {
	rootNode := NewNode(rootBoard, 0) // Starting with the first snake
	expand(rootNode)

	for i := 0; i < iterations; i++ {
		select {
		case <-ctx.Done():
			return rootNode
		default:
			node := rootNode
			for node != nil && len(node.Children) > 0 {
				node = bestChild(node, 1.41)
			}
			if node == nil {
				break
			}
			if node.Visits == 0 {
				expand(node)
			}
			score := evaluateBoard(node.Board)
			if node.SnakeIndex == 0 {
				score = -1 * score
			}
			for node != nil {
				node.Visits++
				node.Score += score
				score = -1 * score
				node = node.Parent
			}
		}
	}
	return rootNode
}
