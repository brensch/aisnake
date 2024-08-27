package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAllMoves(t *testing.T) {
	testCases := []struct {
		Description   string
		Board         Board
		ExpectedMoves []Move
	}{
		{
			Description: "No snakes on the board",
			Board: Board{
				Height: 5,
				Width:  5,
				Snakes: []Snake{},
			},
			ExpectedMoves: []Move{},
		},
		{
			Description: "One snake in the middle of the board",
			Board: Board{
				Height: 5,
				Width:  5,
				Snakes: []Snake{
					{ID: "snake1", Head: Point{X: 2, Y: 2}},
				},
			},
			ExpectedMoves: []Move{
				{Up},
				{Down},
				{Left},
				{Right},
			},
		},
		{
			Description: "One snake at the bottom-left corner of the board",
			Board: Board{
				Height: 5,
				Width:  5,
				Snakes: []Snake{
					{ID: "snake1", Head: Point{X: 0, Y: 0}}, // Bottom-left corner
				},
			},
			ExpectedMoves: []Move{
				{Up},    // Can move up
				{Right}, // Can move right
			},
		},
		{
			Description: "Two snakes on the board with valid moves only",
			Board: Board{
				Height: 5,
				Width:  5,
				Snakes: []Snake{
					{ID: "snake1", Head: Point{X: 1, Y: 1}}, // Near bottom-left corner
					{ID: "snake2", Head: Point{X: 3, Y: 3}}, // Near center
				},
			},
			ExpectedMoves: []Move{
				{Up, Up}, {Up, Down}, {Up, Left}, {Up, Right},
				{Right, Up}, {Right, Down}, {Right, Left}, {Right, Right},
				{Left, Up}, {Left, Down}, {Left, Left}, {Left, Right},
				{Down, Up}, {Down, Down}, {Down, Left}, {Down, Right},
			},
		},
		{
			Description: "Snake on the board edge with other snakes",
			Board: Board{
				Height: 5,
				Width:  5,
				Snakes: []Snake{
					{ID: "snake1", Head: Point{X: 4, Y: 4}}, // Top-right corner
					{ID: "snake2", Head: Point{X: 2, Y: 2}}, // Center
				},
			},
			ExpectedMoves: []Move{
				{Down, Up}, {Down, Down}, {Down, Left}, {Down, Right},
				{Left, Up}, {Left, Down}, {Left, Left}, {Left, Right},
			},
		},
		{
			Description: "Two snakes on the board with corner positions",
			Board: Board{
				Height: 5,
				Width:  5,
				Snakes: []Snake{
					{ID: "snake1", Head: Point{X: 0, Y: 0}}, // Bottom-left corner
					{ID: "snake2", Head: Point{X: 4, Y: 4}}, // Top-right corner
				},
			},
			ExpectedMoves: []Move{
				{Up, Down}, {Up, Left},
				{Right, Down}, {Right, Left},
			},
		},
		{
			Description: "One multi-length snake in the middle of the board",
			Board: Board{
				Height: 5,
				Width:  5,
				Snakes: []Snake{
					{
						ID:   "snake1",
						Head: Point{X: 2, Y: 2},
						Body: []Point{
							{X: 2, Y: 2},
							{X: 2, Y: 1}, // Neck position, right behind the head
							{X: 2, Y: 0},
						},
					},
				},
			},
			ExpectedMoves: []Move{
				{Up},    // Can move up
				{Left},  // Can move left
				{Right}, // Can move right
				// {Down} is invalid as it would cause the snake to move back on itself
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Description, func(t *testing.T) {
			moves := generateAllMoves(tc.Board)

			if tc.ExpectedMoves != nil {
				assert.ElementsMatch(t, tc.ExpectedMoves, moves, "Moves do not match expected values")
			}
		})
	}
}

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
			selectedChild := tc.Parent.SelectChild()

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
	ExpectedMoves    []Move
}

func TestExpand(t *testing.T) {
	testCases := []ExpandTestCase{
		{
			Description: "Expand with one untried move",
			InitialNode: func() *Node {
				board := Board{
					Height: 5, Width: 5,
					Snakes: []Snake{
						{ID: "snake1", Head: Point{X: 2, Y: 2}},
					},
				}
				node := NewNode(board, nil)
				node.UntriedMoves = []Move{
					{Up},
				}
				return node
			}(),
			ExpectedChildren: 1,
			ExpectedMoves: []Move{
				{Up},
			},
		},
		{
			Description: "Expand with multiple untried moves for one snake",
			InitialNode: func() *Node {
				board := Board{
					Height: 5, Width: 5,
					Snakes: []Snake{
						{ID: "snake1", Head: Point{X: 2, Y: 2}},
					},
				}
				node := NewNode(board, nil)
				node.UntriedMoves = []Move{
					{Up}, {Down}, {Left}, {Right},
				}
				return node
			}(),
			ExpectedChildren: 4,
			ExpectedMoves: []Move{
				{Up}, {Down}, {Left}, {Right},
			},
		},
		{
			Description: "Expand with two snakes and multiple untried moves",
			InitialNode: func() *Node {
				board := Board{
					Height: 5, Width: 5,
					Snakes: []Snake{
						{ID: "snake1", Head: Point{X: 2, Y: 2}},
						{ID: "snake2", Head: Point{X: 3, Y: 3}},
					},
				}
				node := NewNode(board, nil)
				node.UntriedMoves = []Move{
					{Up, Down}, {Left, Right},
				}
				return node
			}(),
			ExpectedChildren: 2,
			ExpectedMoves: []Move{
				{Up, Down}, {Left, Right},
			},
		},
		{
			Description: "Expand when no untried moves remain",
			InitialNode: func() *Node {
				board := Board{
					Height: 5, Width: 5,
					Snakes: []Snake{
						{ID: "snake1", Head: Point{X: 2, Y: 2}},
					},
				}
				node := NewNode(board, nil)
				node.UntriedMoves = []Move{}
				return node
			}(),
			ExpectedChildren: 0,
			ExpectedMoves:    []Move{},
		},
		{
			Description: "Expand with multiple snakes and all possible moves",
			InitialNode: func() *Node {
				board := Board{
					Height: 5, Width: 5,
					Snakes: []Snake{
						{ID: "snake1", Head: Point{X: 2, Y: 2}},
						{ID: "snake2", Head: Point{X: 3, Y: 3}},
					},
				}
				node := NewNode(board, nil)
				node.UntriedMoves = generateAllMoves(board)
				return node
			}(),
			ExpectedChildren: 16, // 4 directions per snake, so 4 * 4 = 16 combinations
			ExpectedMoves: generateAllMoves(Board{
				Height: 5, Width: 5,
				Snakes: []Snake{
					{ID: "snake1", Head: Point{X: 2, Y: 2}},
					{ID: "snake2", Head: Point{X: 3, Y: 3}},
				},
			}),
		},
		{
			Description: "Expand with a snake at the board edge",
			InitialNode: func() *Node {
				board := Board{
					Height: 5, Width: 5,
					Snakes: []Snake{
						{ID: "snake1", Head: Point{X: 4, Y: 4}}, // At the bottom-right corner
					},
				}
				node := NewNode(board, nil)
				node.UntriedMoves = []Move{
					{Up}, {Left},
				}
				return node
			}(),
			ExpectedChildren: 2,
			ExpectedMoves: []Move{
				{Up}, {Left},
			},
		},
		{
			Description: "Expand with multiple snakes and hazard on the board",
			InitialNode: func() *Node {
				board := Board{
					Height: 5, Width: 5,
					Hazards: []Point{
						{X: 2, Y: 2},
					},
					Snakes: []Snake{
						{ID: "snake1", Head: Point{X: 1, Y: 1}},
						{ID: "snake2", Head: Point{X: 3, Y: 3}},
					},
				}
				node := NewNode(board, nil)
				node.UntriedMoves = []Move{
					{Right, Left}, {Down, Up},
				}
				return node
			}(),
			ExpectedChildren: 2,
			ExpectedMoves: []Move{
				{Right, Left}, {Down, Up},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Description, func(t *testing.T) {
			children := tc.InitialNode.Expand()

			// Verify the number of children
			require.Equal(t, tc.ExpectedChildren, len(children), "Number of children nodes does not match")

			// Ensure UntriedMoves is cleared after expansion
			require.Equal(t, 0, len(tc.InitialNode.UntriedMoves), "Untried moves should be cleared after expansion")

			for i, child := range children {
				require.NotNil(t, child, "Child node should not be nil")

				// Generate the expected board state
				expectedBoard := copyBoard(tc.InitialNode.Board)
				applyMoves(&expectedBoard, tc.ExpectedMoves[i])

				// Compare the actual child board with the expected board state
				if !assert.Equal(t, expectedBoard, child.Board, "The child's board state does not match the expected state after applying the move") {
					t.Logf("Test failed on case: %s", tc.Description)
					t.Logf("Expected Move: %+v", tc.ExpectedMoves[i])
					t.Logf("Expected Board: %+v", expectedBoard)
					t.Logf("Actual Board: %+v", child.Board)
				}

				// Verify that the parent of the child node is the initial node
				assert.Equal(t, tc.InitialNode, child.Parent, "Child node's parent should be the initial node")
			}
		})
	}
}

type ApplyMovesTestCase struct {
	Description   string
	InitialBoard  Board
	Moves         Move
	ExpectedBoard Board
}

func TestApplyMoves(t *testing.T) {
	testCases := []ApplyMovesTestCase{
		{
			Description: "Single snake moves up and loses health",
			InitialBoard: Board{
				Height: 5, Width: 5,
				Snakes: []Snake{
					{ID: "snake1", Health: 100, Head: Point{X: 2, Y: 2}, Body: []Point{{X: 2, Y: 2}, {X: 2, Y: 1}}},
				},
			},
			Moves: Move{Up},
			ExpectedBoard: Board{
				Height: 5, Width: 5,
				Snakes: []Snake{
					{ID: "snake1", Health: 99, Head: Point{X: 2, Y: 3}, Body: []Point{{X: 2, Y: 3}, {X: 2, Y: 2}}},
				},
			},
		},
		{
			Description: "Single snake eats food, grows, and restores health",
			InitialBoard: Board{
				Height: 5, Width: 5,
				Food: []Point{{X: 2, Y: 3}},
				Snakes: []Snake{
					{ID: "snake1", Health: 98, Head: Point{X: 2, Y: 2}, Body: []Point{{X: 2, Y: 2}, {X: 2, Y: 1}}},
				},
			},
			Moves: Move{Up},
			ExpectedBoard: Board{
				Height: 5, Width: 5,
				Food: []Point{}, // Food is consumed
				Snakes: []Snake{
					{ID: "snake1", Health: 100, Head: Point{X: 2, Y: 3}, Body: []Point{{X: 2, Y: 3}, {X: 2, Y: 2}, {X: 2, Y: 1}}},
				},
			},
		},
		{
			Description: "Snake runs into wall and dies",
			InitialBoard: Board{
				Height: 5, Width: 5,
				Snakes: []Snake{
					{ID: "snake1", Health: 100, Head: Point{X: 4, Y: 4}, Body: []Point{{X: 4, Y: 4}, {X: 3, Y: 4}}},
				},
			},
			Moves: Move{Right},
			ExpectedBoard: Board{
				Height: 5, Width: 5,
				Snakes: []Snake{}, // Snake dies
			},
		},
		{
			Description: "Two snakes collide head-to-head, longer one survives",
			InitialBoard: Board{
				Height: 5, Width: 5,
				Snakes: []Snake{
					{ID: "snake1", Health: 100, Head: Point{X: 2, Y: 2}, Body: []Point{{X: 2, Y: 2}, {X: 1, Y: 2}, {X: 0, Y: 2}}},
					{ID: "snake2", Health: 100, Head: Point{X: 3, Y: 2}, Body: []Point{{X: 3, Y: 2}, {X: 4, Y: 2}}},
				},
			},
			Moves: Move{Right, Left},
			ExpectedBoard: Board{
				Height: 5, Width: 5,
				Snakes: []Snake{
					{ID: "snake1", Health: 99, Head: Point{X: 3, Y: 2}, Body: []Point{{X: 3, Y: 2}, {X: 2, Y: 2}, {X: 1, Y: 2}}},
					// snake2 should die
				},
			},
		},
		{
			Description: "Two snakes collide heads at 90 degrees",
			InitialBoard: Board{
				Height: 5, Width: 5,
				Snakes: []Snake{
					{ID: "snake1", Health: 100, Head: Point{X: 2, Y: 2}, Body: []Point{{X: 2, Y: 2}, {X: 2, Y: 1}, {X: 2, Y: 0}}},
					{ID: "snake2", Health: 100, Head: Point{X: 3, Y: 3}, Body: []Point{{X: 3, Y: 3}, {X: 3, Y: 4}}},
				},
			},
			Moves: Move{Right, Down},
			ExpectedBoard: Board{
				Height: 5, Width: 5,
				Snakes: []Snake{
					// snake1 should survive because it's longer
					{ID: "snake1", Health: 99, Head: Point{X: 3, Y: 2}, Body: []Point{{X: 3, Y: 2}, {X: 2, Y: 2}, {X: 2, Y: 1}}},
					// snake2 should be removed
				},
			},
		},
		{
			Description: "both die on even collision",
			InitialBoard: Board{
				Height: 5, Width: 5,
				Snakes: []Snake{
					{ID: "snake1", Health: 100, Head: Point{X: 2, Y: 2}, Body: []Point{{X: 2, Y: 2}, {X: 2, Y: 1}}},
					{ID: "snake2", Health: 100, Head: Point{X: 3, Y: 3}, Body: []Point{{X: 3, Y: 3}, {X: 3, Y: 4}}},
				},
			},
			Moves: Move{Right, Down},
			ExpectedBoard: Board{
				Height: 5, Width: 5,
				Snakes: []Snake{
					// both snakes should die because they're even
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Description, func(t *testing.T) {
			applyMoves(&tc.InitialBoard, tc.Moves)
			assert.Equal(t, tc.ExpectedBoard, tc.InitialBoard, "The resulting board state does not match the expected board state")
		})
	}
}

func TestMCTSVisualization(t *testing.T) {
	testCases := []struct {
		Description  string
		InitialBoard Board
		Iterations   int
	}{
		{
			Description: "MCTS with small board and few iterations",
			InitialBoard: Board{
				Height: 5,
				Width:  5,
				Snakes: []Snake{
					{ID: "snake1", Head: Point{X: 2, Y: 2}, Health: 100, Body: []Point{{X: 2, Y: 2}}},
				},
				Food: []Point{{X: 4, Y: 4}},
			},
			Iterations: 10,
		},
		{
			Description: "MCTS with multiple snakes and more iterations",
			InitialBoard: Board{
				Height: 7,
				Width:  7,
				Snakes: []Snake{
					{ID: "snake1", Head: Point{X: 2, Y: 2}, Health: 100, Body: []Point{{X: 2, Y: 2}}},
					{ID: "snake2", Head: Point{X: 4, Y: 4}, Health: 100, Body: []Point{{X: 4, Y: 4}}},
				},
				Food: []Point{{X: 5, Y: 5}},
			},
			Iterations: 50,
		},
		// {
		// 	Description: "MCTS with snake near the edge of the board",
		// 	InitialBoard: Board{
		// 		Height: 5,
		// 		Width:  5,
		// 		Snakes: []Snake{
		// 			{ID: "snake1", Head: Point{X: 4, Y: 4}, Health: 100, Body: []Point{{X: 4, Y: 4}}},
		// 		},
		// 		Food: []Point{{X: 2, Y: 2}},
		// 	},
		// 	Iterations: 20,
		// },
	}

	for _, tc := range testCases {
		t.Run(tc.Description, func(t *testing.T) {
			rootBoard := copyBoard(tc.InitialBoard)
			node := MCTS(rootBoard, tc.Iterations)

			// Print out the final state of the best node's board
			fmt.Printf("Test Case: %s\n", tc.Description)
			fmt.Println("Final Board State:")
			fmt.Println(visualizeBoard(node.Board))

			// Print out the final visits and scores
			fmt.Println("Children Nodes' Visits and Scores:")
			for _, child := range node.Parent.Children {
				fmt.Printf("Visits: %d, Score: %.2f\n", child.Visits, evaluateBoard(child.Board))
				fmt.Println(visualizeBoard(child.Board))
				fmt.Println()
			}
		})
	}
}
