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
		move:             nil,
	}

	// Apply any options provided
	for _, opt := range options {
		opt(opts)
	}

	// Ensure board dimensions are positive
	if game.Height <= 0 || game.Width <= 0 {
		return opts.indent + "Invalid board dimensions"
	}

	// Display moves at the top if provided
	if opts.move != nil {
		sb.WriteString(opts.indent)
		for i, direction := range opts.move {
			if i >= len(game.Snakes) {
				continue
			}
			snakeChar := rune('a' + i)
			var arrow rune
			switch direction {
			case Up:
				arrow = '↑'
			case Down:
				arrow = '↓'
			case Left:
				arrow = '←'
			case Right:
				arrow = '→'
			}
			sb.WriteRune(snakeChar)
			sb.WriteRune(arrow)
			sb.WriteRune(' ')
		}
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

	// Function to adjust the Y coordinate to match the expected orientation
	adjustY := func(y int) int {
		return extendedHeight - 1 - (y + 1)
	}

	// Place food on the board
	for _, food := range game.Food {
		board[adjustY(food.Y)][food.X+1] = '🍎'
	}

	// Place hazards on the board
	for _, hazard := range game.Hazards {
		board[adjustY(hazard.Y)][hazard.X+1] = 'H'
	}

	// Place snakes on the board
	for snakeIndex, snake := range game.Snakes {
		snakeChar := rune('a' + snakeIndex)
		if snakeChar > 'z' {
			snakeChar = '?' // Fallback in case of too many snakes
		}

		board[adjustY(snake.Head.Y)][snake.Head.X+1] = unicode.ToUpper(snakeChar)
		for _, part := range snake.Body[1:] {
			board[adjustY(part.Y)][part.X+1] = snakeChar
		}
	}

	// Overlay arrows for moves if the option is provided
	if opts.move != nil {
		for i, direction := range opts.move {
			if i >= len(game.Snakes) {
				continue
			}
			snake := game.Snakes[i]
			newHead := moveHead(snake.Head, direction)

			var arrow rune
			switch direction {
			case Up:
				arrow = '↑'
			case Down:
				arrow = '↓'
			case Left:
				arrow = '←'
			case Right:
				arrow = '→'
			}

			if checkOOB(newHead.X, newHead.Y, game.Width, game.Height) {
				// Place the arrow on the boundary 'x' if the move is out of bounds
				switch direction {
				case Up:
					board[adjustY(game.Height)][snake.Head.X+1] = arrow
				case Down:
					board[adjustY(-1)][snake.Head.X+1] = arrow
				case Left:
					board[adjustY(snake.Head.Y)][0] = arrow
				case Right:
					board[adjustY(snake.Head.Y)][extendedWidth-1] = arrow
				}
			} else {
				board[adjustY(newHead.Y)][newHead.X+1] = arrow
			}
		}
	}

	// Build the string representation of the board
	for _, row := range board {
		sb.WriteString(opts.indent)
		for _, cell := range row {
			sb.WriteRune(cell)
			sb.WriteRune(' ')
		}
		sb.WriteString(opts.newlineCharacter)
	}

	return sb.String()
}

// Helper function to check for out-of-bounds errors
func checkOOB(x, y, width, height int) bool {
	return x < 0 || x >= width || y < 0 || y >= height
}

// Helper function to mark the boundary when an arrow exceeds the board's dimensions
func markBoundary(overlay [][]rune, x, y int, move Direction, width, height int) {
	switch move {
	case Up:
		if y < height-1 {
			overlay[0][x] = '↑'
		} else {
			overlay[height-1][x] = 'x'
		}
	case Down:
		if y > 0 {
			overlay[height-1][x] = '↓'
		} else {
			overlay[0][x] = 'x'
		}
	case Left:
		if x > 0 {
			overlay[y][0] = '←'
		} else {
			overlay[y][width-1] = 'x'
		}
	case Right:
		if x < width-1 {
			overlay[y][width-1] = '→'
		} else {
			overlay[y][0] = 'x'
		}
	}
}

// Helper function to extend the board with 'x' characters when an arrow exceeds the board's dimensions
func extendBoardWithArrows(board *[][]rune, x, y, width, height int) {
	if y >= height {
		*board = append(*board, make([]rune, width))
		height++
	}
	for i := 0; i < height; i++ {
		if x >= width {
			(*board)[i] = append((*board)[i], 'x')
		}
		if y >= height {
			(*board)[y] = append((*board)[y], 'x')
		}
	}
}

// Options struct to hold the customizable parameters
type boardOptions struct {
	indent           string
	newlineCharacter string
	move             Move
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

func WithMove(move Move) func(*boardOptions) {
	return func(o *boardOptions) {
		o.move = move
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
