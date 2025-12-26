// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ai // import "miniflux.app/v2/internal/ai"

import (
	"context"
)

// SummaryProvider defines the interface for AI summary generation.
// Implementations can be mock, OpenAI, Claude, or any other AI service.
type SummaryProvider interface {
	// GenerateSummary generates a summary for the given content.
	// userID is used to look up the user's AI configuration (API key, model, etc.)
	// It returns a channel that streams the summary text chunk by chunk.
	// The channel is closed when the summary is complete or an error occurs.
	// If an error occurs, it will be sent as the last message prefixed with "error:".
	GenerateSummary(ctx context.Context, userID int64, content string) (<-chan string, error)

	// Name returns the name of the provider for logging purposes.
	Name() string
}

// Config holds the configuration for AI services.
type Config struct {
	Provider    string // Provider name: "mock", "openai", "claude", etc.
	APIKey      string // API key for the provider
	Model       string // Model to use (e.g., "gpt-4", "claude-3")
	MaxTokens   int    // Maximum tokens for the summary
	Temperature float64 // Temperature for generation
}

// DefaultConfig returns a default configuration.
func DefaultConfig() *Config {
	return &Config{
		Provider:    "mock",
		MaxTokens:   500,
		Temperature: 0.7,
	}
}
