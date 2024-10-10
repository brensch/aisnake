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
	"strconv"
	"strings"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

// things i want to track from the start to the end that don't get provided by the server
type GameMeta struct {
	otherSnakes []string
	start       time.Time
}

var (
	gameMetaRegistry = make(map[string]GameMeta)         // this is needed since final game states don't necessarily have all snakes
	gameStates       = make(map[string]map[string]*Node) // Global map to store known game states

	loc *time.Location
)

const lagBufferMS = 150

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
	handler := NewGoogleCloudHandler(os.Stdout, slog.LevelDebug)

	// Create a new logger using the custom handler
	logger := slog.New(handler)

	// Set the logger as default
	slog.SetDefault(logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	var err error

	loc, err = time.LoadLocation("America/Los_Angeles")
	if err != nil {
		slog.Error("failed to load tz", "error", err.Error())
		loc = time.UTC
	}

	// Retrieve Discord webhook URL from Google Secret Manager
	secretName := "projects/680796481131/secrets/discord_webhook/versions/latest"
	webhookURL, err := getSecret(secretName)
	if err != nil {
		slog.Error("Failed to retrieve Discord webhook secret", "error", err.Error())
	}

	tidBytSecretName := "projects/680796481131/secrets/tidbyt/versions/latest"
	tidbytSecret, err := getSecret(tidBytSecretName)
	if err != nil {
		slog.Error("Failed to retrieve tidbyt webhook secret", "error", err.Error())
	}

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/start", handleStart(webhookURL))
	http.HandleFunc("/move", handleMove)
	http.HandleFunc("/end", handleEnd(tidbytSecret, webhookURL))

	slog.Debug("Starting BattleSnake on port", "port", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"apiversion": "1",
		"author":     "brensch",
		"color":      "#00ff00",
		"head":       "replit-mark",
		"tail":       "replit-notmark",
		"version":    "0.1.0",
	}
	writeJSON(w, response)
}

func handleStart(webhookURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var game BattleSnakeGame
		if err := json.NewDecoder(r.Body).Decode(&game); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// add a map for this game
		gameStates[game.Game.ID] = make(map[string]*Node)
		var otherSnakes []string
		foundPaul := false
		for _, snake := range game.Board.Snakes {
			if snake.Name == game.You.Name {
				continue
			}
			if snake.Name == "Cucumber Cat" {
				foundPaul = true
			}
			otherSnakes = append(otherSnakes, snake.Name)
		}
		if foundPaul {
			sendDiscordWebhook(webhookURL, fmt.Sprintf("Paul Alert: https://play.battlesnake.com/game/%s", game.Game.ID), []Embed{})
		}
		gameMetaRegistry[game.Game.ID] = GameMeta{
			otherSnakes: otherSnakes,
			start:       time.Now(),
		}
		slog.Info("Game started", "game_id", game.Game.ID, "you", game.You, "other_snakes", otherSnakes)

		writeJSON(w, map[string]string{})
	}
}

func handleMove(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var game BattleSnakeGame
	if err := json.NewDecoder(r.Body).Decode(&game); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log := slog.With("game_id", game.Game.ID, "turn", game.Turn)

	latency, err := strconv.Atoi(game.You.Latency)
	// sends "" on first move
	if game.You.Latency != "" && err != nil {
		log.Error("latency not an int", "error", err.Error())
	}

	allowedThinkingTime := game.Game.Timeout - lagBufferMS

	if latency == game.Game.Timeout {
		log.Error("timed out on last move", "latency", latency)
		// double the buffer if we timed out last turn
		// allowedThinkingTime = allowedThinkingTime - lagBufferMS
	}

	// get the nodemap for this game
	gameState, ok := gameStates[game.Game.ID]
	if !ok {
		log.Error("failed to find gamestate. probably reset during a game.")
		gameState = make(map[string]*Node)
	}

	reorderedBoard := reorderSnakes(game.Board, game.You.ID)
	// fmt.Println(visualizeBoard(reorderedBoard))
	// b, _ := json.Marshal(reorderedBoard)
	// fmt.Println(string(b))

	// timeout to signify end of move
	ctx, cancel := context.WithDeadline(context.Background(), start.Add(time.Duration(allowedThinkingTime)*time.Millisecond))
	defer cancel()

	workers := runtime.NumCPU()
	mctsResult := MCTS(ctx, log, game.Game.ID, reorderedBoard, math.MaxInt, workers, gameState)
	bestMove := determineBestMove(mctsResult)
	// mctsResult := MultiMCTS(ctx, game.Game.ID, reorderedBoard, math.MaxInt, workers, map[string]*MultiNode{})
	// bestMove := MultiDetermineBestMove(mctsResult, 0)
	response := map[string]string{
		"move":  bestMove,
		"shout": fmt.Sprintf("I pondered the orb %d times in %dms. It was nice.", mctsResult.Visits, time.Since(start).Milliseconds()),
	}
	writeJSON(w, response)

	log.Info("Move processed",
		"game", game,
		"move", bestMove,
		"duration_ms", time.Since(start).Milliseconds(),
		"depth", mctsResult.Visits,
	)

	// fmt.Println("yoooooooooo", bestMove)
	// reset this gamestate and load in new nodes
	gameSaveStart := time.Now()
	gameStates[game.Game.ID] = make(map[string]*Node)
	saveNodesAtDepth2(mctsResult, gameStates[game.Game.ID])
	log.Debug("finished saving game state", "duration", time.Since(gameSaveStart).Milliseconds())
	fmt.Println(mctsResult.Visits)

	// slog.Info("Visualized board", "board", visualizeBoard(game.Board))
	// fmt.Println(visualizeBoard(reorderedBoard))
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

func determineBestMove(node *Node) string {
	var bestChild *Node
	maxVisits := int64(-1)

	for _, child := range node.Children {
		if child.Visits > maxVisits {
			bestChild = child
			maxVisits = child.Visits
		}
	}

	if bestChild != nil {
		bestMove := determineMoveDirection(node.Board.Snakes[0].Head, bestChild.Board.Snakes[0].Head)
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

func handleEnd(tidBytSecret, webhookURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		end := time.Now()
		var game BattleSnakeGame
		err := json.NewDecoder(r.Body).Decode(&game)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		outcome, description := describeGameOutcome(game)
		var outcomeEmoji string

		switch outcome {
		case Win:
			outcomeEmoji = "✅"
		case Loss:
			outcomeEmoji = "❌"
		case Draw:
			outcomeEmoji = "🦍"
		}

		ranks, err := GetCompetitionResults()
		if err != nil {
			slog.Error("failed to get ranks", "error", err)
		}

		gameMeta, ok := gameMetaRegistry[game.Game.ID]
		if !ok {
			gameMeta = GameMeta{
				otherSnakes: []string{"server reset during game"},
				start:       time.Now(),
			}
		}

		gameDuration := end.Sub(gameMeta.start)

		slog.Info("Game ended", "game", game, "ranks", ranks, "duration_ms", gameDuration.Milliseconds())

		err = sendDiscordWebhook(webhookURL, fmt.Sprintf("%s [%s](<https://play.battlesnake.com/game/%s>) | %s", outcomeEmoji, strings.Join(gameMeta.otherSnakes, ", "), game.Game.ID, description), []Embed{})
		if err != nil {
			slog.Error("failed to send discord webhook", "error", err.Error())
		}
		err = downloadAndUploadFile(context.Background(), game.Game.ID)
		if err != nil {
			slog.Error("failed to download and upload", "error", err.Error())
		}
		// if err != nil {
		// } else {
		// 	sendDiscordWebhook(
		// 		webhookURL,
		// 		"",
		// 		[]Embed{
		// 			{
		// 				Title:       strings.Join(gameMeta.otherSnakes, ", "),
		// 				Description: description,
		// 				Image: &Image{
		// 					URL: fmt.Sprintf("https://storage.googleapis.com/gregorywebp/%s.gif", game.Game.ID),
		// 				},
		// 				Color: getColorForOutcome(outcome),
		// 				URL:   fmt.Sprintf("https://play.battlesnake.com/game/%s", game.Game.ID),
		// 				Fields: []EmbedField{
		// 					{
		// 						Name:   "turns",
		// 						Value:  fmt.Sprint(game.Turn),
		// 						Inline: true,
		// 					},
		// 					{
		// 						Name:   "latency",
		// 						Value:  game.You.Latency,
		// 						Inline: true,
		// 					},
		// 					{
		// 						Name:   "rank",
		// 						Value:  fmt.Sprint(rank),
		// 						Inline: true,
		// 					},
		// 					{
		// 						Name:   "score",
		// 						Value:  fmt.Sprint(score),
		// 						Inline: true,
		// 					},
		// 					{
		// 						Name:   "game duration",
		// 						Value:  fmt.Sprint(gameDuration.String()),
		// 						Inline: true,
		// 					},
		// 				},
		// 				Footer: &Footer{
		// 					Text: time.Now().In(loc).Format(time.RFC3339),
		// 				},
		// 			},
		// 		},
		// 	)
		// }

		RetrieveGameRenderAndSendToTidbyt(tidBytSecret, game.Game.ID)

		writeJSON(w, map[string]string{})
	}
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
