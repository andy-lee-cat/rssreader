// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ai // import "miniflux.app/v2/internal/ai"

import (
	"fmt"
	"os"
)

// NewProvider creates a new SummaryProvider based on the configuration.
func NewProvider(cfg *Config) (SummaryProvider, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	switch cfg.Provider {
	case "mock":
		return NewMockProvider(), nil
	case "langgraph", "":
		// Default to LangGraph provider
		baseURL := os.Getenv("AI_SERVICE_URL")
		if baseURL == "" {
			baseURL = "http://localhost:5000"
		}
		return NewLangGraphProvider(baseURL), nil
	default:
		return nil, fmt.Errorf("unknown AI provider: %s", cfg.Provider)
	}
}

// defaultProvider is the singleton instance of the default provider.
var defaultProvider SummaryProvider

// GetDefaultProvider returns the default provider.
// Uses AI_SERVICE_URL environment variable if set, otherwise defaults to localhost:5000.
func GetDefaultProvider() SummaryProvider {
	if defaultProvider == nil {
		baseURL := os.Getenv("AI_SERVICE_URL")
		if baseURL == "" {
			baseURL = "http://localhost:5000"
		}
		defaultProvider = NewLangGraphProvider(baseURL)
	}
	return defaultProvider
}

// SetDefaultProvider sets the default provider.
func SetDefaultProvider(p SummaryProvider) {
	defaultProvider = p
}
