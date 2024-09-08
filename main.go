package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(game.Game.Timeout-100)*time.Millisecond)
	defer cancel()

	// Perform MCTS to find the best move
	mctsResult := MCTS(ctx, reorderedBoard, math.MaxInt)
	bestMove := determineBestMove(game, mctsResult)

	// Prepare and send the response immediately
	response := map[string]string{
		"move":  bestMove,
		"shout": "This is a nice move.",
	}
	writeJSON(w, response)

	// Perform non-essential operations after sending the response
	defer func() {
		// Ensure the movetrees directory exists
		if err := os.MkdirAll("movetrees", os.ModePerm); err != nil {
			log.Println("Error creating movetrees directory:", err)
			return
		}

		fmt.Println("--------------------------")
		// Logging additional information
		fmt.Println(visualizeBoard(game.Board))
		fmt.Println("Received move request for snake", game.You.ID)
		log.Println("Made move:", bestMove, "in", time.Since(start).Milliseconds(), "ms with", mctsResult.Visits, "visits")
		// Generate and log the tree diagram
		if err := GenerateMostVisitedPathWithAlternativesHtmlTree(mctsResult); err != nil {
			log.Println("Error saving mermaid tree:", err)
			return
		}
	}()
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
