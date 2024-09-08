package main

import (
	"context"
	"encoding/json"
	"math"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type SelectChildTestCase struct {
	Description   string
	Parent        *Node
	ExpectedChild *Node
}

func TestSelectChild(t *testing.T) {
	testCases := []SelectChildTestCase{
		{
			Description: "Select child with highest UCT value - single child",
			Parent: func() *Node {
				parent := &Node{Visits: 10}
				child := &Node{Visits: 1, Score: 1.0, Parent: parent}
				parent.Children = append(parent.Children, child)
				return parent
			}(),
			ExpectedChild: func() *Node {
				return &Node{Visits: 1, Score: 1.0}
			}(),
		},
		{
			Description: "Select child with highest UCT value - two children",
			Parent: func() *Node {
				parent := &Node{Visits: 20}
				child1 := &Node{Visits: 5, Score: 3.0, Parent: parent}
				child2 := &Node{Visits: 10, Score: 6.0, Parent: parent}
				parent.Children = append(parent.Children, child1, child2)
				return parent
			}(),
			ExpectedChild: func() *Node {
				return &Node{Visits: 5, Score: 3.0}
			}(),
		},
		{
			Description: "Select child when UCT values are equal",
			Parent: func() *Node {
				parent := &Node{Visits: 30}
				child1 := &Node{Visits: 10, Score: 5.0, Parent: parent}
				child2 := &Node{Visits: 10, Score: 5.0, Parent: parent}
				parent.Children = append(parent.Children, child1, child2)
				return parent
			}(),
			ExpectedChild: func() *Node {
				return &Node{Visits: 10, Score: 5.0}
			}(),
		},
		{
			Description: "Select child when parent has no visits",
			Parent: func() *Node {
				parent := &Node{Visits: 0}
				child1 := &Node{Visits: 5, Score: 3.0, Parent: parent}
				child2 := &Node{Visits: 10, Score: 6.0, Parent: parent}
				parent.Children = append(parent.Children, child1, child2)
				return parent
			}(),
			ExpectedChild: func() *Node {
				// function set to select first node although this is not critical
				return &Node{Visits: 5, Score: 3.0}
			}(),
		},
		{
			Description: "Select child when one child has never been visited",
			Parent: func() *Node {
				parent := &Node{Visits: 50}
				child1 := &Node{Visits: 25, Score: 12.0, Parent: parent}
				child2 := &Node{Visits: 0, Score: 0.0, Parent: parent}
				parent.Children = append(parent.Children, child1, child2)
				return parent
			}(),
			ExpectedChild: func() *Node {
				return &Node{Visits: 0, Score: 0.0}
			}(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Description, func(t *testing.T) {
			selectedChild := tc.Parent.Children[0]

			if !assert.NotNil(t, selectedChild, "selected child was nil") {
				return
			}

			// Check if the selected child is the expected one by comparing specific fields
			assert.Equal(t, tc.ExpectedChild.Visits, selectedChild.Visits, "Expected child visits do not match")
			assert.Equal(t, tc.ExpectedChild.Score, selectedChild.Score, "Expected child score does not match")
		})
	}
}

type ExpandTestCase struct {
	Description      string
	InitialNode      *Node
	ExpectedChildren int
	ExpectedMoves    []Direction
}

// func TestExpand(t *testing.T) {
// 	testCases := []ExpandTestCase{
// 		{
// 			Description: "Expand with one untried move",
// 			InitialNode: func() *Node {
// 				board := Board{
// 					Height: 5, Width: 5,
// 					Snakes: []Snake{
// 						{ID: "snake1", Head: Point{X: 2, Y: 2}},
// 					},
// 				}
// 				node := NewNode(board, 0)
// 				// node.UntriedMoves = []Direction{Up}
// 				return node
// 			}(),
// 			ExpectedChildren: 1,
// 			ExpectedMoves:    []Direction{Up},
// 		},
// 		{
// 			Description: "Expand with multiple untried moves for one snake",
// 			InitialNode: func() *Node {
// 				board := Board{
// 					Height: 5, Width: 5,
// 					Snakes: []Snake{
// 						{ID: "snake1", Head: Point{X: 2, Y: 2}},
// 					},
// 				}
// 				node := NewNode(board, 0)
// 				// node.UntriedMoves = []Direction{Up, Down, Left, Right}
// 				return node
// 			}(),
// 			ExpectedChildren: 4,
// 			ExpectedMoves:    []Direction{Up, Down, Left, Right},
// 		},
// 		{
// 			Description: "Expand with two snakes and one untried move for each",
// 			InitialNode: func() *Node {
// 				board := Board{
// 					Height: 5, Width: 5,
// 					Snakes: []Snake{
// 						{ID: "snake1", Head: Point{X: 2, Y: 2}},
// 						{ID: "snake2", Head: Point{X: 3, Y: 3}},
// 					},
// 				}
// 				node := NewNode(board, 0)
// 				// node.UntriedMoves = []Direction{Up, Left} // For snake 1
// 				return node
// 			}(),
// 			ExpectedChildren: 2,
// 			ExpectedMoves:    []Direction{Up, Left},
// 		},
// 		{
// 			Description: "Expand when no untried moves remain",
// 			InitialNode: func() *Node {
// 				board := Board{
// 					Height: 5, Width: 5,
// 					Snakes: []Snake{
// 						{ID: "snake1", Head: Point{X: 2, Y: 2}},
// 					},
// 				}
// 				node := NewNode(board, 0)
// 				// node.UntriedMoves = []Direction{}
// 				return node
// 			}(),
// 			ExpectedChildren: 0,
// 			ExpectedMoves:    []Direction{},
// 		},
// 		{
// 			Description: "Expand with a snake at the board edge",
// 			InitialNode: func() *Node {
// 				board := Board{
// 					Height: 5, Width: 5,
// 					Snakes: []Snake{
// 						{ID: "snake1", Head: Point{X: 4, Y: 4}}, // At the bottom-right corner
// 					},
// 				}
// 				node := NewNode(board, 0)
// 				// node.UntriedMoves = []Direction{Up, Left}
// 				return node
// 			}(),
// 			ExpectedChildren: 2,
// 			ExpectedMoves:    []Direction{Up, Left},
// 		},
// 		{
// 			Description: "Expand with multiple snakes and hazard on the board",
// 			InitialNode: func() *Node {
// 				board := Board{
// 					Height: 5, Width: 5,
// 					Hazards: []Point{
// 						{X: 2, Y: 2},
// 					},
// 					Snakes: []Snake{
// 						{ID: "snake1", Head: Point{X: 1, Y: 1}},
// 						{ID: "snake2", Head: Point{X: 3, Y: 3}},
// 					},
// 				}
// 				node := NewNode(board, 0)
// 				// node.UntriedMoves = []Direction{Right, Down} // For snake 1
// 				return node
// 			}(),
// 			ExpectedChildren: 2,
// 			ExpectedMoves:    []Direction{Right, Down},
// 		},
// 	}

// 	for _, tc := range testCases {
// 		t.Run(tc.Description, func(t *testing.T) {
// 			children := tc.InitialNode.Expand()

// 			// Verify the number of children
// 			require.Equal(t, tc.ExpectedChildren, len(children), "Number of children nodes does not match")

// 			// Ensure UntriedMoves is cleared after expansion
// 			require.Equal(t, 0, len(tc.InitialNode.UntriedMoves), "Untried moves should be cleared after expansion")

// 			for i, child := range children {
// 				require.NotNil(t, child, "Child node should not be nil")

// 				// Generate the expected board state
// 				expectedBoard := copyBoard(tc.InitialNode.Board)
// 				applyMove(&expectedBoard, tc.InitialNode.SnakeIndex, tc.ExpectedMoves[i])

// 				// Compare the actual child board with the expected board state
// 				if !assert.Equal(t, expectedBoard, child.Board, "The child's board state does not match the expected state after applying the move") {
// 					t.Logf("Test failed on case: %s", tc.Description)
// 					t.Logf("Expected Move: %+v", tc.ExpectedMoves[i])
// 					t.Logf("Expected Board: %+v", expectedBoard)
// 					t.Logf("Actual Board: %+v", child.Board)
// 				}

// 				// Verify that the parent of the child node is the initial node
// 				assert.Equal(t, tc.InitialNode, child.Parent, "Child node's parent should be the initial node")
// 			}
// 		})
// 	}
// }

type ApplyMoveTestCase struct {
	Description   string
	InitialBoard  Board
	Move          Direction
	SnakeIndex    int
	ExpectedBoard Board
}

func TestMCTSVisualization(t *testing.T) {
	testCases := []struct {
		Description  string
		InitialBoard Board
		Iterations   int
	}{
		{
			Description: "MCTS with multiple snakes and more iterations",
			InitialBoard: Board{
				Height: 7,
				Width:  7,
				Snakes: []Snake{
					{ID: "snake1", Head: Point{X: 1, Y: 1}, Health: 100, Body: []Point{{X: 1, Y: 1}, {X: 1, Y: 0}}},
					{ID: "snake2", Head: Point{X: 5, Y: 5}, Health: 100, Body: []Point{{X: 5, Y: 5}, {X: 5, Y: 6}}},
				},
				Food: []Point{{X: 3, Y: 3}},
			},
			Iterations: math.MaxInt,
			// Iterations: 1000,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Description, func(t *testing.T) {

			numCPUs := runtime.NumCPU()
			_ = numCPUs
			rootBoard := copyBoard(tc.InitialBoard)
			ctx, _ := context.WithTimeout(context.Background(), 1000*time.Millisecond)
			// node := MCTS(ctx, rootBoard, tc.Iterations, numCPUs)
			node := MCTS(ctx, rootBoard, tc.Iterations)

			require.NotNil(t, node, "node is nil")

			// assert.NoError(t, writeNodeAsMermaidToHTMLFile(node))
			assert.NoError(t, GenerateMostVisitedPathWithAlternativesHtmlTree(node))

		})
	}
}

func TestMCTSVisualizationJSON(t *testing.T) {
	testCases := []struct {
		Description  string
		InitialBoard string
		Iterations   int
	}{
		{
			Description:  "should not get transfixed by death",
			InitialBoard: `{"height":11,"width":11,"food":[{"x":0,"y":4},{"x":1,"y":4}],"hazards":[],"snakes":[{"id":"5baad214-ed5e-4794-bd30-f110e488c474","name":"mcts","health":97,"body":[{"x":3,"y":4},{"x":4,"y":4},{"x":4,"y":5},{"x":5,"y":5},{"x":6,"y":5}],"latency":"405","head":{"x":3,"y":4},"shout":"","customizations":{"color":"#888888","head":"default","tail":"default"}},{"id":"e55aa73a-a108-406c-af9e-9192a380c027","name":"soba","health":89,"body":[{"x":2,"y":5},{"x":2,"y":6},{"x":3,"y":6},{"x":4,"y":6}],"latency":"400","head":{"x":2,"y":5},"shout":"","customizations":{"color":"#118645","head":"replit-mark","tail":"replit-notmark"}}]}`,
			Iterations:   math.MaxInt,
		},
		// {
		// 	Description:  "should not butt heads",
		// 	InitialBoard: `{"height":11,"width":11,"food":[{"x":4,"y":0},{"x":7,"y":4},{"x":9,"y":3},{"x":0,"y":4}],"hazards":[],"snakes":[{"id":"a82fcde3-2bed-4cc5-ac42-a19cc10175ca","name":"mcts","health":66,"body":[{"x":1,"y":9},{"x":0,"y":9},{"x":0,"y":8},{"x":0,"y":7}],"latency":"902","head":{"x":1,"y":9},"shout":"","customizations":{"color":"#888888","head":"default","tail":"default"}},{"id":"4a147cce-14d9-42ba-b5b2-e72b2ecf04a7","name":"soba","health":93,"body":[{"x":3,"y":9},{"x":3,"y":8},{"x":4,"y":8},{"x":5,"y":8},{"x":6,"y":8}],"latency":"401","head":{"x":3,"y":9},"shout":"","customizations":{"color":"#118645","head":"replit-mark","tail":"replit-notmark"}}]}`,
		// 	Iterations:   math.MaxInt,
		// },
	}

	for _, tc := range testCases {
		t.Run(tc.Description, func(t *testing.T) {

			numCPUs := runtime.NumCPU()
			_ = numCPUs
			var board Board
			assert.NoError(t, json.Unmarshal([]byte(tc.InitialBoard), &board))
			rootBoard := copyBoard(board)
			ctx, _ := context.WithTimeout(context.Background(), 1000*time.Millisecond)
			// node := MCTS(ctx, rootBoard, tc.Iterations, numCPUs)
			// for i := 0; i < tc.Iterations; i++ {

			node := MCTS(ctx, rootBoard, tc.Iterations)
			require.NotNil(t, node, "node is nil")

			// assert.NoError(t, writeNodeAsMermaidToHTMLFile(node))
			assert.NoError(t, GenerateMostVisitedPathWithAlternativesHtmlTree(node))
			// }

		})
	}
}
