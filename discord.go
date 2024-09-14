package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
)

type Embed struct {
	Title       string       `json:"title,omitempty"`       // Title of the embed
	Type        string       `json:"type,omitempty"`        // Type of embed (always "rich" for embeds)
	Description string       `json:"description,omitempty"` // Description of the embed
	URL         string       `json:"url,omitempty"`         // URL of the embed
	Timestamp   string       `json:"timestamp,omitempty"`   // ISO8601 timestamp
	Color       int          `json:"color,omitempty"`       // Color code of the embed
	Footer      *Footer      `json:"footer,omitempty"`      // Footer object
	Image       *Image       `json:"image,omitempty"`       // Image object
	Thumbnail   *Thumbnail   `json:"thumbnail,omitempty"`   // Thumbnail object
	Video       *Video       `json:"video,omitempty"`       // Video object
	Provider    *Provider    `json:"provider,omitempty"`    // Provider object
	Author      *Author      `json:"author,omitempty"`      // Author object
	Fields      []EmbedField `json:"fields,omitempty"`      // Array of fields
}

type Footer struct {
	Text         string `json:"text,omitempty"`           // Footer text
	IconURL      string `json:"icon_url,omitempty"`       // URL of footer icon
	ProxyIconURL string `json:"proxy_icon_url,omitempty"` // Proxied URL of footer icon
}

type Image struct {
	URL      string `json:"url,omitempty"`       // URL of the image
	ProxyURL string `json:"proxy_url,omitempty"` // Proxied URL of the image
	Height   int    `json:"height,omitempty"`    // Height of the image
	Width    int    `json:"width,omitempty"`     // Width of the image
}

type Thumbnail struct {
	URL      string `json:"url,omitempty"`       // URL of the thumbnail
	ProxyURL string `json:"proxy_url,omitempty"` // Proxied URL of the thumbnail
	Height   int    `json:"height,omitempty"`    // Height of the thumbnail
	Width    int    `json:"width,omitempty"`     // Width of the thumbnail
}

type Video struct {
	URL    string `json:"url,omitempty"`    // URL of the video
	Height int    `json:"height,omitempty"` // Height of the video
	Width  int    `json:"width,omitempty"`  // Width of the video
}

type Provider struct {
	Name string `json:"name,omitempty"` // Name of the provider
	URL  string `json:"url,omitempty"`  // URL of the provider
}

type Author struct {
	Name         string `json:"name,omitempty"`           // Name of the author
	URL          string `json:"url,omitempty"`            // URL of the author
	IconURL      string `json:"icon_url,omitempty"`       // URL of the author icon
	ProxyIconURL string `json:"proxy_icon_url,omitempty"` // Proxied URL of author icon
}

type EmbedField struct {
	Name   string `json:"name"`   // Name of the field
	Value  string `json:"value"`  // Value of the field
	Inline bool   `json:"inline"` // Whether the field is inline
}

type WebhookPayload struct {
	Content string  `json:"content,omitempty"`
	Embeds  []Embed `json:"embeds,omitempty"`
}

func sendDiscordWebhook(webhookURL, message string, embeds []Embed) error {
	// Create the payload with the embed
	payload := WebhookPayload{
		Embeds:  embeds,
		Content: message,
	}

	// Marshal the payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Send the HTTP POST request to the webhook URL
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return err
	}

	slog.Debug("discord message sent")
	return nil
}
