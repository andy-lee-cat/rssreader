// SPDX-FileCopyrightText: Copyright Andy. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"fmt"
	"net/http"
	"strings"

	"miniflux.app/v2/internal/ai"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/model"
)

// streamAISummary streams the AI summary for an entry.
// If the entry already has a summary in the database, it streams that.
// Otherwise, it generates a new summary using the AI provider.
func (h *handler) streamAISummary(w http.ResponseWriter, r *http.Request) {
	userID := request.UserID(r)
	entryID := request.RouteInt64Param(r, "entryID")

	// Get the entry
	builder := h.store.NewEntryQueryBuilder(userID)
	builder.WithEntryID(entryID)
	builder.WithoutStatus(model.EntryStatusRemoved)

	entry, err := builder.GetEntry()
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	if entry == nil {
		html.NotFound(w, r)
		return
	}

	// Set headers for SSE (Server-Sent Events)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		html.ServerError(w, r, fmt.Errorf("streaming not supported"))
		return
	}

	// If we already have a summary, stream it directly
	if entry.AISummary != "" {
		streamExistingSummary(w, flusher, entry.AISummary)
		return
	}

	// Generate new summary using AI provider
	aiServer := ai.GetAIServer()

	// Extract text content from HTML
	textContent := extractTextFromHTML(entry.Content)
	if textContent == "" {
		textContent = entry.Title
	}

	// Generate summary (pass userID for AI service to look up user's API key)
	ctx := r.Context()
	ch, err := aiServer.GenerateSummary(ctx, userID, textContent)
	if err != nil {
		sendSSEError(w, flusher, err.Error())
		return
	}

	var summaryBuilder strings.Builder
	for chunk := range ch {
		summaryBuilder.WriteString(chunk)
		sendSSEData(w, flusher, chunk)
	}

	// Save the complete summary to database
	fullSummary := summaryBuilder.String()
	if fullSummary != "" && !strings.HasPrefix(fullSummary, "error:") {
		if err := h.store.UpdateEntryAISummary(userID, entryID, fullSummary); err != nil {
			fmt.Printf("Failed to save AI summary: %v\n", err)
		}
	}

	sendDone(w, flusher)
}

// streamExistingSummary streams an existing summary character by character.
func streamExistingSummary(w http.ResponseWriter, flusher http.Flusher, summary string) {
	for _, r := range summary {
		sendSSEData(w, flusher, string(r))
	}
	sendDone(w, flusher)
}

// sendSSEData sends data via SSE.
func sendSSEData(w http.ResponseWriter, flusher http.Flusher, data string) {
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

// sendSSEError sends an error message via SSE.
func sendSSEError(w http.ResponseWriter, flusher http.Flusher, errMsg string) {
	fmt.Fprintf(w, "event: error\ndata: %s\n\n", errMsg)
	flusher.Flush()
}

// sendDone sends a done event via SSE.
func sendDone(w http.ResponseWriter, flusher http.Flusher) {
	fmt.Fprintf(w, "event: done\ndata: complete\n\n")
	flusher.Flush()
}

// extractTextFromHTML extracts plain text from HTML content.
// TODO(Andy): This is a simple implementation; you may want to use a proper HTML parser.
func extractTextFromHTML(htmlContent string) string {
	// Simple approach: remove HTML tags
	// For production, consider using golang.org/x/net/html or similar
	var result strings.Builder
	inTag := false

	for _, r := range htmlContent {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
			result.WriteRune(' ')
		case !inTag:
			result.WriteRune(r)
		}
	}

	// Clean up whitespace
	text := result.String()
	text = strings.Join(strings.Fields(text), " ")
	return strings.TrimSpace(text)
}
