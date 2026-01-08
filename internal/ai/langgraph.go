// SPDX-FileCopyrightText: Copyright Andy. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ai // import "miniflux.app/v2/internal/ai"

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// LangGraphAIServer calls the Python LangGraph AI service.
type LangGraphAIServer struct {
	baseURL    string
	httpClient *http.Client
}

// NewLangGraphAIServer creates a new LangGraphAIServer.
func NewLangGraphAIServer(baseURL string) *LangGraphAIServer {
	if baseURL == "" {
		baseURL = "http://localhost:5000"
	}
	return &LangGraphAIServer{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute, // Long timeout for streaming
		},
	}
}

// Name returns the provider name.
func (p *LangGraphAIServer) Name() string {
	return "langgraph"
}

// GenerateSummary calls the Python AI service to generate a summary.
func (p *LangGraphAIServer) GenerateSummary(ctx context.Context, userID int64, content string) (<-chan string, error) {
	ch := make(chan string)

	go func() {
		defer close(ch)

		// Prepare request body
		reqBody := map[string]interface{}{
			"user_id":   userID,
			"content":   content,
			"streaming": true,
		}
		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			ch <- fmt.Sprintf("error: %v", err)
			return
		}

		// Create request
		req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/summary/generate", bytes.NewBuffer(jsonBody))
		if err != nil {
			ch <- fmt.Sprintf("error: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")

		// Send request
		resp, err := p.httpClient.Do(req)
		if err != nil {
			ch <- fmt.Sprintf("error: %v", err)
			return
		}
		defer resp.Body.Close()

		// Check status code
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			ch <- fmt.Sprintf("error: HTTP %d - %s", resp.StatusCode, string(body))
			return
		}

		// Parse SSE stream
		reader := bufio.NewReader(resp.Body)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					ch <- fmt.Sprintf("error: %v", err)
				}
				return
			}

			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			// Handle SSE events
			if strings.HasPrefix(line, "event:") {
				eventType := strings.TrimSpace(strings.TrimPrefix(line, "event:"))
				if eventType == "done" || eventType == "error" {
					// Read the data line and return
					return
				}
				continue
			}

			if strings.HasPrefix(line, "data:") {
				data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if data == "" {
					continue
				}

				// Parse JSON chunk
				var chunkData struct {
					Chunk   string `json:"chunk"`
					Success bool   `json:"success"`
					Error   string `json:"error"`
				}
				if err := json.Unmarshal([]byte(data), &chunkData); err != nil {
					// Not JSON, might be plain text
					ch <- data
					continue
				}

				if chunkData.Error != "" {
					ch <- fmt.Sprintf("error: %s", chunkData.Error)
					return
				}

				if chunkData.Chunk != "" {
					ch <- chunkData.Chunk
				}
			}
		}
	}()

	return ch, nil
}
