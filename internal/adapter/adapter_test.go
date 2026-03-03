package adapter

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/tpural/claude-code-api-adapter/internal/types"
)

func TestBuildPrompt_Basic(t *testing.T) {
	messages := []types.Message{
		{Role: "system", Content: "You are helpful."},
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
		{Role: "user", Content: "How are you?"},
	}

	prompt := BuildPrompt(messages, false)

	if !strings.Contains(prompt, "You are helpful.") {
		t.Error("expected system message in prompt")
	}
	if !strings.Contains(prompt, "User: Hello") {
		t.Error("expected user message in prompt")
	}
	if !strings.Contains(prompt, "Assistant: Hi there!") {
		t.Error("expected assistant message in prompt")
	}
	if !strings.Contains(prompt, "User: How are you?") {
		t.Error("expected second user message in prompt")
	}
}

func TestBuildPrompt_ResumeOnlyLastUser(t *testing.T) {
	messages := []types.Message{
		{Role: "user", Content: "First message"},
		{Role: "assistant", Content: "First response"},
		{Role: "user", Content: "Second message"},
	}

	prompt := BuildPrompt(messages, true)

	if prompt != "Second message" {
		t.Errorf("expected only last user message, got %q", prompt)
	}
}

func TestBuildPrompt_ResumeNoUser(t *testing.T) {
	messages := []types.Message{
		{Role: "assistant", Content: "Something"},
	}

	prompt := BuildPrompt(messages, true)
	if prompt != "Something" {
		t.Errorf("expected last message content, got %q", prompt)
	}
}

func TestResolveTools_Omitted(t *testing.T) {
	result, err := ResolveTools(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "default" {
		t.Errorf("expected 'default', got %q", result)
	}
}

func TestResolveTools_EmptyArray(t *testing.T) {
	result, err := ResolveTools([]types.Tool{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `""` {
		t.Errorf("expected empty string flag, got %q", result)
	}
}

func TestResolveTools_WithTools(t *testing.T) {
	tools := []types.Tool{
		{Type: "function", Function: types.ToolFunction{Name: "bash"}},
		{Type: "function", Function: types.ToolFunction{Name: "read_file"}},
		{Type: "function", Function: types.ToolFunction{Name: "grep"}},
	}

	result, err := ResolveTools(tools, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Bash") {
		t.Error("expected Bash in resolved tools")
	}
	if !strings.Contains(result, "Read") {
		t.Error("expected Read in resolved tools")
	}
	if !strings.Contains(result, "Grep") {
		t.Error("expected Grep in resolved tools")
	}
}

func TestResolveTools_DirectClaudeNames(t *testing.T) {
	tools := []types.Tool{
		{Type: "function", Function: types.ToolFunction{Name: "Bash"}},
		{Type: "function", Function: types.ToolFunction{Name: "Edit"}},
	}

	result, err := ResolveTools(tools, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Bash,Edit" {
		t.Errorf("expected 'Bash,Edit', got %q", result)
	}
}

func TestResolveTools_ToolChoiceNone(t *testing.T) {
	choice := json.RawMessage(`"none"`)
	result, err := ResolveTools(nil, choice)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `""` {
		t.Errorf("expected empty string flag, got %q", result)
	}
}

func TestResolveTools_ToolChoiceForced(t *testing.T) {
	choice := json.RawMessage(`{"type":"function","function":{"name":"bash"}}`)
	result, err := ResolveTools(nil, choice)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Bash" {
		t.Errorf("expected 'Bash', got %q", result)
	}
}

func TestResolveTools_Dedup(t *testing.T) {
	tools := []types.Tool{
		{Type: "function", Function: types.ToolFunction{Name: "read"}},
		{Type: "function", Function: types.ToolFunction{Name: "read_file"}},
		{Type: "function", Function: types.ToolFunction{Name: "Read"}},
	}

	result, err := ResolveTools(tools, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Read" {
		t.Errorf("expected single 'Read', got %q", result)
	}
}

func TestBuildCLIArgs_NonStreaming(t *testing.T) {
	req := &types.ChatCompletionRequest{
		Model: "sonnet",
		Messages: []types.Message{
			{Role: "user", Content: "Hello"},
		},
		Stream: false,
	}

	args := BuildCLIArgs(req, "default", "abc-123", false)

	assertContains(t, args, "-p")
	assertContains(t, args, "--model")
	assertContains(t, args, "sonnet")
	assertContains(t, args, "--dangerously-skip-permissions")
	assertContains(t, args, "--output-format")
	assertContains(t, args, "json")
	assertContains(t, args, "--session-id")
	assertContains(t, args, "abc-123")
	assertNotContains(t, args, "--resume")
	assertNotContains(t, args, "--no-session-persistence")
}

func TestBuildCLIArgs_Streaming(t *testing.T) {
	req := &types.ChatCompletionRequest{
		Model: "opus",
		Messages: []types.Message{
			{Role: "user", Content: "Hello"},
		},
		Stream: true,
	}

	args := BuildCLIArgs(req, "Bash,Read", "", false)

	assertContains(t, args, "stream-json")
	assertContains(t, args, "--verbose")
	assertContains(t, args, "--include-partial-messages")
	assertContains(t, args, "--no-session-persistence")
	assertNotContains(t, args, "--session-id")
}

func TestBuildCLIArgs_Resume(t *testing.T) {
	req := &types.ChatCompletionRequest{
		Model: "sonnet",
		Messages: []types.Message{
			{Role: "user", Content: "Continue"},
		},
		Stream: false,
	}

	args := BuildCLIArgs(req, "default", "existing-session", true)

	assertContains(t, args, "--resume")
	assertContains(t, args, "existing-session")
	assertNotContains(t, args, "--session-id")
}

func assertContains(t *testing.T, args []string, val string) {
	t.Helper()
	for _, a := range args {
		if a == val {
			return
		}
	}
	t.Errorf("expected args to contain %q, got %v", val, args)
}

func assertNotContains(t *testing.T, args []string, val string) {
	t.Helper()
	for _, a := range args {
		if a == val {
			t.Errorf("expected args NOT to contain %q, got %v", val, args)
			return
		}
	}
}
