// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ai // import "miniflux.app/v2/internal/ai"

import (
	"context"
	"time"
	"unicode/utf8"
)

// MockProvider is a mock implementation of SummaryProvider for testing.
// It returns the first 100 characters of the content as the summary.
type MockProvider struct {
	// CharDelay is the delay between each character (for streaming effect).
	CharDelay time.Duration
}

// NewMockProvider creates a new MockProvider with default settings.
func NewMockProvider() *MockProvider {
	return &MockProvider{
		CharDelay: 50 * time.Millisecond,
	}
}

// Name returns the provider name.
func (m *MockProvider) Name() string {
	return "mock"
}

// GenerateSummary generates a mock summary by returning the first 100 characters.
// It streams the result character by character to simulate AI generation.
func (m *MockProvider) GenerateSummary(ctx context.Context, content string) (<-chan string, error) {
	ch := make(chan string)

	go func() {
		defer close(ch)

		// Generate mock summary: first 100 characters
		summary := m.truncateToRunes(content, 100)
		if utf8.RuneCountInString(content) > 100 {
			summary += "..."
		}

		// Stream character by character
		for _, r := range summary {
			select {
			case <-ctx.Done():
				return
			default:
				ch <- string(r)
				if m.CharDelay > 0 {
					time.Sleep(m.CharDelay)
				}
			}
		}
	}()

	return ch, nil
}

// truncateToRunes truncates a string to the specified number of runes.
func (m *MockProvider) truncateToRunes(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}
