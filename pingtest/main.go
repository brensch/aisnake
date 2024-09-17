package main

import (
	"encoding/json"
	"net/http"
)

func handleIndex(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"apiversion": "1",
		"author":     "",           // Optional: Add your username
		"color":      "#888888",    // Optional: Customize your Battlesnake color
		"head":       "default",    // Optional: Choose a head design
		"tail":       "default",    // Optional: Choose a tail design
		"version":    "0.0.1-beta", // Optional: Your Battlesnake version
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	http.HandleFunc("/", handleIndex)
	http.ListenAndServe(":8080", nil)
}
