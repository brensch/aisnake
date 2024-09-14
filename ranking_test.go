package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetDuelsRankAndScore(t *testing.T) {

	start := time.Now()
	// Call the function with the mock server URL
	rank, score, err := GetDuelsRankAndScore()

	t.Log(rank, score, time.Since(start))
	// Assertions
	assert.NoError(t, err)
	assert.NotZero(t, rank, "Rank should not be 0")
	assert.NotZero(t, score, "Score should be 9,166")
}
