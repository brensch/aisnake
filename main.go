package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"time"
)

func main() {
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/start", handleStart)
	http.HandleFunc("/move", handleMove)
	http.HandleFunc("/end", handleEnd)

	port := "8080"
	fmt.Printf("Starting BattleSnake on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// handleIndex handles the root endpoint and provides basic info about the snake.
func handleIndex(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"apiversion": "1",
		"author":     "brensch",
		"color":      "#888888", // Customize your snake's color
		"head":       "default", // Customize your snake's head
		"tail":       "default", // Customize your snake's tail
		"version":    "0.1.0",
	}
	writeJSON(w, response)
}

// handleStart is called when the game starts.
func handleStart(w http.ResponseWriter, r *http.Request) {
	var game BattleSnakeGame
	if err := json.NewDecoder(r.Body).Decode(&game); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Log the start of the game
	fmt.Printf("Game %s started\n", game.Game.ID)

	writeJSON(w, map[string]string{})
}

// handleMove is called on every turn to determine the snake's next move.
func handleMove(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var game BattleSnakeGame
	if err := json.NewDecoder(r.Body).Decode(&game); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create a context with a timeout to ensure MCTS doesn't run indefinitely
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(game.Game.Timeout-50)*time.Millisecond)
	defer cancel()

	// Run MCTS to determine the best move
	mctsResult := MCTS(ctx, game.Board, math.MaxInt, 4)

	// Determine the move based on the best child node returned by MCTS
	bestMove := determineBestMove(game, mctsResult)

	// Respond with the move
	response := map[string]string{
		"move":  bestMove,
		"shout": "This is a nice move.",
	}
	writeJSON(w, response)
	log.Println("made move", bestMove, time.Since(start).Milliseconds())
}

// determineBestMove finds the best move from the MCTS result
func determineBestMove(game BattleSnakeGame, node *Node) string {
	var bestChild *Node
	maxVisits := -1

	// Iterate through the children to find the one with the highest visit count
	for _, child := range node.Children {
		if child.Visits > maxVisits {
			bestChild = child
			maxVisits = child.Visits
		}
	}

	// If we found a best child, determine the direction to move
	if bestChild != nil {
		bestMove := determineMoveDirection(game.You.Head, bestChild.Board.Snakes[0].Head)
		return bestMove
	}

	// Fallback to a random move if no children are found (shouldn't happen if MCTS is working correctly)
	moves := []string{"up", "down", "left", "right"}
	return moves[rand.Intn(len(moves))]
}

// determineMoveDirection determines the direction to move based on the change in position
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

// handleEnd is called when the game ends.
func handleEnd(w http.ResponseWriter, r *http.Request) {
	var game BattleSnakeGame
	if err := json.NewDecoder(r.Body).Decode(&game); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Log the end of the game
	fmt.Printf("Game %s ended after %d turns\n", game.Game.ID, game.Turn)

	writeJSON(w, map[string]string{})
}

// writeJSON is a helper function to write a JSON response.
func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
