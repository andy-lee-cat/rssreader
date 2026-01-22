// SPDX-FileCopyrightText: Copyright Andy. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ai_test // import "miniflux.app/v2/internal/ai"

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ai "miniflux.app/v2/internal/ai"
)

var (
	cutLen   = 13
	chunkLen = 5
)

func TestHandleSSEResponseAISummary(t *testing.T) {
	content := `abc"\nde'fghijk'"lmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ`
	queryPath := ai.AISummaryPath

	server := getMonkServer(t)
	defer server.Close()
	client := ai.NewAIClient(server.URL)

	t.Run("Got right SSE response", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		ch := client.HandleSSEResponse(ctx, 1, content, queryPath)

		loop := 0
		finish := false
		// 直接发送以 data: {"chunk": xxx} 的格式发送，最后发送 event: done，并关闭通道
		for msg := range ch {
			start := loop * chunkLen
			end := (loop + 1) * chunkLen
			if (loop+1)*chunkLen > cutLen {
				finish = true
				end = cutLen
			}
			chunk := content[start:end]
			want := fmt.Sprintf(`data: {"chunk": "%s"}`, chunk)
			assertEqual(t, want, msg)
			loop += 1
			if finish {
				assertSingleMsg(t, ch, `event: done`)
				return
			}
		}
	})

	t.Run("Got Internet error", func(t *testing.T) {
		failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer failServer.Close()

		client := ai.NewAIClient(failServer.URL)

		ctx := context.Background()
		ch := client.HandleSSEResponse(ctx, 1, content, queryPath)

		// 只接收到一个 event: error
		assertSingleMsg(t, ch, "event: error")
	})

	t.Run("Context cancelled by client", func(t *testing.T) {
		client := ai.NewAIClient(server.URL)
		// 设置一个极短的超时时间测试客户端主动关闭连接时的场景
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		ch := client.HandleSSEResponse(ctx, 1, content, queryPath)

		// 确保通道能最终关闭，而不是永久阻塞
		select {
		case _, ok := <-ch:
			if !ok {
				// 通道关闭，符合预期
			}
		case <-time.After(1 * time.Second):
			t.Error("Channel did not close after context cancellation")
		}
	})
}

func getMonkServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Helper()
		if r.URL.Path != ai.AISummaryPath {
			t.Errorf("Expected path %s, got %s", ai.AISummaryPath, r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		defer r.Body.Close()

		var req ai.SummaryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
			return
		}

		prefix := truncateToRunes(t, req.Content, cutLen)

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)

		// 模拟流式输出
		for i := 0; i < len(prefix); i += 5 {
			var chunk string
			if i+5 < len(prefix) {
				chunk = string(prefix[i : i+5])
			} else {
				chunk = string(prefix[i:])
			}
			sendSSEData(t, w, chunk)
		}
		sendSSEDone(t, w)
	}))
}

func truncateToRunes(t *testing.T, s string, n int) string {
	t.Helper()
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}

func sendSSEData(t *testing.T, w http.ResponseWriter, data string) {
	t.Helper()
	dataMap := map[string]string{"chunk": data}
	jsonData, err := json.Marshal(dataMap)
	if err != nil {
		t.Errorf("Failed to marshal JSON: %v", err)
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", jsonData)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func sendSSEDone(t *testing.T, w http.ResponseWriter) {
	t.Helper()
	fmt.Fprintf(w, "event: done\ndata: {}\n\n")
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func assertEqual(t *testing.T, want, got string) {
	t.Helper()
	if want != got {
		t.Errorf("Expected %s, but got %s", want, got)
	}
}

func assertChanClosed(t *testing.T, ch <-chan string) {
	t.Helper()
	select {
	case msg, ok := <-ch:
		if ok {
			t.Errorf("Expected channel to be closed, but got message: %s", msg)
		}
	case <-time.After(100 * time.Millisecond):
		t.Errorf("Expected channel to be closed")
	}
}

func assertSingleMsg(t *testing.T, ch <-chan string, want string) {
	t.Helper()

	// 验证第一个消息是期望信息
	select {
	case msg, ok := <-ch:
		if !ok {
			t.Errorf("Expected %s, got closed channel", want)
			return
		}
		if !strings.HasPrefix(msg, want) {
			t.Errorf("Expected %s, got %s", want, msg)
		}
	case <-time.After(100 * time.Millisecond):
		t.Errorf("Timeout: Expected %s, got nothing", want)
	}

	// 验证通道随后关闭（不应该有更多的消息）
	select {
	case msg, ok := <-ch:
		if ok {
			t.Errorf("Expected channel to be closed after %s, but got additional message: %s", want, msg)
		}
	case <-time.After(100 * time.Millisecond):
		t.Errorf("Expected channel to be closed after %s", want)
	}
}
