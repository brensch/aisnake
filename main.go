package main

import (
	"bytes"
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
	"strings"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

var (
	gameStates = make(map[string]map[string]*Node) // Global map to store known game states
	// TODO: make this non global
	webhookURL   string = ""
	tidbytSecret string = ""
)

// Struct for Discord Webhook payload
type WebhookPayload struct {
	Content string `json:"content"`
}

func sendDiscordWebhook(webhookURL, message string) {
	slog.Warn("discord message", "payload", message)
	if webhookURL == "" {
		// If webhook URL is empty, log the message instead
		slog.Info("No webhook URL found, logging message instead", "message", message)
		return
	}

	payload := WebhookPayload{
		Content: message,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		slog.Error("failed to marshal payload", "err", err)
		return
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		slog.Error("failed to send discord webhook", "err", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		slog.Error("received non ok message", "code", resp.StatusCode)
		return
	}

}

func getSecret(secretName string) (string, error) {
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create secret manager client: %w", err)
	}
	defer client.Close()

	// Build the request.
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: secretName,
	}

	// Call the API.
	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to access secret version: %w", err)
	}

	// Extract the secret payload.
	payload := result.Payload.GetData()
	return string(payload), nil
}

func main() {

	// Set up the custom handler for Google Cloud
	handler := NewGoogleCloudHandler(os.Stdout, slog.LevelInfo)

	// Create a new logger using the custom handler
	logger := slog.New(handler)

	// Set the logger as default
	slog.SetDefault(logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	slog.Info("Starting BattleSnake on port", "port", port)

	// Retrieve Discord webhook URL from Google Secret Manager
	secretName := "projects/680796481131/secrets/discord_webhook/versions/latest"
	var err error
	webhookURL, err = getSecret(secretName)
	if err != nil {
		slog.Error("Failed to retrieve Discord webhook secret", "error", err.Error())
		webhookURL = "" // Ensure webhookURL is empty if retrieval fails
	}

	tidBytSecretName := "projects/680796481131/secrets/tidbyt/versions/latest"
	tidbytSecret, err = getSecret(tidBytSecretName)
	if err != nil {
		slog.Error("Failed to retrieve Discord webhook secret", "error", err.Error())
		webhookURL = "" // Ensure webhookURL is empty if retrieval fails
	}

	// Try to send a test message via webhook
	sendDiscordWebhook(webhookURL, "Starting up")

	defer func() {
		sendDiscordWebhook(webhookURL, "Shutting down")
	}()

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/start", handleStart)
	http.HandleFunc("/move", handleMove)
	http.HandleFunc("/end", handleEnd)

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
	var otherSnakes []string
	for _, snake := range game.Board.Snakes {
		if snake.ID == game.You.ID {
			continue
		}
		otherSnakes = append(otherSnakes, snake.Name)
	}

	sendDiscordWebhook(webhookURL, fmt.Sprintf("Game %s started against %s", game.Game.ID, strings.Join(otherSnakes, ",")))

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
	// 100ms safety timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(game.Game.Timeout-100)*time.Millisecond)
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
	// fmt.Println(visualizeBoard(game.Board))
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

	slog.Info("Game ended", "game", game)

	outcome := describeGameOutcome(game)
	// put you back onto the board if you died
	game.Board.Snakes = append(game.Board.Snakes, game.You)
	board := visualizeBoard(reorderSnakes(game.Board, game.You.ID), WithNewlineCharacter("\n"))

	sendDiscordWebhook(webhookURL, fmt.Sprintf("Game %s finished on turn %d. %s.\nhttps://play.battlesnake.com/game/%s\n```\n%s\n```", game.Game.ID, game.Turn, outcome, game.Game.ID, board))

	// WebSocket URL for the game
	wsURL := fmt.Sprintf("wss://engine.battlesnake.com/games/%s/events", game.Game.ID)

	// Collect game frames
	frames, won, err := collectGameFrames(wsURL)
	if err != nil {
		slog.Error("Failed to collect game frames", "error", err.Error())
	}
	slog.Info("got frames from websocket", "turns", len(frames))

	// Render frames to WebP and push to Tidbyt
	err = renderGameToGIF(frames, deviceID, won)
	if err != nil {
		slog.Error("Failed to render game to gif", "error", err.Error())
	}

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

func describeGameOutcome(game BattleSnakeGame) string {
	// Check if you lost by colliding with a wall
	if game.You.Head.X < 0 || game.You.Head.X >= game.Board.Width || game.You.Head.Y < 0 || game.You.Head.Y >= game.Board.Height {
		return "You lost by crashing into a wall."
	}

	// Check if you lost by colliding with another snake
	for _, snake := range game.Board.Snakes {
		if snake.ID != game.You.ID {
			for _, segment := range snake.Body {
				if game.You.Head == segment {
					return fmt.Sprintf("You lost by colliding with %s.", snake.Name)
				}
			}
		}
	}

	// Check if you lost by starving
	if game.You.Health <= 0 {
		return "You lost by starving to death."
	}

	// Check if all snakes died (a draw)
	livingSnakes := 0
	for _, snake := range game.Board.Snakes {
		if snake.Health > 0 {
			livingSnakes++
		}
	}
	if livingSnakes == 0 {
		return "It's a draw! All snakes died."
	}

	// Check if you won because all other snakes starved or collided
	if len(game.Board.Snakes) == 1 && game.Board.Snakes[0].ID == game.You.ID {
		// If only your snake remains, it means you won
		return "You won."
	}

	// Check if an opponent starved
	for _, snake := range game.Board.Snakes {
		if snake.ID != game.You.ID && snake.Health <= 0 {
			return fmt.Sprintf("You won because %s starved.", snake.Name)
		}
	}

	// Check if you lost by entering a hazard
	for _, hazard := range game.Board.Hazards {
		if game.You.Head == hazard {
			return "You lost by entering a hazard."
		}
	}

	// Default outcome
	return "Seems like a draw."
}
