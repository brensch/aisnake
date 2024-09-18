package main

// import (
// 	"context"
// 	"log/slog"
// 	"math"
// )

// type Node struct {
// 	Board      Board
// 	SnakeIndex int
// 	Parent     *Node
// 	Children   []*Node
// 	Visits     int
// 	Score      float64
// 	MyScore    float64
// }

// func NewNode(board Board, snakeIndex int) *Node {
// 	return &Node{
// 		Board:      board,
// 		SnakeIndex: snakeIndex,
// 		Children:   make([]*Node, 0), // Ensuring Children slice is initialized
// 		Visits:     0,
// 		Score:      0,
// 	}
// }

// // Expand the node to add children based on the board's state
// func expand(node *Node) {
// 	// If already expanded, return immediately
// 	if len(node.Children) > 0 {
// 		return
// 	}

// 	nextSnakeIndex := (node.SnakeIndex + 1) % len(node.Board.Snakes)

// 	moves := generateSafeMoves(node.Board, nextSnakeIndex)
// 	if len(moves) == 0 {
// 		moves = append(moves, Up)
// 	}

// 	for _, move := range moves {
// 		newBoard := copyBoard(node.Board)
// 		applyMove(&newBoard, nextSnakeIndex, move)

// 		child := NewNode(newBoard, nextSnakeIndex)
// 		child.Parent = node
// 		node.Children = append(node.Children, child)
// 	}
// }

// func isTerminal(board Board) bool {
// 	aliveSnakesCount := 0
// 	for _, snake := range board.Snakes {
// 		if !isSnakeDead(snake) {
// 			aliveSnakesCount++
// 		}
// 	}
// 	return aliveSnakesCount <= 1
// }

// // isSnakeDead checks if a snake is dead by looking at its health and body length.
// func isSnakeDead(snake Snake) bool {
// 	return len(snake.Body) == 0 || snake.Health <= 0
// }

// func (n *Node) UCT(parent *Node, explorationParam float64) float64 {
// 	if n.Visits == 0 {
// 		return math.MaxFloat64
// 	}

// 	exploitation := n.Score / float64(n.Visits)
// 	exploration := explorationParam * math.Sqrt(math.Log(float64(parent.Visits))/float64(n.Visits))

// 	return exploitation + exploration
// }

// // Select the best child node based on the UCT value
// func bestChild(node *Node, explorationParam float64) *Node {
// 	if len(node.Children) == 0 {
// 		return nil // No children available
// 	}

// 	bestValue := -math.MaxFloat64
// 	var bestNode *Node

// 	for _, child := range node.Children {
// 		if child == nil {
// 			continue // Skip nil children
// 		}

// 		value := child.UCT(node, explorationParam)

// 		if value > bestValue {
// 			bestValue = value
// 			bestNode = child
// 		}
// 	}

// 	return bestNode
// }

// func MCTS(ctx context.Context, gameID string, rootBoard Board, iterations int, numWorkers int, gameStates map[string]*Node) *Node {
// 	// Generate the hash for the current board state
// 	boardKey := boardHash(rootBoard)
// 	var rootNode *Node
// 	// If the board state is already known, use the existing node
// 	if existingNode, ok := gameStates[boardKey]; ok {
// 		slog.Info("board cache lookup", "hit", true, "cache_size", len(gameStates), "visits", existingNode.Visits)
// 		rootNode = existingNode
// 	} else {
// 		// Otherwise, create a new node and add it to the game state map
// 		slog.Info("board cache lookup", "hit", false, "cache_size", len(gameStates))
// 		rootNode = NewNode(rootBoard, -1)
// 		expand(rootNode)
// 	}

// 	for i := 0; i < iterations; i++ {
// 		// Check for cancellation before each iteration
// 		select {
// 		case <-ctx.Done():
// 			return rootNode
// 		default:
// 			// Continue execution
// 		}

// 		node := rootNode
// 		// Selection
// 		for {
// 			// Check for cancellation during selection
// 			select {
// 			case <-ctx.Done():
// 				return rootNode
// 			default:
// 				// Continue execution
// 			}

// 			if len(node.Children) == 0 {
// 				break
// 			}
// 			nextNode := bestChild(node, 1.41)
// 			if nextNode == nil {
// 				break
// 			}
// 			node = nextNode
// 		}

// 		// Expansion
// 		if !isTerminal(node.Board) && node.Visits > 0 {
// 			expand(node)
// 			if len(node.Children) > 0 {
// 				// Optionally, select a child to proceed (e.g., the first one)
// 				node = node.Children[0]
// 			}
// 		}

// 		// Simulation
// 		var score float64
// 		if node.Visits == 0 {
// 			score = evaluateBoard(node.Board, node.SnakeIndex)
// 			node.MyScore = score
// 		} else {
// 			score = node.MyScore
// 		}

// 		// Backpropagation
// 		for n := node; n != nil; n = n.Parent {
// 			n.Visits++
// 			n.Score += score
// 			score = -score
// 		}
// 	}

// 	return rootNode
// }

// func evaluateBoard(board Board, snakeIndex int) float64 {
// 	if isSnakeDead(board.Snakes[snakeIndex]) {
// 		return -2
// 	}

// 	// TODO: Adjust for multiplayer scenarios
// 	if isSnakeDead(board.Snakes[(snakeIndex+1)%len(board.Snakes)]) {
// 		return 2
// 	}

// 	// Voronoi evaluation: Calculate the area controlled by each snake
// 	// Note: This is a simplified version
// 	voronoi := GenerateVoronoi(board)
// 	totalCells := float64(board.Width * board.Height)
// 	controlledCells := 0.0
// 	opponentsCells := 0.0
// 	lengthBonus := 0.0

// 	// Count the number of cells each snake controls in the Voronoi diagram
// 	for y := 0; y < board.Height; y++ {
// 		for x := 0; x < board.Width; x++ {
// 			if voronoi[y][x] == snakeIndex {
// 				controlledCells++
// 			} else if voronoi[y][x] != -1 {
// 				opponentsCells++
// 			}
// 		}
// 	}

// 	// Calculate length bonus/penalty
// 	mySnake := board.Snakes[snakeIndex]
// 	for i, opponent := range board.Snakes {
// 		if i != snakeIndex && !isSnakeDead(opponent) {
// 			lengthDifference := len(mySnake.Body) - len(opponent.Body)

// 			if lengthDifference >= 2 {
// 				// Bonus for being longer
// 				lengthBonus += 0.3 * float64(lengthDifference)
// 			} else {
// 				// Penalty for being shorter
// 				lengthBonus += 0.1 * float64(lengthDifference)
// 			}
// 		}
// 	}

// 	// Return a score based on the difference in controlled areas and length bonus
// 	return ((controlledCells - opponentsCells) / totalCells) + lengthBonus
// }
