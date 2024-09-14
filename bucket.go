package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"cloud.google.com/go/storage"
)

// downloadAndUploadFile streams the file from the URL and uploads it directly to the Google Cloud Storage bucket.
func downloadAndUploadFile(ctx context.Context, gameID string) error {

	url := fmt.Sprintf("https://exporter.battlesnake.com/games/%s/gif", gameID)
	bucketName := "gregorywebp"
	// Make a GET request to the URL
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	// Check if the response is OK (status code 200)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Create a Google Cloud Storage client
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create storage client: %w", err)
	}
	defer client.Close()

	// Get a reference to the bucket and object (file)
	bucket := client.Bucket(bucketName)
	object := bucket.Object(fmt.Sprintf("%s.gif", gameID))

	// Create a new writer for the object in the bucket
	writer := object.NewWriter(ctx)

	// Stream the file from the URL directly to the bucket
	_, err = io.Copy(writer, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to copy data to bucket: %w", err)
	}

	// Close the writer to complete the upload
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	slog.Debug("file uploaded", "game_id", gameID)
	return nil
}
