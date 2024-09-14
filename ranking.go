package main

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
)

const (
	brenschProfile = "https://play.battlesnake.com/profile/brensch" // you're the only one who matters
)

// GetDuelsRankAndScore fetches the profile page and extracts the Duels rank and score
func GetDuelsRankAndScore() (rank, score int, err error) {
	// Perform HTTP GET request
	resp, err := http.Get(brenschProfile)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to retrieve URL: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read response body: %w", err)
	}

	// Convert body to string for regex matching
	bodyStr := string(body)

	// Regex to extract Duels score
	scoreRegex := regexp.MustCompile(`<p class="text-4xl text-center font-bold">([\d,]+)</p>`)
	scoreMatch := scoreRegex.FindStringSubmatch(bodyStr)
	if len(scoreMatch) < 2 {
		return 0, 0, fmt.Errorf("failed to find Duels score")
	}
	// Remove commas and convert score to integer
	scoreStr := scoreMatch[1]
	scoreStr = regexp.MustCompile(",").ReplaceAllString(scoreStr, "")
	score, err = strconv.Atoi(scoreStr)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to convert score to integer: %w", err)
	}

	// Regex to extract Duels rank
	rankRegex := regexp.MustCompile(`<p class="text-lg text-center text-sm">Rank: (\d+)</p>`)
	rankMatch := rankRegex.FindStringSubmatch(bodyStr)
	if len(rankMatch) < 2 {
		return 0, 0, fmt.Errorf("failed to find Duels rank")
	}
	rank, err = strconv.Atoi(rankMatch[1])
	if err != nil {
		return 0, 0, fmt.Errorf("failed to convert rank to integer: %w", err)
	}

	return rank, score, nil
}
