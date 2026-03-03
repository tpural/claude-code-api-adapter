// Package handler implements the HTTP request handlers for
// claude-code-api-adapter.
package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/tpural/claude-code-api-adapter/internal/adapter"
	"github.com/tpural/claude-code-api-adapter/internal/claude"
	"github.com/tpural/claude-code-api-adapter/internal/session"
	"github.com/tpural/claude-code-api-adapter/internal/types"
)

// Handler serves the OpenAI-compatible API endpoints.
type Handler struct {
	executor claude.Executor
	sessions *session.Manager
}

// New returns a Handler backed by the given executor and session manager.
func New(executor claude.Executor, sessions *session.Manager) *Handler {
	return &Handler{executor: executor, sessions: sessions}
}

// ChatCompletions handles POST /v1/chat/completions, supporting both
// streaming and non-streaming responses.
func (h *Handler) ChatCompletions(w http.ResponseWriter, r *http.Request) {
	var req types.ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if req.Model == "" {
		writeError(w, http.StatusBadRequest, "model is required")
		return
	}
	if len(req.Messages) == 0 {
		writeError(w, http.StatusBadRequest, "messages is required and must not be empty")
		return
	}

	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = r.Header.Get("X-Session-Id")
	}

	sid, sessionDir, isResume, err := h.sessions.GetOrCreateSession(sessionID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "session error: "+err.Error())
		return
	}

	toolsFlag, err := adapter.ResolveTools(req.Tools, req.ToolChoice)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tools/tool_choice: "+err.Error())
		return
	}

	args := adapter.BuildCLIArgs(&req, toolsFlag, sid, isResume)

	log.Printf("request model=%s stream=%v session=%s resume=%v tools=%s",
		req.Model, req.Stream, sid, isResume, toolsFlag)

	if req.Stream {
		h.handleStreaming(w, r, args, sessionDir, sid, req.Model)
	} else {
		h.handleNonStreaming(w, r, args, sessionDir, sid, req.Model)
	}
}

func (h *Handler) handleNonStreaming(w http.ResponseWriter, r *http.Request, args []string, cwd, sessionID, model string) {
	output, err := h.executor.Execute(r.Context(), args, cwd)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "claude execution failed: "+err.Error())
		return
	}

	if output.IsError {
		writeError(w, http.StatusInternalServerError, "claude returned error: "+output.Result)
		return
	}

	resp := types.ChatCompletionResponse{
		ID:      "chatcmpl-" + uuid.New().String(),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []types.CompletionChoice{
			{
				Index: 0,
				Message: &types.Message{
					Role:    "assistant",
					Content: output.Result,
				},
				FinishReason: "stop",
			},
		},
		Usage:     types.Usage{},
		SessionID: sessionID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Session-Id", sessionID)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) handleStreaming(w http.ResponseWriter, r *http.Request, args []string, cwd, sessionID, model string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Session-Id", sessionID)
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	completionID := "chatcmpl-" + uuid.New().String()
	created := time.Now().Unix()

	initialChunk := types.ChatCompletionChunk{
		ID:      completionID,
		Object:  "chat.completion.chunk",
		Created: created,
		Model:   model,
		Choices: []types.ChunkChoice{
			{
				Index: 0,
				Delta: types.ChunkDelta{Role: "assistant"},
			},
		},
		SessionID: sessionID,
	}
	writeSSEChunk(w, flusher, initialChunk)

	err := h.executor.ExecuteStream(r.Context(), args, cwd, func(eventJSON []byte) error {
		text, ok := claude.ExtractTextDelta(eventJSON)
		if !ok || text == "" {
			return nil
		}

		chunk := types.ChatCompletionChunk{
			ID:      completionID,
			Object:  "chat.completion.chunk",
			Created: created,
			Model:   model,
			Choices: []types.ChunkChoice{
				{
					Index: 0,
					Delta: types.ChunkDelta{Content: text},
				},
			},
			SessionID: sessionID,
		}
		return writeSSEChunk(w, flusher, chunk)
	})

	if err != nil {
		log.Printf("stream error: %v", err)
	}

	stopReason := "stop"
	finalChunk := types.ChatCompletionChunk{
		ID:      completionID,
		Object:  "chat.completion.chunk",
		Created: created,
		Model:   model,
		Choices: []types.ChunkChoice{
			{
				Index:        0,
				Delta:        types.ChunkDelta{},
				FinishReason: &stopReason,
			},
		},
		SessionID: sessionID,
	}
	writeSSEChunk(w, flusher, finalChunk)

	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}

func writeSSEChunk(w http.ResponseWriter, flusher http.Flusher, chunk types.ChatCompletionChunk) error {
	data, err := json.Marshal(chunk)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", data)
	if err != nil {
		return err
	}
	flusher.Flush()
	return nil
}

// ListModels handles GET /v1/models, returning the set of Claude model
// identifiers in OpenAI-compatible format.
func (h *Handler) ListModels(w http.ResponseWriter, r *http.Request) {
	models := types.ModelsResponse{
		Object: "list",
		Data: []types.Model{
			{ID: "claude-sonnet-4-20250514", Object: "model", Created: 1700000000, OwnedBy: "anthropic"},
			{ID: "claude-opus-4-20250514", Object: "model", Created: 1700000000, OwnedBy: "anthropic"},
			{ID: "claude-haiku-3-5-20241022", Object: "model", Created: 1700000000, OwnedBy: "anthropic"},
			{ID: "sonnet", Object: "model", Created: 1700000000, OwnedBy: "anthropic"},
			{ID: "opus", Object: "model", Created: 1700000000, OwnedBy: "anthropic"},
			{ID: "haiku", Object: "model", Created: 1700000000, OwnedBy: "anthropic"},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(types.ErrorResponse{
		Error: types.ErrorDetail{
			Message: message,
			Type:    "invalid_request_error",
		},
	})
}
