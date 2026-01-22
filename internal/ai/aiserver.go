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
	"sync"
	"time"
)

var (
	clientInstance AIProvider
	once           sync.Once
)

const (
	AISummaryPath = "/api/summary"
)

type AIProvider interface {
	GenerateSummary(ctx context.Context, userID int64, content string) (<-chan string, error)
}

type AIClient struct {
	baseURL    string
	httpClient *http.Client
}

func GetAIClient() AIProvider {
	once.Do(func() {
		// 默认初始化，实际配置可以从 config 包读取
		clientInstance = NewAIClient("http://localhost:5000")
	})
	return clientInstance
}
func NewAIClient(baseURL string) *AIClient {
	if baseURL == "" {
		baseURL = "http://localhost:5000"
	}
	return &AIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute, // Long timeout for streaming
		},
	}
}

type SummaryRequest struct {
	UserID    int    `json:"user_id"`
	Content   string `json:"content"`
	Streaming bool   `json:"streaming"`
}

type ChunkData struct {
	Chunk string `json:"chunk"`
}

func (c *AIClient) GenerateSummary(ctx context.Context, userID int64, content string) (<-chan string, error) {
	return c.HandleSSEResponse(ctx, userID, content, AISummaryPath), nil
}

func (c *AIClient) HandleSSEResponse(ctx context.Context, userID int64, content string, queryPath string) <-chan string {
	ch := make(chan string)
	go func() {
		defer close(ch)

		// Prepare request body
		reqBody := SummaryRequest{
			UserID:    int(userID),
			Content:   content,
			Streaming: true,
		}
		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			ch <- "event: error"
			return
		}

		// Create request
		req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+queryPath, bytes.NewBuffer(jsonBody))
		if err != nil {
			ch <- "event: error"
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")

		// Send request
		resp, err := c.httpClient.Do(req)
		if err != nil {
			ch <- "event: error"
			return
		}
		defer resp.Body.Close()

		// Check status code
		if resp.StatusCode != http.StatusOK {
			ch <- "event: error"
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
					ch <- fmt.Sprintf("event: %s", eventType)
					return
				}
				continue
			}

			if strings.HasPrefix(line, "data:") {
				data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if data == "" {
					continue
				}

				var chunkData ChunkData
				if err := json.Unmarshal([]byte(data), &chunkData); err != nil {
					ch <- "event: error"
					return
				}

				if chunkData.Chunk != "" {
					ch <- fmt.Sprintf(`data: {"chunk": "%s"}`, chunkData.Chunk)
				}
			}
		}
	}()
	return ch
}
