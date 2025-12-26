package main

import (
	"log/slog"
	"testing"
)

func TestRetrieveGameRenderAndSendToTidbyt(t *testing.T) {

	testCases := []struct {
		Description string
		GameID      string
	}{
		{
			Description: "two player game",
			GameID:      "01f75b47-80eb-4062-a345-b256f7187809",
		},
	}

	tidBytSecretName := "projects/680796481131/secrets/tidbyt/versions/latest"
	tidbytSecret, err := getSecret(tidBytSecretName)
	if err != nil {
		slog.Error("Failed to retrieve tidbyt webhook secret", "error", err.Error())
	}
	for _, test := range testCases {
		t.Run(test.Description, func(t *testing.T) {

			RetrieveGameRenderAndSendToTidbyt(test.GameID, tidbytSecret)
		})
	}

}
