// Package claude manages subprocess execution of the Claude Code CLI,
// supporting both synchronous JSON and streaming output modes.
package claude

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"

	"github.com/tpural/claude-code-api-adapter/internal/types"
)

// Executor abstracts Claude CLI subprocess execution so that handlers
// can be tested with mock implementations.
type Executor interface {
	// Execute runs the CLI synchronously and returns parsed JSON output.
	Execute(ctx context.Context, args []string, cwd string) (*types.ClaudeJSONOutput, error)
	// ExecuteStream runs the CLI in streaming mode, calling onEvent for
	// each line of newline-delimited JSON written to stdout.
	ExecuteStream(ctx context.Context, args []string, cwd string, onEvent func(eventJSON []byte) error) error
}

// RealExecutor implements Executor using the claude CLI binary.
type RealExecutor struct{}

func (e *RealExecutor) Execute(ctx context.Context, args []string, cwd string) (*types.ClaudeJSONOutput, error) {
	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Dir = cwd

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := stderr.String()
		if stderrStr != "" {
			return nil, fmt.Errorf("claude exited with error: %w\nstderr: %s", err, stderrStr)
		}
		return nil, fmt.Errorf("claude exited with error: %w", err)
	}

	var output types.ClaudeJSONOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		return nil, fmt.Errorf("parse claude output: %w\nraw: %s", err, stdout.String())
	}

	return &output, nil
}

func (e *RealExecutor) ExecuteStream(ctx context.Context, args []string, cwd string, onEvent func(eventJSON []byte) error) error {
	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Dir = cwd

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("create stdout pipe: %w", err)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start claude: %w", err)
	}

	scanner := bufio.NewScanner(stdoutPipe)
	scanner.Buffer(make([]byte, 0, 256*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		if err := onEvent(line); err != nil {
			_ = cmd.Process.Kill()
			return err
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return fmt.Errorf("read stdout: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		stderrStr := stderr.String()
		if stderrStr != "" {
			return fmt.Errorf("claude exited with error: %w\nstderr: %s", err, stderrStr)
		}
		return fmt.Errorf("claude exited with error: %w", err)
	}

	return nil
}

// ExtractTextDelta attempts to extract the text content from a stream-json event line.
// Returns the text and true if this is a text_delta event, empty string and false otherwise.
func ExtractTextDelta(eventJSON []byte) (string, bool) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(eventJSON, &raw); err != nil {
		return "", false
	}

	typeVal, ok := raw["type"]
	if !ok {
		return "", false
	}
	var eventType string
	if err := json.Unmarshal(typeVal, &eventType); err != nil {
		return "", false
	}

	if eventType == "content_block_delta" {
		return extractFromContentBlockDelta(raw)
	}

	if eventType == "stream_event" {
		return extractFromStreamEvent(raw)
	}

	return "", false
}

func extractFromContentBlockDelta(raw map[string]json.RawMessage) (string, bool) {
	deltaRaw, ok := raw["delta"]
	if !ok {
		return "", false
	}
	var delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(deltaRaw, &delta); err != nil {
		return "", false
	}
	if delta.Type == "text_delta" {
		return delta.Text, true
	}
	return "", false
}

// extractFromStreamEvent handles the "stream_event" wrapper shape where the
// actual delta is nested inside an "event" object.
func extractFromStreamEvent(raw map[string]json.RawMessage) (string, bool) {
	eventRaw, ok := raw["event"]
	if !ok {
		return "", false
	}

	var event map[string]json.RawMessage
	if err := json.Unmarshal(eventRaw, &event); err != nil {
		return "", false
	}

	deltaRaw, ok := event["delta"]
	if !ok {
		return "", false
	}

	var delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(deltaRaw, &delta); err != nil {
		return "", false
	}
	if delta.Type == "text_delta" {
		return delta.Text, true
	}
	return "", false
}
