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

	// Build the string representation of the board using manual spacing for alignment
	for _, row := range board {
		sb.WriteString(opts.indent)
		for _, cell := range row {
			sb.WriteRune(cell)
			sb.WriteString("  ") // Add extra spacing to simulate a table
		}
		sb.WriteString(opts.newlineCharacter)
	}

	// Append the health status of each snake at the bottom
	sb.WriteString(opts.indent + "Snake Health:")
	sb.WriteString(opts.newlineCharacter)

	for i, snake := range game.Snakes {
		sb.WriteString(fmt.Sprintf("Snake %c: %d", 'a'+i, snake.Health))
		sb.WriteString(opts.newlineCharacter)

	}
	sb.WriteString(opts.newlineCharacter)

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

const maxEdges = 499

func GenerateMostVisitedPathWithAlternativesMermaidTree(node *Node) string {
	edges := 0
	var sb strings.Builder

	// Add Mermaid config for the graph
	sb.WriteString("graph TD;\n")

	// Generate the most visited path with alternatives at each node, starting at depth 0
	generateMostVisitedPathWithAlternatives(node, 0, &edges, &sb)

	return sb.String()
}

// generateMostVisitedPathWithAlternatives generates the Mermaid diagram following the most visited path,
// but shows all alternative children at each node, and fills in content for every node including Voronoi and controlled positions.
func generateMostVisitedPathWithAlternatives(node *Node, depth int, edgeCount *int, sb *strings.Builder) string {
	if node == nil || *edgeCount >= maxEdges {
		return ""
	}

	// Generate a unique identifier for the node
	nodeID := fmt.Sprintf("Node_%p", node)

	// Node label with detailed info
	nodeLabel := fmt.Sprintf("Nodeid: %s<br/>Visits: %d<br/>Average Score: %.3f<br/>Who moved:  %d",
		nodeID, node.Visits, node.Score/float64(node.Visits), node.SnakeIndex)

	// Add the board state using visualizeBoard with <br/> for newlines
	boardVisualization := visualizeBoard(node.Board, WithNewlineCharacter("<br/>"))
	nodeLabel += "<br/>" + boardVisualization

	// Generate Voronoi diagram and count controlled positions for each snake
	voronoi := GenerateVoronoi(node.Board)
	voronoiVisualization := VisualizeVoronoi(voronoi, node.Board.Snakes, WithNewlineCharacter("<br/>"))
	nodeLabel += "<br/>" + voronoiVisualization

	// Count controlled positions by each snake (A, B, etc.)
	controlledPositions := make([]int, len(node.Board.Snakes))
	for _, row := range voronoi {
		for _, owner := range row {
			if owner >= 0 && owner < len(controlledPositions) {
				controlledPositions[owner]++
			}
		}
	}
	for i, count := range controlledPositions {
		nodeLabel += fmt.Sprintf("Snake %c controls: %d positions<br/>", 'A'+i, count)
	}

	// Add the node definition with full details
	sb.WriteString(fmt.Sprintf("%s[\"%s\"]\n", nodeID, nodeLabel))

	// Sort children by visit count, descending
	sort.Slice(node.Children, func(i, j int) bool {
		return node.Children[i].Visits > node.Children[j].Visits
	})

	// If there are no children, return
	if len(node.Children) == 0 {
		return sb.String()
	}

	// Show all children as alternative branches with full content
	for _, child := range node.Children {
		if *edgeCount >= maxEdges {
			break
		}

		childID := fmt.Sprintf("Node_%p", child)

		// Child node label with detailed info
		childLabel := fmt.Sprintf("Nodeid: %s<br/>Visits: %d<br/>Average Score: %.3f<br/>",
			childID, child.Visits, child.Score/float64(child.Visits))

		// Add the board state using visualizeBoard with <br/> for newlines
		childBoardVisualization := visualizeBoard(child.Board, WithNewlineCharacter("<br/>"))
		childLabel += "<br/>" + childBoardVisualization

		// Generate Voronoi diagram and count controlled positions for each snake in the child
		childVoronoi := GenerateVoronoi(child.Board)
		childVoronoiVisualization := VisualizeVoronoi(childVoronoi, child.Board.Snakes, WithNewlineCharacter("<br/>"))
		childLabel += "<br/>" + childVoronoiVisualization

		// Count controlled positions for each snake in the child
		childControlledPositions := make([]int, len(child.Board.Snakes))
		for _, row := range childVoronoi {
			for _, owner := range row {
				if owner >= 0 && owner < len(childControlledPositions) {
					childControlledPositions[owner]++
				}
			}
		}
		for i, count := range childControlledPositions {
			childLabel += fmt.Sprintf("Snake %c controls: %d positions<br/>", 'A'+i, count)
		}

		// Calculate UCB value for the edge between this node and its child
		ucbValue := child.UCT(node, 1.41)

		// Add the edge between the current node and the child node with UCB value
		// TODO: add back ucb as method
		sb.WriteString(fmt.Sprintf("%s -->|UCB: %.2f| %s\n", nodeID, ucbValue, childID))

		// Add the child node's content to the diagram
		sb.WriteString(fmt.Sprintf("%s[\"%s\"]\n", childID, childLabel))

		*edgeCount++ // Increment the edge counter
	}

	// Follow the most visited child for the primary path
	mostVisitedChild := node.Children[0]

	// Recursively generate the diagram for the most visited child
	return generateMostVisitedPathWithAlternatives(mostVisitedChild, depth+1, edgeCount, sb)
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
				sb.WriteString(". ") // Unassigned cells
			} else {
				sb.WriteString(fmt.Sprintf("%c ", 'A'+owner)) // Each snake gets a unique letter
			}
		}
		sb.WriteString(opts.newlineCharacter)
	}

	return sb.String()
}

// func GenerateMostVisitedPathWithAlternativesHtmlTreeOld(node *Node) error {
// 	timestamp := time.Now().Format("20060102_150405.000000")
// 	uuid := uuid.New().String()
// 	filename := filepath.Join("movetrees", fmt.Sprintf("%s_%s.html", timestamp, uuid))

// 	// Generate DOT structure for Graphviz, pruning to most visited path + direct neighbors
// 	dotData := generatePrunedDotTreeData(node)

// 	board, err := json.Marshal(node.Board)
// 	if err != nil {
// 		return err
// 	}

// 	// Write the HTML file with embedded viz.js for visualization
// 	htmlContent := fmt.Sprintf(`
// 	<!DOCTYPE html>
// 	<html lang="en">
// 	<head>
// 		<meta charset="UTF-8">
// 		<meta name="viewport" content="width=device-width, initial-scale=1.0">
// 		<title>Most Visited Path Tree</title>
// 		<script src="https://cdnjs.cloudflare.com/ajax/libs/viz.js/2.1.2/viz.js"></script>
// 		<script src="https://cdnjs.cloudflare.com/ajax/libs/viz.js/2.1.2/full.render.js"></script>
// 	</head>
// 	<body>
// 		<div>%s</div>
// 		<div id="graph" style="width:100vw; height:100vh;"></div>
// 		<script>
// 			const dot = %q;

// 			// Render the DOT content using viz.js
// 			const viz = new Viz();
// 			viz.renderSVGElement(dot)
// 				.then(function(element) {
// 					document.getElementById('graph').appendChild(element);
// 				})
// 				.catch(error => {
// 					console.error("Error rendering DOT:", error);
// 				});
// 		</script>
// 	</body>
// 	</html>`, string(board), dotData)

// 	// Write the HTML file to disk
// 	err = os.WriteFile(filename, []byte(htmlContent), 0644)
// 	if err != nil {
// 		return fmt.Errorf("failed to write file: %w", err)
// 	}

// 	fmt.Printf("Generated move tree: %s\nFile: %s\n", uuid, filepath.Join(".", filename))
// 	return nil
// }

// // visualizeNode generates the DOT representation of a single node, including its label, visits, score, board state, and controlled positions
// func visualizeNode(node *Node) string {
// 	if node == nil {
// 		return ""
// 	}

// 	nodeID := fmt.Sprintf("Node_%p", node)
// 	nodeLabel := fmt.Sprintf("Nodeid: %s\nVisits: %d\nAvg Score: %.3f\nSnake moved: %d",
// 		nodeID, node.Visits, node.Score/float64(node.Visits), node.SnakeIndex)

// 	// Add the board state visualization
// 	boardVisualization := visualizeBoard(node.Board, WithNewlineCharacter("<br/>"))
// 	nodeLabel += "\n" + boardVisualization

// 	// Add controlled positions from the Voronoi diagram
// 	voronoi := GenerateVoronoi(node.Board)
// 	nodeVoronoiVisualization := VisualizeVoronoi(voronoi, node.Board.Snakes, WithNewlineCharacter("\n"))
// 	nodeLabel += "\n" + nodeVoronoiVisualization
// 	controlledPositions := make([]int, len(node.Board.Snakes))
// 	for _, row := range voronoi {
// 		for _, owner := range row {
// 			if owner >= 0 && owner < len(controlledPositions) {
// 				controlledPositions[owner]++
// 			}
// 		}
// 	}
// 	for i, count := range controlledPositions {
// 		nodeLabel += fmt.Sprintf("\nSnake %c controls: %d positions", 'A'+i, count)
// 	}

// 	return fmt.Sprintf("  %s [label=\"%s\", fontname=\"Courier\"];\n", nodeID, nodeLabel)
// }

// visualizeNode generates the DOT representation of a single node, including its label, visits, score, board state, and controlled positions
func visualizeNode(node *Node) string {
	if node == nil {
		return ""
	}

	nodeID := fmt.Sprintf("Node_%p", node)
	// Using <br/> instead of \n to create HTML-based line breaks that D3 can interpret
	nodeLabel := fmt.Sprintf("%s\nVisits: %d\nAvg Score: %.3f\nSnake moved: %d\n\n",
		nodeID, node.Visits, node.Score/float64(node.Visits), node.SnakeIndex)

	// Add the board state visualization
	boardVisualization := visualizeBoard(node.Board, WithNewlineCharacter("\n"))
	nodeLabel += "\n" + boardVisualization

	// Add controlled positions from the Voronoi diagram
	voronoi := GenerateVoronoi(node.Board)
	nodeVoronoiVisualization := VisualizeVoronoi(voronoi, node.Board.Snakes, WithNewlineCharacter("\n"))
	nodeLabel += "\n" + nodeVoronoiVisualization
	controlledPositions := make([]int, len(node.Board.Snakes))
	for _, row := range voronoi {
		for _, owner := range row {
			if owner >= 0 && owner < len(controlledPositions) {
				controlledPositions[owner]++
			}
		}
	}
	for i, count := range controlledPositions {
		nodeLabel += fmt.Sprintf("\nSnake %c controls: %d positions", 'A'+i, count)
	}

	// Return the node label with HTML line breaks
	return nodeLabel
}

// // generatePrunedDotTreeData generates the pruned DOT data for Graphviz, including only the most visited path and direct neighbors
// func generatePrunedDotTreeData(node *Node) string {
// 	if node == nil {
// 		return ""
// 	}

// 	var sb strings.Builder
// 	sb.WriteString("digraph G {\n")
// 	sb.WriteString("  rankdir=\"TB\";\n") // Top to Bottom layout
// 	sb.WriteString("  node [ shape=\"box\" style=\"rounded,filled\" fontname=\"Lato\" margin=0.2 ]\n")
// 	sb.WriteString("  edge [ fontname=\"Lato\" ]\n")

// 	// Generate the pruned tree, focusing on the most visited path and direct neighbors
// 	// traversePrunedDotTree(node, &sb)
// 	traverseMostVisitedPaths(node, &sb)

// 	sb.WriteString("}\n")
// 	return sb.String()
// }

// traverseMostVisitedPaths traverses the most visited path starting from each of the first-layer children.
func traverseMostVisitedPaths(root *Node, sb *strings.Builder) {
	if root == nil {
		return
	}

	// Use the visualizeNode function to add the root node's representation to the DOT data
	sb.WriteString(visualizeNode(root))

	// Sort children of the root node by visit count, descending
	sort.Slice(root.Children, func(i, j int) bool {
		return root.Children[i].Visits > root.Children[j].Visits
	})

	// For each child in the first layer of nodes
	for _, firstLayerChild := range root.Children {
		// Visualize the first layer child
		sb.WriteString(visualizeNode(firstLayerChild))

		// Add the edge between the root node and the first layer child
		childID := fmt.Sprintf("Node_%p", firstLayerChild)
		rootID := fmt.Sprintf("Node_%p", root)
		ucbValue := firstLayerChild.UCT(root, 1.41)
		sb.WriteString(fmt.Sprintf("  %s -> %s [label=\"UCB: %.5f\"];\n", rootID, childID, ucbValue))

		// Recursively traverse the most visited path starting from this child
		traverseMostVisitedPaths(firstLayerChild, sb)
	}
}

// traverseMostVisitedPath recursively generates the most visited path from a given node in DOT format
func traverseMostVisitedPath(node *Node, sb *strings.Builder) {
	if node == nil {
		return
	}

	// Sort children by visit count, descending
	sort.Slice(node.Children, func(i, j int) bool {
		return node.Children[i].Visits > node.Children[j].Visits
	})

	// If the node has children, follow the most visited path
	if len(node.Children) > 0 {
		// Add direct neighbors (children) as nodes using visualizeNode
		for _, child := range node.Children {
			// Visualize the child node
			sb.WriteString(visualizeNode(child))

			// Add the edge between the current node and the child node
			childID := fmt.Sprintf("Node_%p", child)
			nodeID := fmt.Sprintf("Node_%p", node)
			ucbValue := child.UCT(node, 1.41)
			sb.WriteString(fmt.Sprintf("  %s -> %s [label=\"UCB: %.5f\"];\n", nodeID, childID, ucbValue))
		}

		// Recursively process only the most visited child
		mostVisitedChild := node.Children[0]

		// Add rank=same to ensure the most visited node is in the center
		sb.WriteString(fmt.Sprintf("{ rank=same; %s; }\n", fmt.Sprintf("Node_%p", mostVisitedChild)))
		traverseMostVisitedPath(mostVisitedChild, sb)
	}
}

type TreeNode struct {
	ID            string      `json:"id"`
	Visits        int         `json:"visits"`
	AverageScore  float64     `json:"avg_score"`
	UCB           float64     `json:"ucb"`
	IsMostVisited bool        `json:"isMostVisited"`
	Children      []*TreeNode `json:"children"`
	Body          string      `json:"body"`
}

func GenerateMostVisitedPathWithAlternativesHtmlTree(node *Node) error {

	treeNode := generateTreeData(node)
	timestamp := time.Now().Format("20060102_150405.000000")
	uuid := uuid.New().String()
	filename := filepath.Join("visualiser", "tree-data", fmt.Sprintf("%s_%s.json", timestamp, uuid))

	// // Parse the HTML template
	// tmpl, err := template.ParseFiles("template.html")
	// if err != nil {
	// 	return err
	// }

	// Create the output file
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// // Execute the template with the treeData
	// data := struct {
	// 	TreeData string
	// }{
	// 	TreeData: string(treeData),
	// }

	// Generate the tree data in JSON format
	err = json.NewEncoder(file).Encode(treeNode)
	if err != nil {
		return err
	}

	// err = tmpl.Execute(file, data)
	// if err != nil {
	// 	return fmt.Errorf("failed to write template: %w", err)
	// }

	fmt.Printf("Generated move tree: %s\nFile: %s\n", uuid, filepath.Join(".", filename))
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
		}

		treeNode.Children = append(treeNode.Children, childNode)

		// Recur only on the most visited child
		// if i == 0 {
		traverseAndBuildTree(child, childNode)
		// }
	}
}
