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
	"path/filepath"
	"time"

	"github.com/google/uuid"
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
	var game BattleSnakeGame
	err := json.NewDecoder(r.Body).Decode(&game)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Println("Received move request for snake", game.You.ID)

	reorderedBoard := reorderSnakes(game.Board, game.You.ID)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(game.Game.Timeout-100)*time.Millisecond)
	defer cancel()

	mctsResult := MCTS(ctx, reorderedBoard, math.MaxInt)

	bestMove := determineBestMove(game, mctsResult)

	response := map[string]string{
		"move":  bestMove,
		"shout": "This is a nice move.",
	}
	writeJSON(w, response)

	// Ensure the movetrees directory exists
	err = os.MkdirAll("movetrees", os.ModePerm)
	if err != nil {
		log.Println("Error creating movetrees directory:", err)
		return
	}

	err = writeNodeAsMermaidToHTMLFile(mctsResult)
	if err != nil {
		log.Println("Error saving mermaid tree:", err)
		return
	}

	fmt.Println(visualizeBoard(game.Board))

	// Log the filename with a format that makes it clickable in VS Code terminal
	log.Println("Made move:", bestMove, "in", time.Since(start).Milliseconds(), "ms with", mctsResult.Visits, "visits")

}

func writeNodeAsMermaidToHTMLFile(node *Node) error {
	// Generate Mermaid content
	timestamp := time.Now().Format("20060102_150405")
	uuid := uuid.New().String()
	filename := filepath.Join("movetrees", fmt.Sprintf("%s_%s.html", timestamp, uuid))
	mermaidContent := GenerateMostVisitedPathWithAlternativesMermaidTree(node)

	// Write the Mermaid diagram to the file with a proper HTML template
	htmlContent := fmt.Sprintf(`
	<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>Mermaid Diagram</title>
		<style>
			body {
				margin: 0;
				font-family: Arial, sans-serif;
				display: flex;
				justify-content: center;
				align-items: center;
				height: 100vh;
				overflow: hidden;
			}
			.mermaid-container {
				width: 100%%;
				max-width: 1000px;
				height: 80vh;
				overflow: auto;
				border: 1px solid #ccc;
				padding: 10px;
			}
		</style>
		<script type="module">
			import mermaid from 'https://cdn.jsdelivr.net/npm/mermaid@10/dist/mermaid.esm.min.mjs';
			mermaid.initialize({ 
				startOnLoad: true,
				maxTextSize: 100000000 // Increase max character count
			});

			// Add zoom functionality
			document.addEventListener('keydown', (event) => {
				const svgElement = document.querySelector('.mermaid svg');
				let scale = parseFloat(svgElement.getAttribute('data-scale')) || 1;

				if (event.ctrlKey && event.key === '+') { // Zoom in
					scale += 0.1;
				} else if (event.ctrlKey && event.key === '-') { // Zoom out
					scale = Math.max(0.1, scale - 0.1);
				}

				svgElement.setAttribute('data-scale', scale);
				svgElement.style.transform = 'scale(' + scale + ')';
				svgElement.style.transformOrigin = '0 0';
			});
		</script>
	</head>
	<body>
		<div class="mermaid-container">
			<div class="mermaid">
				%s
			</div>
		</div>
	</body>
	</html>`, mermaidContent)

	fmt.Printf("Generated move tree: %s\nFile: %s\n", uuid, filepath.Join(".", filename))

	return os.WriteFile(filename, []byte(htmlContent), 0644)
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
