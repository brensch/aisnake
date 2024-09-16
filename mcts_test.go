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
			workers := runtime.NumCPU()
			node := MCTS(ctx, "testid", rootBoard, tc.Iterations, 2*workers, make(map[string]*Node))

			require.NotNil(t, node, "node is nil")

			// assert.NoError(t, writeNodeAsMermaidToHTMLFile(node))
			assert.NoError(t, GenerateMostVisitedPathWithAlternativesHtmlTree(node))

		})
	}
}

// requires that snake in question is at index 0
func TestMCTSVisualizationJSON(t *testing.T) {
	testCases := []struct {
		Description     string
		InitialBoard    string
		Iterations      int
		AcceptableMoves []string
	}{
		// {
		// 	Description:     "don't go into corner. non negotiable.",
		// 	InitialBoard:    `{"height":11,"width":11,"food":[{"x":7,"y":9},{"x":1,"y":8},{"x":10,"y":10},{"x":9,"y":9},{"x":4,"y":0},{"x":2,"y":0},{"x":5,"y":9},{"x":7,"y":1},{"x":2,"y":4},{"x":3,"y":7},{"x":0,"y":9}],"hazards":[],"snakes":[{"id":"708f16b8-783d-465a-b45e-c7000f4c9cea","name":"mcts","health":90,"body":[{"x":1,"y":0},{"x":1,"y":1},{"x":0,"y":1},{"x":0,"y":2},{"x":1,"y":2}],"latency":"400","head":{"x":1,"y":0},"shout":"","customizations":{"color":"#888888","head":"default","tail":"default"}},{"id":"a4e16294-0082-4064-8ae2-12ed0a5774a0","name":"soba","health":88,"body":[{"x":6,"y":3},{"x":6,"y":4},{"x":7,"y":4},{"x":7,"y":5},{"x":8,"y":5},{"x":8,"y":6},{"x":7,"y":6},{"x":6,"y":6},{"x":5,"y":6},{"x":5,"y":7}],"latency":"400","head":{"x":6,"y":3},"shout":"","customizations":{"color":"#118645","head":"replit-mark","tail":"replit-notmark"}}]}`,
		// 	Iterations:      math.MaxInt,
		// 	AcceptableMoves: []string{"right"},
		// },

		// {
		// 	Description:     "should not get transfixed by death",
		// 	InitialBoard:    `{"height":11,"width":11,"food":[{"x":0,"y":4},{"x":1,"y":4}],"hazards":[],"snakes":[{"id":"5baad214-ed5e-4794-bd30-f110e488c474","name":"mcts","health":97,"body":[{"x":3,"y":4},{"x":4,"y":4},{"x":4,"y":5},{"x":5,"y":5},{"x":6,"y":5}],"latency":"405","head":{"x":3,"y":4},"shout":"","customizations":{"color":"#888888","head":"default","tail":"default"}},{"id":"e55aa73a-a108-406c-af9e-9192a380c027","name":"soba","health":89,"body":[{"x":2,"y":5},{"x":2,"y":6},{"x":3,"y":6},{"x":4,"y":6}],"latency":"400","head":{"x":2,"y":5},"shout":"","customizations":{"color":"#118645","head":"replit-mark","tail":"replit-notmark"}}]}`,
		// 	Iterations:      math.MaxInt,
		// 	AcceptableMoves: []string{"left", "down"},
		// },
		// {
		// 	Description:     "should not butt heads",
		// 	InitialBoard:    `{"height":11,"width":11,"food":[{"x":4,"y":0},{"x":7,"y":4},{"x":9,"y":3},{"x":0,"y":4}],"hazards":[],"snakes":[{"id":"a82fcde3-2bed-4cc5-ac42-a19cc10175ca","name":"mcts","health":66,"body":[{"x":1,"y":9},{"x":0,"y":9},{"x":0,"y":8},{"x":0,"y":7}],"latency":"902","head":{"x":1,"y":9},"shout":"","customizations":{"color":"#888888","head":"default","tail":"default"}},{"id":"4a147cce-14d9-42ba-b5b2-e72b2ecf04a7","name":"soba","health":93,"body":[{"x":3,"y":9},{"x":3,"y":8},{"x":4,"y":8},{"x":5,"y":8},{"x":6,"y":8}],"latency":"401","head":{"x":3,"y":9},"shout":"","customizations":{"color":"#118645","head":"replit-mark","tail":"replit-notmark"}}]}`,
		// 	Iterations:      math.MaxInt,
		// 	AcceptableMoves: []string{"down"},
		// },
		// {
		// 	Description:     "should not go into corner",
		// 	InitialBoard:    `{"height":11,"width":11,"food":[{"x":5,"y":5},{"x":0,"y":2},{"x":1,"y":2},{"x":6,"y":1},{"x":8,"y":3},{"x":7,"y":4}],"hazards":[],"snakes":[{"id":"732e98bd-90f7-4c74-bb0d-08a59c3d1604","name":"mcts","health":88,"body":[{"x":9,"y":10},{"x":9,"y":9},{"x":10,"y":9},{"x":10,"y":8},{"x":10,"y":7}],"latency":"902","head":{"x":9,"y":10},"shout":"","customizations":{"color":"#888888","head":"default","tail":"default"}},{"id":"f9b45e5b-af6a-47f0-9bcb-7b78f7caa534","name":"soba","health":89,"body":[{"x":1,"y":10},{"x":0,"y":10},{"x":0,"y":9},{"x":1,"y":9},{"x":2,"y":9},{"x":3,"y":9}],"latency":"401","head":{"x":1,"y":10},"shout":"","customizations":{"color":"#118645","head":"replit-mark","tail":"replit-notmark"}}]}`,
		// 	Iterations:      math.MaxInt,
		// 	AcceptableMoves: []string{"left"},
		// },
		// {
		// 	Description:     "don't pass through yourself",
		// 	InitialBoard:    `{"height":11,"width":11,"food":[{"x":2,"y":9}],"hazards":[],"snakes":[{"id":"0238ebfc-896f-4cd8-a132-3b06a92a3b02","name":"mcts","health":97,"body":[{"x":10,"y":9},{"x":9,"y":9},{"x":9,"y":8},{"x":9,"y":7},{"x":10,"y":7},{"x":10,"y":6},{"x":10,"y":5}],"latency":"907","head":{"x":10,"y":9},"shout":"","customizations":{"color":"#888888","head":"default","tail":"default"}},{"id":"983869f1-491d-4323-951d-dc822d7cc787","name":"soba","health":61,"body":[{"x":3,"y":6},{"x":4,"y":6},{"x":5,"y":6},{"x":5,"y":5},{"x":5,"y":4}],"latency":"400","head":{"x":3,"y":6},"shout":"","customizations":{"color":"#118645","head":"replit-mark","tail":"replit-notmark"}}]}`,
		// 	Iterations:      math.MaxInt,
		// 	AcceptableMoves: []string{"up"},
		// },
		// {
		// 	Description:     "should down, counter intuitive (seems like left but will get boxed out of top. may need further investigation)",
		// 	InitialBoard:    `{"height":11,"width":11,"food":[{"x":0,"y":0},{"x":1,"y":3},{"x":2,"y":3},{"x":1,"y":4},{"x":4,"y":9}],"hazards":[],"snakes":[{"id":"a5053727-e14f-43aa-ba60-8bb43c54610c","name":"mcts","health":77,"body":[{"x":8,"y":5},{"x":8,"y":6},{"x":9,"y":6},{"x":9,"y":7},{"x":9,"y":8},{"x":9,"y":9},{"x":9,"y":10},{"x":10,"y":10},{"x":10,"y":9},{"x":10,"y":8},{"x":10,"y":7}],"latency":"904","head":{"x":8,"y":5},"shout":"","customizations":{"color":"#888888","head":"default","tail":"default"}},{"id":"6653a691-0d7e-4f0f-a9ed-e430e87b003d","name":"soba","health":76,"body":[{"x":5,"y":6},{"x":6,"y":6},{"x":6,"y":7},{"x":5,"y":7},{"x":4,"y":7},{"x":3,"y":7},{"x":3,"y":6},{"x":2,"y":6},{"x":1,"y":6},{"x":1,"y":5},{"x":2,"y":5}],"latency":"400","head":{"x":5,"y":6},"shout":"","customizations":{"color":"#118645","head":"replit-mark","tail":"replit-notmark"}}]}`,
		// 	Iterations:      math.MaxInt,
		// 	AcceptableMoves: []string{"left", "down"}, // left seems better but down is possibly correct
		// },
		// {
		// 	Description:     "don't get transfixed by draws",
		// 	InitialBoard:    `{"height":11,"width":11,"food":[{"x":8,"y":0},{"x":5,"y":5},{"x":5,"y":10},{"x":10,"y":4}],"hazards":[],"snakes":[{"id":"6fb7c491-2ad6-4718-83c6-613840dfe0ea","name":"mcts","health":96,"body":[{"x":6,"y":0},{"x":5,"y":0},{"x":4,"y":0},{"x":3,"y":0}],"latency":"902","head":{"x":6,"y":0},"shout":"","customizations":{"color":"#888888","head":"default","tail":"default"}},{"id":"eaf1ae04-8bcb-4e74-8ab7-6328d207f797","name":"soba","health":94,"body":[{"x":4,"y":2},{"x":5,"y":2},{"x":5,"y":1}],"latency":"401","head":{"x":4,"y":2},"shout":"","customizations":{"color":"#118645","head":"replit-mark","tail":"replit-notmark"}}]}`,
		// 	Iterations:      10,
		// 	AcceptableMoves: []string{"up"},
		// },
		{
			Description:  "control isn't right",
			InitialBoard: `{"height":11,"width":11,"food":[{"x":0,"y":2},{"x":0,"y":4},{"x":7,"y":0},{"x":6,"y":10}],"hazards":[],"snakes":[{"id":"8d1de07d-92cf-4ac9-a23e-45aeb8bc14c1","name":"mcts","health":63,"body":[{"x":8,"y":3},{"x":7,"y":3},{"x":7,"y":4},{"x":6,"y":4},{"x":6,"y":3}],"latency":"406","head":{"x":8,"y":3},"shout":"","customizations":{"color":"#888888","head":"default","tail":"default"}},{"id":"a6afe25e-c5fc-450a-b9f1-40f638fe8be0","name":"soba","health":91,"body":[{"x":9,"y":6},{"x":8,"y":6},{"x":7,"y":6},{"x":6,"y":6},{"x":6,"y":7},{"x":6,"y":8}],"latency":"401","head":{"x":9,"y":6},"shout":"","customizations":{"color":"#118645","head":"replit-mark","tail":"replit-notmark"}}]}`,
			Iterations:   math.MaxInt,
		},
		// {
		// 	Description:  "should move towards centre",
		// 	InitialBoard: `{"height":11,"width":11,"food":[{"x":2,"y":0},{"x":9,"y":8},{"x":0,"y":0},{"x":7,"y":10}],"hazards":[],"snakes":[{"id":"15eec745-def3-4e65-8250-bbf9869d304f","name":"mcts","health":90,"body":[{"x":1,"y":5},{"x":1,"y":6},{"x":1,"y":7},{"x":1,"y":8}],"latency":"401","head":{"x":1,"y":5},"shout":"","customizations":{"color":"#888888","head":"default","tail":"default"}},{"id":"58977559-5285-417b-bde9-824d647160d9","name":"soba","health":96,"body":[{"x":2,"y":10},{"x":2,"y":9},{"x":2,"y":8},{"x":3,"y":8},{"x":4,"y":8},{"x":5,"y":8}],"latency":"401","head":{"x":2,"y":10},"shout":"","customizations":{"color":"#118645","head":"replit-mark","tail":"replit-notmark"}}]}`,
		// 	Iterations:   math.MaxInt,
		// },
		// {
		// 	Description:  "don't move into trap",
		// 	InitialBoard: `{"height":11,"width":11,"food":[{"x":10,"y":2},{"x":5,"y":5},{"x":9,"y":1}],"hazards":[],"snakes":[{"id":"38241f5a-33f2-426c-a754-abf0d779521a","name":"mcts","health":86,"body":[{"x":7,"y":1},{"x":6,"y":1},{"x":5,"y":1}],"latency":"401","head":{"x":7,"y":1},"shout":"","customizations":{"color":"#888888","head":"default","tail":"default"}},{"id":"4898c918-def5-48a1-95cd-b38e58b66f2d","name":"soba","health":92,"body":[{"x":4,"y":2},{"x":4,"y":1},{"x":3,"y":1},{"x":2,"y":1}],"latency":"401","head":{"x":4,"y":2},"shout":"","customizations":{"color":"#118645","head":"replit-mark","tail":"replit-notmark"}}]}`,
		// 	Iterations:   math.MaxInt,
		// },
		// {
		// 	Description:  "other snake's tail shouldn't trick us into trap",
		// 	InitialBoard: `{"height":11,"width":11,"food":[{"x":1,"y":8},{"x":0,"y":8},{"x":10,"y":7},{"x":7,"y":10},{"x":8,"y":1}],"hazards":[],"snakes":[{"id":"gs_xY7RRtB98qft6dGwHtRmxtHc","name":"Gregory","health":51,"body":[{"x":2,"y":4},{"x":2,"y":5},{"x":3,"y":5},{"x":4,"y":5},{"x":5,"y":5}],"latency":"459","head":{"x":2,"y":4},"shout":"This is a nice move.","customizations":{"color":"#888888","head":"default","tail":"default"}},{"id":"gs_TMvV6VVr6TcKYDkq4f8PtKH9","name":"snakos","health":81,"body":[{"x":4,"y":2},{"x":4,"y":3},{"x":4,"y":4},{"x":5,"y":4},{"x":6,"y":4},{"x":7,"y":4},{"x":8,"y":4},{"x":9,"y":4},{"x":10,"y":4},{"x":10,"y":3},{"x":10,"y":2},{"x":10,"y":1},{"x":10,"y":0},{"x":9,"y":0},{"x":8,"y":0},{"x":7,"y":0},{"x":6,"y":0},{"x":5,"y":0},{"x":4,"y":0},{"x":3,"y":0},{"x":3,"y":1},{"x":3,"y":2}],"latency":"78","head":{"x":4,"y":2},"shout":"chasing tail","customizations":{"color":"#ff8645","head":"replit-mark","tail":"replit-notmark"}}]}`,
		// 	Iterations:   math.MaxInt,
		// },
		// {
		// 	Description:  "other snake's tail shouldn't trick us into trap",
		// 	InitialBoard: `{"height":11,"width":11,"food":[{"x":1,"y":8},{"x":0,"y":8},{"x":10,"y":7},{"x":7,"y":10},{"x":8,"y":1}],"hazards":[],"snakes":[{"id":"gs_xY7RRtB98qft6dGwHtRmxtHc","name":"Gregory","health":51,"body":[{"x":2,"y":4},{"x":2,"y":5},{"x":3,"y":5},{"x":4,"y":5},{"x":5,"y":5}],"latency":"459","head":{"x":2,"y":4},"shout":"This is a nice move.","customizations":{"color":"#888888","head":"default","tail":"default"}},{"id":"gs_TMvV6VVr6TcKYDkq4f8PtKH9","name":"snakos","health":81,"body":[{"x":4,"y":2},{"x":4,"y":3},{"x":4,"y":4},{"x":5,"y":4},{"x":6,"y":4},{"x":7,"y":4},{"x":8,"y":4},{"x":9,"y":4},{"x":10,"y":4},{"x":10,"y":3},{"x":10,"y":2},{"x":10,"y":1},{"x":10,"y":0},{"x":9,"y":0},{"x":8,"y":0},{"x":7,"y":0},{"x":6,"y":0},{"x":5,"y":0},{"x":4,"y":0},{"x":3,"y":0},{"x":3,"y":1},{"x":3,"y":2}],"latency":"78","head":{"x":4,"y":2},"shout":"chasing tail","customizations":{"color":"#ff8645","head":"replit-mark","tail":"replit-notmark"}}]}`,
		// 	Iterations:   math.MaxInt,
		// },
		// {
		// 	Description:  "can chase tail to freedom",
		// 	InitialBoard: `{"height":11,"width":11,"food":[{"X":10,"Y":9},{"X":9,"Y":10},{"X":0,"Y":0},{"X":10,"Y":4},{"X":0,"Y":10},{"X":0,"Y":5},{"X":0,"Y":7},{"X":1,"Y":0},{"X":6,"Y":6},{"X":3,"Y":3}],"hazards":[],"snakes":[{"id":"gs_vRg7TtfdrGy79wG4GjfwPhx6","name":"Gregory","health":96,"body":[{"X":1,"Y":4},{"X":1,"Y":3},{"X":1,"Y":2},{"X":2,"Y":2},{"X":3,"Y":2},{"X":3,"Y":1},{"X":2,"Y":1},{"X":2,"Y":0},{"X":3,"Y":0},{"X":4,"Y":0},{"X":5,"Y":0},{"X":6,"Y":0},{"X":6,"Y":1},{"X":7,"Y":1},{"X":8,"Y":1},{"X":8,"Y":2},{"X":7,"Y":2},{"X":6,"Y":2},{"X":5,"Y":2},{"X":4,"Y":2},{"X":4,"Y":3},{"X":4,"Y":4},{"X":3,"Y":4}],"latency":"457","head":{"X":1,"Y":4},"shout":"This is a nice move."},{"id":"gs_HRKWrTyp847KtHDVtTDbmy8Q","name":"soba","health":88,"body":[{"X":1,"Y":6},{"X":2,"Y":6},{"X":3,"Y":6},{"X":3,"Y":5},{"X":4,"Y":5},{"X":5,"Y":5},{"X":5,"Y":4},{"X":5,"Y":3},{"X":6,"Y":3},{"X":7,"Y":3},{"X":7,"Y":4},{"X":6,"Y":4},{"X":6,"Y":5},{"X":7,"Y":5},{"X":8,"Y":5},{"X":9,"Y":5},{"X":10,"Y":5},{"X":10,"Y":6},{"X":9,"Y":6},{"X":8,"Y":6},{"X":7,"Y":6},{"X":7,"Y":7},{"X":8,"Y":7},{"X":8,"Y":8},{"X":7,"Y":8},{"X":6,"Y":8}],"latency":"411","head":{"X":1,"Y":6},"shout":"swag"}]}`,
		// 	Iterations:   math.MaxInt,
		// },
		// {
		// 	Description:  "up leads to kill from opponent - was due to food not being cached (need a test for this eventually)",
		// 	InitialBoard: `{"height":11,"width":11,"food":[{"X":4,"Y":10},{"X":0,"Y":1},{"X":9,"Y":2},{"X":0,"Y":0},{"X":0,"Y":3},{"X":9,"Y":10},{"X":9,"Y":5},{"X":6,"Y":0},{"X":2,"Y":0},{"X":1,"Y":10},{"X":9,"Y":4},{"X":7,"Y":10},{"X":8,"Y":1},{"X":6,"Y":6}],"hazards":[],"snakes":[{"id":"gs_P6tqpPjgJRCxPQm8yKTkd43S","name":"Gregory","health":21,"body":[{"X":6,"Y":5},{"X":5,"Y":5},{"X":5,"Y":6},{"X":4,"Y":6}],"latency":"458","head":{"X":6,"Y":5},"shout":"This is a nice move."},{"id":"gs_crwYTW6B7RkCh7YvQDmRJqhS","name":"soba","health":88,"body":[{"X":8,"Y":7},{"X":7,"Y":7},{"X":6,"Y":7},{"X":6,"Y":8},{"X":6,"Y":9},{"X":5,"Y":9},{"X":4,"Y":9},{"X":3,"Y":9}],"latency":"409","head":{"X":8,"Y":7},"shout":"swag"}]}`,
		// 	Iterations:   math.MaxInt,
		// },
		// {
		// 	Description:     "goes towards longer snake to its death. should escape left. caused by incorrectly judging winning position",
		// 	InitialBoard:    `{"height":11,"width":11,"food":[{"x":10,"y":0},{"x":10,"y":3},{"x":8,"y":1},{"x":9,"y":0},{"x":3,"y":1},{"x":4,"y":2},{"x":8,"y":4},{"x":3,"y":0},{"x":9,"y":5},{"x":3,"y":8}],"hazards":[],"snakes":[{"id":"bbc27600-9763-4cce-954a-b3d6fa0d58de","name":"mcts","health":72,"body":[{"x":2,"y":8},{"x":2,"y":7},{"x":2,"y":6},{"x":2,"y":5},{"x":1,"y":5},{"x":1,"y":4},{"x":2,"y":4},{"x":2,"y":3},{"x":3,"y":3},{"x":4,"y":3},{"x":5,"y":3},{"x":6,"y":3},{"x":6,"y":4},{"x":6,"y":5},{"x":5,"y":5},{"x":4,"y":5},{"x":4,"y":4},{"x":3,"y":4},{"x":3,"y":5}],"latency":"451","head":{"x":2,"y":8},"shout":"","customizations":{"color":"#888888","head":"default","tail":"default"}},{"id":"a34717ee-ee2f-472e-ba78-a99e446a310a","name":"soba","health":92,"body":[{"x":5,"y":7},{"x":6,"y":7},{"x":7,"y":7},{"x":8,"y":7},{"x":9,"y":7},{"x":10,"y":7},{"x":10,"y":8},{"x":10,"y":9},{"x":10,"y":10},{"x":9,"y":10},{"x":9,"y":9},{"x":9,"y":8},{"x":8,"y":8},{"x":7,"y":8},{"x":6,"y":8},{"x":6,"y":9},{"x":5,"y":9},{"x":5,"y":8},{"x":4,"y":8},{"x":4,"y":9},{"x":3,"y":9},{"x":2,"y":9}],"latency":"401","head":{"x":5,"y":7},"shout":"","customizations":{"color":"#118645","head":"replit-mark","tail":"replit-notmark"}}]}`,
		// 	Iterations:      math.MaxInt,
		// 	AcceptableMoves: []string{"left"},
		// },
	}

	for _, tc := range testCases {
		t.Run(tc.Description, func(t *testing.T) {

			numCPUs := runtime.NumCPU()
			_ = numCPUs
			var board Board
			assert.NoError(t, json.Unmarshal([]byte(tc.InitialBoard), &board))
			rootBoard := copyBoard(board)
			ctx, _ := context.WithTimeout(context.Background(), 500*time.Millisecond)

			workers := runtime.NumCPU()
			t.Log("using workers", workers)
			node := MCTS(ctx, "testid", rootBoard, tc.Iterations, 1, make(map[string]*Node))
			t.Log("made moves", node.Visits)
			bestMove := determineBestMove(node)

			assert.Contains(t, tc.AcceptableMoves, bestMove, "snake made move it shouldn't have, moved %s", bestMove)

			require.NotNil(t, node, "node is nil")

			// assert.NoError(t, GenerateMostVisitedPathWithAlternativesHtmlTree(node))

		})
	}
}
