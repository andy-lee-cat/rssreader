// SPDX-FileCopyrightText: Copyright Andy. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ai // import "miniflux.app/v2/internal/ai"

import (
	"context"
	"os"
)

// SummaryProvider defines the interface for AI summary generation.
// Implementations can be mock, OpenAI, Claude, or any other AI service.
type AIServer interface {
	// GenerateSummary generates a summary for the given content.
	// userID is used to look up the user's AI configuration (API key, model, etc.)
	// It returns a channel that streams the summary text chunk by chunk.
	// The channel is closed when the summary is complete or an error occurs.
	// If an error occurs, it will be sent as the last message prefixed with "error:".
	GenerateSummary(ctx context.Context, userID int64, content string) (<-chan string, error)

	// Name returns the name of the provider for logging purposes.
	Name() string
}

// defaultAIServer is the singleton instance of the default AIServer.
var defaultAIServer AIServer

// GetDefaultAIServer returns the default AIServer.
// Uses AI_SERVICE_URL environment variable if set, otherwise defaults to localhost:5000.
func GetAIServer() AIServer {
	if defaultAIServer == nil {
		baseURL := os.Getenv("AI_SERVICE_URL")
		if baseURL == "" {
			baseURL = "http://localhost:5000"
		}
		defaultAIServer = NewLangGraphAIServer(baseURL)
	}
	return defaultAIServer
}
