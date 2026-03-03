// Package types defines OpenAI-compatible request, response, and streaming
// structures used by claude-code-api-adapter.
package types

import "encoding/json"

// ChatCompletionRequest is an OpenAI-compatible chat completion request,
// extended with SessionID for session persistence.
type ChatCompletionRequest struct {
	Model      string          `json:"model"`
	Messages   []Message       `json:"messages"`
	Stream     bool            `json:"stream,omitempty"`
	Tools      []Tool          `json:"tools,omitempty"`
	ToolChoice json.RawMessage `json:"tool_choice,omitempty"`

	// Provider extension: maps to Claude Code's --session-id / --resume flags.
	SessionID string `json:"session_id,omitempty"`
}

// Message is a single message in the conversation history.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// Tool describes a tool available to the model.
type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function,omitempty"`
	Custom   *CustomTool  `json:"custom,omitempty"`
}

// ToolFunction describes a function-type tool definition.
type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

// CustomTool is a non-standard tool definition passed through to Claude Code.
type CustomTool struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// ToolChoiceFunction is an explicit tool selection: {"type":"function","function":{"name":"..."}}.
type ToolChoiceFunction struct {
	Type     string             `json:"type"`
	Function ToolChoiceFuncName `json:"function"`
}

// ToolChoiceFuncName is the function name within a ToolChoiceFunction.
type ToolChoiceFuncName struct {
	Name string `json:"name"`
}

// ChatCompletionResponse is a non-streaming chat completion response.
type ChatCompletionResponse struct {
	ID        string             `json:"id"`
	Object    string             `json:"object"`
	Created   int64              `json:"created"`
	Model     string             `json:"model"`
	Choices   []CompletionChoice `json:"choices"`
	Usage     Usage              `json:"usage"`
	SessionID string             `json:"session_id,omitempty"`
}

// CompletionChoice is a single choice within a completion response.
type CompletionChoice struct {
	Index        int      `json:"index"`
	Message      *Message `json:"message,omitempty"`
	FinishReason string   `json:"finish_reason"`
}

// Usage reports token consumption for a request.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatCompletionChunk is a single SSE frame in a streaming response.
type ChatCompletionChunk struct {
	ID        string        `json:"id"`
	Object    string        `json:"object"`
	Created   int64         `json:"created"`
	Model     string        `json:"model"`
	Choices   []ChunkChoice `json:"choices"`
	SessionID string        `json:"session_id,omitempty"`
}

// ChunkChoice is a single choice within a streaming chunk.
type ChunkChoice struct {
	Index        int        `json:"index"`
	Delta        ChunkDelta `json:"delta"`
	FinishReason *string    `json:"finish_reason"`
}

// ChunkDelta is the incremental content in a streaming chunk.
type ChunkDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// ClaudeJSONOutput is the parsed JSON from claude --output-format json.
type ClaudeJSONOutput struct {
	Result    string `json:"result"`
	SessionID string `json:"session_id,omitempty"`
	IsError   bool   `json:"is_error,omitempty"`
}

// ModelsResponse is the GET /v1/models response.
type ModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

// Model is a single entry in a ModelsResponse.
type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// ErrorResponse is the OpenAI error envelope.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail is the error payload inside an ErrorResponse.
type ErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}
