package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"time"
)

// Define a custom handler for Google Cloud logging
type GoogleCloudHandler struct {
	writer     *os.File
	level      slog.Level
	extraAttrs map[string]interface{} // Store additional attributes
}

// NewGoogleCloudHandler creates a new handler for Google Cloud
func NewGoogleCloudHandler(writer *os.File, level slog.Level) *GoogleCloudHandler {
	return &GoogleCloudHandler{
		writer: writer,
		level:  level,
	}
}

// Enabled checks if the log level is enabled
func (h *GoogleCloudHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle writes the log entry in JSON format for Google Cloud
func (h *GoogleCloudHandler) Handle(_ context.Context, r slog.Record) error {
	severity := convertToSeverity(r.Level)

	// Collect attributes as a map
	attrs := map[string]interface{}{}
	r.Attrs(func(attr slog.Attr) bool {
		attrs[attr.Key] = attr.Value.Any()
		return true
	})

	// Merge any extra attributes (from WithAttrs) with the current ones
	for k, v := range h.extraAttrs {
		attrs[k] = v
	}

	// Structure the log entry for Google Cloud
	logEntry := map[string]interface{}{
		"severity": severity,
		"message":  r.Message,
		"time":     time.Now().Format(time.RFC3339Nano),
	}

	// Merge additional attributes into the log entry
	for k, v := range attrs {
		logEntry[k] = v
	}

	// Encode the log entry as JSON and write it to the output
	encoder := json.NewEncoder(h.writer)
	if err := encoder.Encode(logEntry); err != nil {
		return err
	}
	return nil
}

// WithAttrs returns a new handler with additional attributes
func (h *GoogleCloudHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// Add the new attributes to the current handler and return it
	newHandler := *h
	if newHandler.extraAttrs == nil {
		newHandler.extraAttrs = map[string]interface{}{}
	}
	for _, attr := range attrs {
		newHandler.extraAttrs[attr.Key] = attr.Value.Any()
	}
	return &newHandler
}

// WithGroup returns a new handler that adds a group for the next logs
func (h *GoogleCloudHandler) WithGroup(name string) slog.Handler {
	// For simplicity, we will just return a copy of the handler
	return h
}

// Converts slog.Level to Google Cloud severity strings
func convertToSeverity(level slog.Level) string {
	switch level {
	case slog.LevelInfo:
		return "INFO"
	case slog.LevelWarn:
		return "WARNING"
	case slog.LevelError:
		return "ERROR"
	case slog.LevelDebug:
		return "DEBUG"
	default:
		return "DEFAULT"
	}
}
