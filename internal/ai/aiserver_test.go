// SPDX-FileCopyrightText: Copyright Andy. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ai_test

import (
	"context"
	"testing"
	"time"
	"unicode/utf8"
)

type MockAIServer struct {
	CharDelay time.Duration
}

func NewMockAIServer() *MockAIServer {
	return &MockAIServer{
		CharDelay: 50 * time.Millisecond,
	}
}

func (m *MockAIServer) Name() string {
	return "mock"
}

// GenerateSummary generates a mock summary by returning the first 100 characters.
// It streams the result character by character to simulate AI generation.
// userID is ignored in mock implementation.
func (m *MockAIServer) GenerateSummary(ctx context.Context, userID int64, content string) (<-chan string, error) {
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
func (m *MockAIServer) truncateToRunes(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}

func TestAIServer(t *testing.T) {
	t.Run("Get Server Name", func(t *testing.T) {
		monkAIServer := NewMockAIServer()
		assertEqual(t, "mock", monkAIServer.Name())
	})

}

func assertEqual(t *testing.T, expected, actual string) {
	t.Helper()
	if expected != actual {
		t.Errorf("Expected %s, but got %s", expected, actual)
	}
}
