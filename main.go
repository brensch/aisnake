package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"runtime"
	"sync"
	"time"
)

// TODO: get rid of globals
// write all of this well also.
var (
	gameStates   = make(map[string]*Node) // Global map to store known game states
	lastMoveTime time.Time                // Tracks the last time a handleMove request was made
	flushTime    = 5 * time.Second
	mu           sync.Mutex // Mutex to protect access to gameStates and lastMoveTime
)

func main() {

	go clearOldGameStates()

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/start", handleStart)
	http.HandleFunc("/move", handleMove)
	http.HandleFunc("/end", handleEnd)

	port := "8080"
	fmt.Printf("Starting BattleSnake on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// Background function to clear the gameStates map if no move request has come in for 30 seconds
func clearOldGameStates() {
	for {
		time.Sleep(3 * time.Second)
		mu.Lock()
		if !lastMoveTime.IsZero() && time.Since(lastMoveTime) > flushTime {
			fmt.Printf("Flushed %d gamestates\n", len(gameStates))
			gameStates = make(map[string]*Node) // Clear the gameStates map
			lastMoveTime = time.Time{}
		}
		mu.Unlock()
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"apiversion": "1",
		"author":     "brensch",
		"color":      "#888888",
		"head":       "default",
		"tail":       "default",
		"version":    "0.1.0",
	}
	writeJSON(w, response)
}

func handleStart(w http.ResponseWriter, r *http.Request) {
	var game BattleSnakeGame
	if err := json.NewDecoder(r.Body).Decode(&game); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Printf("Game %s started\n", game.Game.ID)
	fmt.Println(game.You)

	writeJSON(w, map[string]string{})
}

func handleMove(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Decode the incoming JSON to the game structure
	var game BattleSnakeGame
	if err := json.NewDecoder(r.Body).Decode(&game); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Pre-process the game board and set up the context for MCTS
	reorderedBoard := reorderSnakes(game.Board, game.You.ID)
	// 50ms is a strong yolo
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(game.Game.Timeout-50)*time.Millisecond)
	defer cancel()

	// Update the lastMoveTime to track the current request
	mu.Lock()
	lastMoveTime = time.Now()
	mu.Unlock()

	// Perform MCTS to find the best move, reuse known game states if possible
	workers := runtime.NumCPU()
	mctsResult := MCTS(ctx, reorderedBoard, math.MaxInt, workers, gameStates)
	bestMove := determineBestMove(game, mctsResult)
	saveNodesAtDepth2(mctsResult, gameStates)

	// Prepare and send the response immediately
	response := map[string]string{
		"move":  bestMove,
		"shout": "This is a nice move.",
	}
	writeJSON(w, response)

	// Non-essential logging
	go func() {
		fmt.Println("--------------------------")
		// Logging additional information
		fmt.Println(visualizeBoard(game.Board))
		yo, _ := json.Marshal(game.Board)
		fmt.Println(string(yo))
		fmt.Println("Received move request for snake", game.You.ID)
		log.Println("Made move:", bestMove, "in", time.Since(start).Milliseconds(), "ms with depth", mctsResult.Visits, "visits")
	}()
}

// Save all nodes at depth 2 to the map, these will be the potential returned nodes
func saveNodesAtDepth2(rootNode *Node, gameStates map[string]*Node) {
	// First level: rootNode's children
	for _, child := range rootNode.Children {
		// Second level: child nodes' children (depth 2)
		for _, grandchild := range child.Children {
			// Generate a hash of the board state (ignoring food, etc.)
			boardKey := boardHash(grandchild.Board)

			// Save the grandchild node (at depth 2) to the map
			gameStates[boardKey] = grandchild
		}
	}
}

func reorderSnakes(board Board, youID string) Board {
	var youIndex int
	for index, snake := range board.Snakes {
		if snake.ID == youID {
			youIndex = index
			break
		}
	}
	board.Snakes[0], board.Snakes[youIndex] = board.Snakes[youIndex], board.Snakes[0]
	return board
}

func determineBestMove(game BattleSnakeGame, node *Node) string {
	var bestChild *Node
	maxVisits := -1

	for _, child := range node.Children {
		if child.Visits > maxVisits {
			bestChild = child
			maxVisits = child.Visits
		}
	}

	if bestChild != nil {
		bestMove := determineMoveDirection(game.You.Head, bestChild.Board.Snakes[0].Head)
		return bestMove
	}

	moves := []string{"up", "down", "left", "right"}
	return moves[rand.Intn(len(moves))]
}

func determineMoveDirection(head, nextHead Point) string {
	if nextHead.X < head.X {
		return "left"
	}
	if nextHead.X > head.X {
		return "right"
	}
	if nextHead.Y < head.Y {
		return "down"
	}
	return "up"
}

func handleEnd(w http.ResponseWriter, r *http.Request) {
	var game BattleSnakeGame
	if err := json.NewDecoder(r.Body).Decode(&game); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Printf("Game %s ended after %d turns\n", game.Game.ID, game.Turn)

	writeJSON(w, map[string]string{})
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func boardHash(board Board) string {
	// Create a string that represents the board's state without considering food
	hash := ""
	for _, snake := range board.Snakes {
		for _, part := range snake.Body {
			hash += fmt.Sprintf("S%v%v", part.X, part.Y) // Snake position
		}
	}
	return hash
}
