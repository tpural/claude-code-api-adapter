package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tpural/claude-code-api-adapter/internal/session"
	"github.com/tpural/claude-code-api-adapter/internal/types"
)

type mockExecutor struct {
	Result      string
	IsError     bool
	StreamLines []string
	ExecError   error
}

func (m *mockExecutor) Execute(ctx context.Context, args []string, cwd string) (*types.ClaudeJSONOutput, error) {
	if m.ExecError != nil {
		return nil, m.ExecError
	}
	return &types.ClaudeJSONOutput{
		Result:  m.Result,
		IsError: m.IsError,
	}, nil
}

func (m *mockExecutor) ExecuteStream(ctx context.Context, args []string, cwd string, onEvent func(eventJSON []byte) error) error {
	if m.ExecError != nil {
		return m.ExecError
	}
	for _, line := range m.StreamLines {
		if err := onEvent([]byte(line)); err != nil {
			return err
		}
	}
	return nil
}

func TestChatCompletions_NonStreaming(t *testing.T) {
	sm := session.NewManager(t.TempDir())
	executor := &mockExecutor{Result: "Hello from Claude!"}
	h := New(executor, sm)

	body := `{"model":"sonnet","messages":[{"role":"user","content":"Hi"}]}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ChatCompletions(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result types.ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Object != "chat.completion" {
		t.Errorf("expected object 'chat.completion', got %q", result.Object)
	}
	if len(result.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(result.Choices))
	}
	if result.Choices[0].Message.Content != "Hello from Claude!" {
		t.Errorf("expected content 'Hello from Claude!', got %q", result.Choices[0].Message.Content)
	}
	if result.Choices[0].FinishReason != "stop" {
		t.Errorf("expected finish_reason 'stop', got %q", result.Choices[0].FinishReason)
	}
	if result.SessionID == "" {
		t.Error("expected non-empty session_id in response")
	}
	if result.Model != "sonnet" {
		t.Errorf("expected model 'sonnet', got %q", result.Model)
	}

	sessionHeader := resp.Header.Get("X-Session-Id")
	if sessionHeader == "" {
		t.Error("expected X-Session-Id header")
	}
	if sessionHeader != result.SessionID {
		t.Errorf("header session_id %q != body session_id %q", sessionHeader, result.SessionID)
	}
}

func TestChatCompletions_Streaming(t *testing.T) {
	sm := session.NewManager(t.TempDir())
	executor := &mockExecutor{
		StreamLines: []string{
			`{"type":"content_block_delta","delta":{"type":"text_delta","text":"Hello"}}`,
			`{"type":"content_block_delta","delta":{"type":"text_delta","text":" world"}}`,
		},
	}
	h := New(executor, sm)

	body := `{"model":"sonnet","messages":[{"role":"user","content":"Hi"}],"stream":true}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ChatCompletions(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("expected text/event-stream, got %q", ct)
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyStr := string(bodyBytes)

	if !strings.Contains(bodyStr, `"content":"Hello"`) {
		t.Error("expected Hello chunk in SSE output")
	}
	if !strings.Contains(bodyStr, `"content":" world"`) {
		t.Error("expected world chunk in SSE output")
	}
	if !strings.Contains(bodyStr, "data: [DONE]") {
		t.Error("expected [DONE] at end of stream")
	}
	if !strings.Contains(bodyStr, `"finish_reason":"stop"`) {
		t.Error("expected final chunk with finish_reason stop")
	}
}

func TestChatCompletions_MissingModel(t *testing.T) {
	sm := session.NewManager(t.TempDir())
	executor := &mockExecutor{}
	h := New(executor, sm)

	body := `{"messages":[{"role":"user","content":"Hi"}]}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.ChatCompletions(w, req)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Result().StatusCode)
	}
}

func TestChatCompletions_MissingMessages(t *testing.T) {
	sm := session.NewManager(t.TempDir())
	executor := &mockExecutor{}
	h := New(executor, sm)

	body := `{"model":"sonnet"}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.ChatCompletions(w, req)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Result().StatusCode)
	}
}

func TestChatCompletions_SessionIDFromHeader(t *testing.T) {
	sm := session.NewManager(t.TempDir())

	id, _, err := sm.NewSession()
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	executor := &mockExecutor{Result: "Continued"}
	h := New(executor, sm)

	body := `{"model":"sonnet","messages":[{"role":"user","content":"Continue"}]}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("X-Session-Id", id)
	w := httptest.NewRecorder()

	h.ChatCompletions(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result types.ChatCompletionResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.SessionID != id {
		t.Errorf("expected session_id %q, got %q", id, result.SessionID)
	}
}

func TestChatCompletions_ClaudeError(t *testing.T) {
	sm := session.NewManager(t.TempDir())
	executor := &mockExecutor{Result: "Something went wrong", IsError: true}
	h := New(executor, sm)

	body := `{"model":"sonnet","messages":[{"role":"user","content":"Hi"}]}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.ChatCompletions(w, req)

	if w.Result().StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Result().StatusCode)
	}
}

func TestListModels(t *testing.T) {
	sm := session.NewManager(t.TempDir())
	executor := &mockExecutor{}
	h := New(executor, sm)

	req := httptest.NewRequest("GET", "/v1/models", nil)
	w := httptest.NewRecorder()

	h.ListModels(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result types.ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Object != "list" {
		t.Errorf("expected object 'list', got %q", result.Object)
	}
	if len(result.Data) == 0 {
		t.Error("expected at least one model")
	}

	found := false
	for _, m := range result.Data {
		if m.ID == "sonnet" {
			found = true
		}
		if m.Object != "model" {
			t.Errorf("expected model object 'model', got %q", m.Object)
		}
		if m.OwnedBy != "anthropic" {
			t.Errorf("expected owned_by 'anthropic', got %q", m.OwnedBy)
		}
	}
	if !found {
		t.Error("expected 'sonnet' alias in models list")
	}
}
