package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCompetitionResults(t *testing.T) {
	results, err := GetCompetitionResults()
	assert.NoError(t, err, "should not have an error getting rankings")

	for _, result := range results {
		t.Log(result.Name, result.Rank, result.Score)
	}
}
