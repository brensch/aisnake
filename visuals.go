package main

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

func visualizeBoard(game Board, options ...func(*boardOptions)) string {
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

	// Ensure board dimensions are positive
	if game.Height <= 0 || game.Width <= 0 {
		return opts.indent + "Invalid board dimensions"
	}

	// Create a 2D slice to represent the board
	board := make([][]rune, game.Height)
	for i := range board {
		board[i] = make([]rune, game.Width)
		for j := range board[i] {
			board[i][j] = '.' // Initialize all positions as empty
		}
	}

	// Function to adjust the Y coordinate to match the expected orientation
	adjustY := func(y int) int {
		return game.Height - 1 - y
	}

	// Helper function to check for out-of-bounds errors
	checkOOB := func(x, y int) bool {
		return x >= 0 && x < game.Width && y >= 0 && y < game.Height
	}

	// Place food on the board
	for _, food := range game.Food {
		if checkOOB(food.X, food.Y) {
			board[adjustY(food.Y)][food.X] = '🍎'
		} else {
			sb.WriteString(fmt.Sprintf("%sFood OOB at (%d, %d)%s", opts.indent, food.X, food.Y, opts.newlineCharacter))
		}
	}

	// Place hazards on the board
	for _, hazard := range game.Hazards {
		if checkOOB(hazard.X, hazard.Y) {
			board[adjustY(hazard.Y)][hazard.X] = 'H'
		} else {
			sb.WriteString(fmt.Sprintf("%sHazard OOB at (%d, %d)%s", opts.indent, hazard.X, hazard.Y, opts.newlineCharacter))
		}
	}

	// Place snakes on the board
	for snakeIndex, snake := range game.Snakes {
		if len(snake.Body) == 0 {
			sb.WriteString(fmt.Sprintf("%sSnake has 0 length %s%s", opts.indent, snake.ID, opts.newlineCharacter))
			continue
		}

		// Calculate the character to represent this snake
		snakeChar := rune('a' + snakeIndex)
		if snakeChar > 'z' {
			snakeChar = '?' // Fallback in case of too many snakes
		}

		// Place snake head first
		if checkOOB(snake.Head.X, snake.Head.Y) {
			board[adjustY(snake.Head.Y)][snake.Head.X] = unicode.ToUpper(snakeChar)
		} else {
			sb.WriteString(fmt.Sprintf("%sSnake head OOB at (%d, %d)%s", opts.indent, snake.Head.X, snake.Head.Y, opts.newlineCharacter))
		}

		// Place snake body
		for _, part := range snake.Body {
			if part != snake.Head { // Skip the head position
				if checkOOB(part.X, part.Y) {
					board[adjustY(part.Y)][part.X] = snakeChar
				} else {
					sb.WriteString(fmt.Sprintf("%sSnake body OOB at (%d, %d)%s", opts.indent, part.X, part.Y, opts.newlineCharacter))
				}
			}
		}
	}

	// Build the string representation of the board
	for _, row := range board {
		sb.WriteString(opts.indent) // Apply indentation for each row
		for _, cell := range row {
			sb.WriteRune(cell)
			sb.WriteRune(' ')
		}
		sb.WriteString(opts.newlineCharacter)
	}

	return sb.String()
}

// Options struct to hold the customizable parameters
type boardOptions struct {
	indent           string
	newlineCharacter string
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

const maxEdges = 499

func GenerateMermaidTree(node *Node, depth int) string {
	edges := 0
	pathsToShow := 5
	return generateMermaidTree(node, depth, &edges, pathsToShow)
}

// generateMermaidTree generates a Mermaid diagram for the top N deepest paths and all root children
func generateMermaidTree(node *Node, depth int, edgeCount *int, topN int) string {
	if node == nil || *edgeCount >= maxEdges {
		return ""
	}

	// Find all paths in the tree
	var paths []Path
	findDeepestPaths(node, []*Node{}, &paths)

	// Sort paths by depth in descending order
	sort.Slice(paths, func(i, j int) bool {
		return paths[i].Depth > paths[j].Depth
	})

	// Select the top N deepest paths
	if len(paths) > topN {
		paths = paths[:topN]
	}

	// Create a set of nodes that are part of the top N deepest paths
	nodeSet := make(map[*Node]bool)
	for _, path := range paths {
		for _, n := range path.Nodes {
			nodeSet[n] = true
		}
	}

	// Include all first-level children of the root node in the nodeSet
	for _, child := range node.Children {
		nodeSet[child] = true
	}

	var sb strings.Builder

	// Start with the root node and add Mermaid config
	if depth == 0 {
		sb.WriteString("graph TD;\n")
	}

	// Generate the diagram for nodes in the nodeSet and all root children
	return generateMermaidTreeForNode(node, depth, edgeCount, nodeSet, &sb)
}

// generateMermaidTreeForNode recursively generates the Mermaid diagram for the nodes in the nodeSet
func generateMermaidTreeForNode(node *Node, depth int, edgeCount *int, nodeSet map[*Node]bool, sb *strings.Builder) string {
	if node == nil || *edgeCount >= maxEdges {
		return ""
	}

	// Skip nodes that aren't part of the top N deepest paths or root children
	if !nodeSet[node] {
		return ""
	}

	// Generate a unique identifier for the node
	nodeID := fmt.Sprintf("Node_%p", node)

	// Node details for root (without UCB value)
	nodeLabel := fmt.Sprintf("Visits: %d<br/>Average Score: %.2f<br/>Untried Moves: %d",
		node.Visits, node.Score/float64(node.Visits), len(node.UntriedMoves))

	// Add the board state using visualizeBoard with <br/> for newlines
	boardVisualization := visualizeBoard(node.Board, WithNewlineCharacter("<br/>"))
	nodeLabel += "<br/>" + boardVisualization

	// Add the Voronoi visualization with <br/> for newlines
	voronoiVisualization := VisualizeVoronoi(GenerateVoronoi(node.Board), node.Board.Snakes, WithNewlineCharacter("<br/>"))
	nodeLabel += "<br/>" + voronoiVisualization

	// Add the node definition
	sb.WriteString(fmt.Sprintf("%s[\"%s\"]\n", nodeID, nodeLabel))

	// Recursively process children
	for _, child := range node.Children {
		if *edgeCount >= maxEdges {
			break
		}

		// Skip children that aren't part of the top N deepest paths or root children
		if !nodeSet[child] {
			continue
		}

		childID := fmt.Sprintf("Node_%p", child)

		// Calculate UCB value for the edge between this node and its child
		ucbValue := node.UCTValue(child)

		// Add the edge between the current node and the child node with UCB value
		sb.WriteString(fmt.Sprintf("%s -->|UCB: %.2f| %s\n", nodeID, ucbValue, childID))
		*edgeCount++ // Increment the edge counter

		// Recursively generate the diagram for the child node
		generateMermaidTreeForNode(child, depth+1, edgeCount, nodeSet, sb)
	}

	return sb.String()
}

// Path represents a path in the tree with its corresponding depth
type Path struct {
	Nodes []*Node
	Depth int
}

// findDeepestPaths recursively finds the deepest paths in the tree
func findDeepestPaths(node *Node, currentPath []*Node, paths *[]Path) {
	if node == nil {
		return
	}

	// Append the current node to the path
	currentPath = append(currentPath, node)

	// If the node has no children, this is a leaf node, so record the path
	if len(node.Children) == 0 {
		*paths = append(*paths, Path{Nodes: append([]*Node(nil), currentPath...), Depth: len(currentPath)})
	} else {
		// Recurse for each child
		for _, child := range node.Children {
			findDeepestPaths(child, currentPath, paths)
		}
	}
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
				sb.WriteString(". ") // Unassigned cells
			} else {
				sb.WriteString(fmt.Sprintf("%c ", 'A'+owner)) // Each snake gets a unique letter
			}
		}
		sb.WriteString(opts.newlineCharacter)
	}

	return sb.String()
}
