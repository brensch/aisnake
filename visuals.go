package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

func visualizeBoard(game Board, options ...func(*boardOptions)) string {
	var sb strings.Builder

	// Default options
	opts := &boardOptions{
		indent:           "",
		newlineCharacter: "\n",
		snakeIndex:       -1,    // Default: no snake selected
		move:             Unset, // Default: no move selected
	}

	// Apply any options provided
	for _, opt := range options {
		opt(opts)
	}

	// Validate board dimensions
	if game.Height <= 0 || game.Width <= 0 {
		return opts.indent + "Invalid board dimensions"
	}

	// Display the move at the top if a move is set
	var arrow rune
	if opts.move != Unset && opts.snakeIndex >= 0 && opts.snakeIndex < len(game.Snakes) {
		sb.WriteString(opts.indent)
		snakeChar := rune('a' + opts.snakeIndex)
		switch opts.move {
		case Up:
			arrow = '↑'
		case Down:
			arrow = '↓'
		case Left:
			arrow = '←'
		case Right:
			arrow = '→'
		default:
			arrow = ' ' // Handle unexpected cases
		}
		sb.WriteRune(snakeChar)
		sb.WriteRune(arrow)
		sb.WriteString(opts.newlineCharacter)
	}

	// Extend the board by 1 in every direction
	extendedHeight := game.Height + 2
	extendedWidth := game.Width + 2

	// Create a 2D slice to represent the extended board
	board := make([][]rune, extendedHeight)
	for i := range board {
		board[i] = make([]rune, extendedWidth)
		for j := range board[i] {
			if i == 0 || i == extendedHeight-1 || j == 0 || j == extendedWidth-1 {
				board[i][j] = 'x' // Set the boundary to 'x'
			} else {
				board[i][j] = '.' // Initialize all positions as empty
			}
		}
	}

	// Function to adjust the Y coordinate safely
	adjustY := func(y int) int {
		if y < 0 || y >= game.Height {
			return -1 // Return invalid index if out of bounds
		}
		return extendedHeight - 1 - (y + 1)
	}

	// Place food on the board safely
	for _, food := range game.Food {
		adjustedY := adjustY(food.Y)
		if adjustedY != -1 && food.X+1 < extendedWidth {
			board[adjustedY][food.X+1] = '♥'
		}
	}

	// Place hazards on the board safely
	for _, hazard := range game.Hazards {
		adjustedY := adjustY(hazard.Y)
		if adjustedY != -1 && hazard.X+1 < extendedWidth {
			board[adjustedY][hazard.X+1] = 'H'
		}
	}

	// Place snakes on the board safely
	for i, snake := range game.Snakes {
		if len(snake.Body) == 0 || snake.Health == 0 {
			continue
		}
		snakeChar := rune('a' + i)
		if snakeChar > 'z' {
			snakeChar = '?' // Fallback in case of too many snakes
		}

		headY := adjustY(snake.Head.Y)
		if headY != -1 && snake.Head.X+1 < extendedWidth {
			board[headY][snake.Head.X+1] = unicode.ToUpper(snakeChar)
		}
		for _, part := range snake.Body[1:] {
			partY := adjustY(part.Y)
			if partY != -1 && part.X+1 < extendedWidth {
				board[partY][part.X+1] = snakeChar
			}
		}
	}

	// Overlay the arrow for the current snake's move safely
	if opts.move != Unset && opts.snakeIndex != -1 && arrow != ' ' {
		newHead := moveHead(game.Snakes[opts.snakeIndex].Head, opts.move)
		adjustedY := adjustY(newHead.Y)
		if adjustedY != -1 && newHead.X+1 < extendedWidth {
			board[adjustedY][newHead.X+1] = arrow
		}
	}

	// // Append the health status of each snake
	// for i, snake := range game.Snakes {
	// 	sb.WriteString(fmt.Sprintf("Snake %c health: %d", 'a'+i, snake.Health))
	// 	sb.WriteString(opts.newlineCharacter)
	// }
	// sb.WriteString(opts.newlineCharacter)

	// Build the string representation of the board using manual spacing for alignment
	for _, row := range board {
		sb.WriteString(opts.indent)
		for _, cell := range row {
			sb.WriteRune(cell)
			sb.WriteString("  ") // Add extra spacing to simulate a table
		}
		sb.WriteString(opts.newlineCharacter)
	}

	return sb.String()
}

// Options struct to hold the customizable parameters
type boardOptions struct {
	indent           string
	newlineCharacter string
	move             Direction // Represents the move of a single snake
	snakeIndex       int       // The index of the snake whose move is being visualized
}

// Option functions to set optional parameters
func WithIndent(indent string) func(*boardOptions) {
	return func(o *boardOptions) {
		o.indent = indent
	}
}

func WithNewlineCharacter(newlineCharacter string) func(*boardOptions) {
	return func(o *boardOptions) {
		o.newlineCharacter = newlineCharacter
	}
}

func WithMove(move Direction, snakeIndex int) func(*boardOptions) {
	return func(o *boardOptions) {
		o.move = move
		o.snakeIndex = snakeIndex
	}
}

// Path represents a path in the tree with its corresponding depth
type Path struct {
	Nodes []*Node
	Depth int
}

func VisualizeVoronoi(voronoi [][]int, snakes []Snake, options ...func(*boardOptions)) string {
	var sb strings.Builder

	// Default options
	opts := &boardOptions{
		indent:           "",
		newlineCharacter: "\n",
	}

	// Apply any options provided
	for _, opt := range options {
		opt(opts)
	}

	// Reverse the order of rows to display the grid correctly
	for y := len(voronoi) - 1; y >= 0; y-- {
		sb.WriteString(opts.indent) // Apply indentation for each row
		for x := 0; x < len(voronoi[y]); x++ {
			owner := voronoi[y][x]
			if owner == -1 {
				sb.WriteString(".") // Unassigned cells
			} else {
				sb.WriteString(fmt.Sprintf("%c", 'A'+owner)) // Each snake gets a unique letter
			}
			sb.WriteString("  ") // Add extra spacing to simulate a table
		}
		sb.WriteString(opts.newlineCharacter)
	}

	return sb.String()
}

// visualizeNode generates the DOT representation of a single node, including its label, visits, score, board state, and controlled positions
func visualizeNode(node *Node) string {
	if node == nil {
		return ""
	}

	nodeID := fmt.Sprintf("Node_%p", node)
	// Using <br/> instead of \n to create HTML-based line breaks that D3 can interpret
	nodeLabel := fmt.Sprintf("%s\nVisits: %d\nAvg Score: %.3f\nMy Score: %.3f\nSnake moving: %c\n\n",
		nodeID, node.Visits, node.Score/float64(node.Visits), node.MyScore, 'A'+node.SnakeIndex)
	voronoi := GenerateVoronoi(node.Board)
	controlledPositions := make([]int, len(node.Board.Snakes))
	for _, row := range voronoi {
		for _, owner := range row {
			if owner >= 0 && owner < len(controlledPositions) {
				controlledPositions[owner]++
			}
		}
	}
	for i, count := range controlledPositions {
		nodeLabel += fmt.Sprintf("Snake %c: %d cells, %d len\n", 'A'+i, count, len(node.Board.Snakes[i].Body))
	}
	// Add the board state visualization
	boardVisualization := visualizeBoard(node.Board, WithNewlineCharacter("\n"))
	nodeLabel += boardVisualization

	// Add controlled positions from the Voronoi diagram

	nodeVoronoiVisualization := VisualizeVoronoi(voronoi, node.Board.Snakes, WithNewlineCharacter("\n"))
	nodeLabel += "\n" + nodeVoronoiVisualization

	// Return the node label with HTML line breaks
	return nodeLabel
}

type TreeNode struct {
	ID            string      `json:"id"`
	Visits        int         `json:"visits"`
	AverageScore  float64     `json:"avg_score"`
	UCB           float64     `json:"ucb"`
	IsMostVisited bool        `json:"isMostVisited"`
	Children      []*TreeNode `json:"children"`
	Body          string      `json:"body"`
	Board         Board       `json:"board"`
}

func GenerateMostVisitedPathWithAlternativesHtmlTree(node *Node) error {

	treeNode := generateTreeData(node)
	timestamp := time.Now().Format("20060102_150405.000000")
	uuid := uuid.New().String()
	fileName := fmt.Sprintf("%s_%s", timestamp, uuid)

	fileLocation := filepath.Join("visualiser", "tree-data", fmt.Sprintf("%s.json", fileName))

	// Create the output file
	file, err := os.Create(fileLocation)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Generate the tree data in JSON format
	err = json.NewEncoder(file).Encode(treeNode)
	if err != nil {
		return err
	}

	fmt.Printf("Generated move tree: %s\nLink: http://localhost:5173/trees/%s\n", uuid, fileName)
	return nil
}

// generateTreeData recursively generates the tree structure in JSON format
func generateTreeData(node *Node) *TreeNode {
	if node == nil {
		return nil
	}

	rootNode := &TreeNode{
		ID:            fmt.Sprintf("Node_%p", node),
		Visits:        node.Visits,
		UCB:           0.0, // Root has no UCB
		IsMostVisited: true,
		Children:      make([]*TreeNode, 0),
		Body:          visualizeNode(node),
		Board:         node.Board,
	}

	// Traverse children
	traverseAndBuildTree(node, rootNode)
	return rootNode
}

// traverseAndBuildTree populates the TreeNode structure with children and marks the most visited path
func traverseAndBuildTree(node *Node, treeNode *TreeNode) {
	if node == nil {
		return
	}

	// Sort children by visit count, descending
	sort.Slice(node.Children, func(i, j int) bool {
		return node.Children[i].Visits > node.Children[j].Visits
	})

	for i, child := range node.Children {
		childNode := &TreeNode{
			ID:            fmt.Sprintf("Node_%p", child),
			Visits:        child.Visits,
			UCB:           child.UCT(node, 1.41),
			IsMostVisited: i == 0, // Only mark the most visited path
			Children:      make([]*TreeNode, 0),
			Body:          visualizeNode(child),
			Board:         child.Board,
		}

		treeNode.Children = append(treeNode.Children, childNode)

		// Recur only on the most visited child
		// if i == 0 {
		traverseAndBuildTree(child, childNode)
		// }
	}
}
