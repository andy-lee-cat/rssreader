// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ai // import "miniflux.app/v2/internal/ai"

import (
	"fmt"
)

// NewProvider creates a new SummaryProvider based on the configuration.
func NewProvider(cfg *Config) (SummaryProvider, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	switch cfg.Provider {
	case "mock", "":
		return NewMockProvider(), nil
	// TODO: Add more providers here
	// case "openai":
	//     return NewOpenAIProvider(cfg), nil
	// case "claude":
	//     return NewClaudeProvider(cfg), nil
	default:
		return nil, fmt.Errorf("unknown AI provider: %s", cfg.Provider)
	}
}

// defaultProvider is the singleton instance of the default provider.
var defaultProvider SummaryProvider

// GetDefaultProvider returns the default provider (mock for now).
func GetDefaultProvider() SummaryProvider {
	if defaultProvider == nil {
		defaultProvider = NewMockProvider()
	}
	return defaultProvider
}

// SetDefaultProvider sets the default provider.
func SetDefaultProvider(p SummaryProvider) {
	defaultProvider = p
}
