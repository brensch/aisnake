package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"math"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"
)

var (
	gameStates = make(map[string]map[string]*Node) // Global map to store known game states
	mu         sync.Mutex                          // Mutex to protect access to gameStates and lastMoveTime
)

func main() {
	// Set up the custom handler for Google Cloud
	handler := NewGoogleCloudHandler(os.Stdout, slog.LevelInfo)

	// Create a new logger using the custom handler
	logger := slog.New(handler)

	// Set the logger as default
	slog.SetDefault(logger)

	slog.Info("Starting BattleSnake service", "port", 8080)

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/start", handleStart)
	http.HandleFunc("/move", handleMove)
	http.HandleFunc("/end", handleEnd)

	port := "8080"
	slog.Info("Starting BattleSnake on port", "port", port)
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

	// add a map for this game
	gameStates[game.Game.ID] = make(map[string]*Node)

	slog.Info("Game started", "game_id", game.Game.ID, "you", game.You)

	writeJSON(w, map[string]string{})
}

func handleMove(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var game BattleSnakeGame
	if err := json.NewDecoder(r.Body).Decode(&game); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// get the nodemap for this game
	gameState, ok := gameStates[game.Game.ID]
	if !ok {
		slog.Error("failed to find gamestate. seems like bug.")
		gameState = make(map[string]*Node)
	}

	reorderedBoard := reorderSnakes(game.Board, game.You.ID)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(game.Game.Timeout-50)*time.Millisecond)
	defer cancel()

	workers := runtime.NumCPU()
	mctsResult := MCTS(ctx, game.Game.ID, reorderedBoard, math.MaxInt, workers, gameState)
	bestMove := determineBestMove(game, mctsResult)

	response := map[string]string{
		"move":  bestMove,
		"shout": "This is a nice move.",
	}
	writeJSON(w, response)

	// go func() {
	// reset this gamestate and load in new nodes
	gameStates[game.Game.ID] = make(map[string]*Node)
	saveNodesAtDepth2(mctsResult, gameStates[game.Game.ID])
	slog.Info("Move processed",
		"game_id", game.Game.ID,
		"snake_id", game.You.ID,
		"move", bestMove,
		"duration_ms", time.Since(start).Milliseconds(),
		"depth", mctsResult.Visits,
		"board", reorderedBoard,
	)
	// }()

	// slog.Info("Visualized board", "board", visualizeBoard(game.Board))
	fmt.Println(visualizeBoard(game.Board))
	// // Ensure the movetrees directory exists
	// if err := os.MkdirAll("movetrees", os.ModePerm); err != nil {
	// 	log.Println("Error creating movetrees directory:", err)
	// 	return
	// }
	// Generate and log the tree diagram
	// err := GenerateMostVisitedPathWithAlternativesHtmlTree(mctsResult)
	// if err != nil {
	// 	log.Println("Error saving mermaid tree:", err)
	// 	return
	// }
}

func saveNodesAtDepth2(rootNode *Node, gameStates map[string]*Node) {
	for _, child := range rootNode.Children {
		for _, grandchild := range child.Children {
			boardKey := boardHash(grandchild.Board)
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

	// tidy the cache
	delete(gameStates, game.Game.ID)

	slog.Info("Game ended", "game_id", game.Game.ID, "turns", game.Turn)

	writeJSON(w, map[string]string{})
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func boardHash(board Board) string {
	hash := ""
	for i, snake := range board.Snakes {
		for _, part := range snake.Body {
			hash += fmt.Sprintf("S%d%v%v", i, part.X, part.Y)
		}
	}
	for _, food := range board.Food {
		hash += fmt.Sprintf("f%v%v", food.X, food.Y)
	}

	for _, hazard := range board.Hazards {
		hash += fmt.Sprintf("h%v%v", hazard.X, hazard.Y)
	}

	return hash
}
