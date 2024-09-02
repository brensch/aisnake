package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"
)

func main() {

	numCPUs := runtime.NumCPU()
	fmt.Printf("Number of available CPUs: %d\n", numCPUs)
	// Define the test game state
	gameState := BattleSnakeGame{
		Game: Game{
			ID: "game-id-string",
			Ruleset: Ruleset{
				Name:    "standard",
				Version: "1.0.0",
				Settings: Settings{
					FoodSpawnChance:     15,
					MinimumFood:         1,
					HazardDamagePerTurn: 0,
				},
			},
			Map:     "standard",
			Source:  "standard",
			Timeout: 500,
		},
		Turn: 10,
		Board: Board{
			Height: 11,
			Width:  11,
			Food: []Point{
				{X: 5, Y: 5},
			},
			Hazards: []Point{},
			Snakes: []Snake{
				{
					ID:     "snake-id-1",
					Name:   "My Snake",
					Health: 90,
					Body: []Point{
						{X: 1, Y: 1},
						{X: 1, Y: 2},
						{X: 1, Y: 3},
					},
					Latency: "123",
					Head:    Point{X: 1, Y: 1},
					Shout:   "I'm hungry!",
					Customizations: Customizations{
						Color: "#FF0000",
						Head:  "default",
						Tail:  "default",
					},
				},
				{
					ID:     "snake-id-2",
					Name:   "Opponent Snake",
					Health: 80,
					Body: []Point{
						{X: 9, Y: 9},
						{X: 9, Y: 8},
						{X: 9, Y: 7},
					},
					Latency: "456",
					Head:    Point{X: 9, Y: 9},
					Shout:   "I'm coming for you!",
					Customizations: Customizations{
						Color: "#00FF00",
						Head:  "beluga",
						Tail:  "pixel",
					},
				},
			},
		},
		You: Snake{
			ID:     "snake-id-1",
			Name:   "My Snake",
			Health: 90,
			Body: []Point{
				{X: 1, Y: 1},
				{X: 1, Y: 2},
				{X: 1, Y: 3},
			},
			Latency: "123",
			Head:    Point{X: 1, Y: 1},
			Shout:   "I'm hungry!",
			Customizations: Customizations{
				Color: "#FF0000",
				Head:  "default",
				Tail:  "default",
			},
		},
	}

	// Marshal the game state into JSON
	jsonData, err := json.Marshal(gameState)
	if err != nil {
		fmt.Printf("Error marshaling game state: %v\n", err)
		return
	}

	// Send the POST request to the BattleSnake API
	url := "http://localhost:8080/move"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: time.Second * 10}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Read and print the response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("Error decoding response: %v\n", err)
		return
	}

	fmt.Printf("Response from BattleSnake API: %v\n", result)
}

// lazy, will tidy
type Game struct {
	ID      string  `json:"id"`
	Ruleset Ruleset `json:"ruleset"`
	Map     string  `json:"map"`
	Source  string  `json:"source"`
	Timeout int     `json:"timeout"`
}

type Ruleset struct {
	Name     string   `json:"name"`
	Version  string   `json:"version"`
	Settings Settings `json:"settings"`
}

type Settings struct {
	FoodSpawnChance     int `json:"foodSpawnChance"`
	MinimumFood         int `json:"minimumFood"`
	HazardDamagePerTurn int `json:"hazardDamagePerTurn"`
}

type Board struct {
	Height  int     `json:"height"`
	Width   int     `json:"width"`
	Food    []Point `json:"food"`
	Hazards []Point `json:"hazards"`
	Snakes  []Snake `json:"snakes"`
}

type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Snake struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Health  int     `json:"health"`
	Body    []Point `json:"body"`
	Latency string  `json:"latency"`
	Head    Point   `json:"head"`
	// Length         int            `json:"length"`
	Shout          string         `json:"shout"`
	Customizations Customizations `json:"customizations"`
}

type Customizations struct {
	Color string `json:"color"`
	Head  string `json:"head"`
	Tail  string `json:"tail"`
}

type BattleSnakeGame struct {
	Game  Game  `json:"game"`
	Turn  int   `json:"turn"`
	Board Board `json:"board"`
	You   Snake `json:"you"`
}
