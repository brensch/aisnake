package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

const (
	apiURL   = "https://api.tidbyt.com/v0/devices/%s/push"
	deviceID = "jocundly-liberated-allied-panda-3f1"
)

type PushRequest struct {
	Image          string `json:"image"`
	InstallationID string `json:"installationID,omitempty"`
	Background     bool   `json:"background"`
}

func PushToTidbyt(deviceID, webpBase64 string) error {

	// Prepare the request body
	requestBody := PushRequest{
		Image:      webpBase64,
		Background: false, // Set to true if you want to push the image in the background
	}

	// Serialize request to JSON
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	// Send the POST request to Tidbyt
	pushURL := fmt.Sprintf(apiURL, deviceID)
	req, err := http.NewRequest("POST", pushURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tidbytSecret))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to Tidbyt API: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("tidbyt API returned status: %v", resp.Status)
	}

	slog.Info("Image successfully pushed to Tidbyt")
	return nil
}
