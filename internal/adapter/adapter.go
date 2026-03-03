// Package adapter translates OpenAI API requests into Claude Code CLI arguments.
package adapter

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tpural/claude-code-api-adapter/internal/types"
)

// toolMap maps OpenAI-style function names to Claude Code tool names.
var toolMap = map[string]string{
	"bash":       "Bash",
	"shell":      "Bash",
	"terminal":   "Bash",
	"read":       "Read",
	"read_file":  "Read",
	"edit":       "Edit",
	"edit_file":  "Edit",
	"write":      "Write",
	"write_file": "Write",
	"grep":       "Grep",
	"search":     "Grep",
	"glob":       "Glob",
	"find_files": "Glob",
	"ls":         "LS",
	"list":       "LS",
	"web_search": "WebSearch",
	"web_fetch":  "WebFetch",
	"notebook":   "NotebookEdit",
}

// knownClaudeTools is the set of canonical Claude Code tool names.
var knownClaudeTools = map[string]bool{
	"Bash":         true,
	"Read":         true,
	"Edit":         true,
	"Write":        true,
	"Grep":         true,
	"Glob":         true,
	"LS":           true,
	"WebSearch":    true,
	"WebFetch":     true,
	"NotebookEdit": true,
	"TodoRead":     true,
	"TodoWrite":    true,
}

// resolveToolName maps an OpenAI-style or user-supplied tool name to
// the canonical Claude Code tool name. Unrecognized names are passed through.
func resolveToolName(name string) string {
	if mapped, ok := toolMap[strings.ToLower(name)]; ok {
		return mapped
	}
	for known := range knownClaudeTools {
		if strings.EqualFold(name, known) {
			return known
		}
	}
	return name
}

// ResolveTools converts OpenAI tools + tool_choice into a --tools value for the CLI.
// Returns the string to pass as --tools, or "default" when tools should be unrestricted.
func ResolveTools(tools []types.Tool, toolChoiceRaw json.RawMessage) (string, error) {
	toolChoiceStr, toolChoiceObj, err := parseToolChoice(toolChoiceRaw)
	if err != nil {
		return "", err
	}

	if toolChoiceStr == "none" {
		return `""`, nil
	}

	if toolChoiceObj != nil {
		resolved := resolveToolName(toolChoiceObj.Function.Name)
		return resolved, nil
	}

	if tools == nil {
		return "default", nil
	}

	if len(tools) == 0 {
		return `""`, nil
	}

	seen := make(map[string]bool)
	var resolved []string
	for _, t := range tools {
		name := ""
		if t.Type == "function" {
			name = t.Function.Name
		} else if t.Custom != nil {
			name = t.Custom.Name
		}
		if name == "" {
			continue
		}
		mapped := resolveToolName(name)
		if !seen[mapped] {
			seen[mapped] = true
			resolved = append(resolved, mapped)
		}
	}

	if len(resolved) == 0 {
		return "default", nil
	}
	return strings.Join(resolved, ","), nil
}

// parseToolChoice decodes the raw tool_choice JSON into either a string
// keyword ("auto", "none") or a structured ToolChoiceFunction object.
func parseToolChoice(raw json.RawMessage) (string, *types.ToolChoiceFunction, error) {
	if len(raw) == 0 {
		return "", nil, nil
	}

	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, nil, nil
	}

	var obj types.ToolChoiceFunction
	if err := json.Unmarshal(raw, &obj); err == nil && obj.Type != "" {
		return "", &obj, nil
	}

	return "", nil, fmt.Errorf("invalid tool_choice: %s", string(raw))
}

// BuildPrompt serializes OpenAI messages into a text prompt for claude -p.
// If isResume is true, only the last user message is included.
func BuildPrompt(messages []types.Message, isResume bool) string {
	if isResume {
		for i := len(messages) - 1; i >= 0; i-- {
			if messages[i].Role == "user" {
				return messages[i].Content
			}
		}
		if len(messages) > 0 {
			return messages[len(messages)-1].Content
		}
		return ""
	}

	var parts []string
	for _, m := range messages {
		switch m.Role {
		case "system", "developer":
			parts = append(parts, m.Content)
		case "user":
			parts = append(parts, "User: "+m.Content)
		case "assistant":
			parts = append(parts, "Assistant: "+m.Content)
		default:
			parts = append(parts, m.Role+": "+m.Content)
		}
	}
	return strings.Join(parts, "\n\n")
}

// BuildCLIArgs constructs the full argument list for the claude subprocess.
func BuildCLIArgs(req *types.ChatCompletionRequest, toolsFlag string, sessionID string, isResume bool) []string {
	prompt := BuildPrompt(req.Messages, isResume)

	args := []string{
		"-p", prompt,
		"--model", req.Model,
		"--dangerously-skip-permissions",
	}

	if toolsFlag != "" {
		args = append(args, "--tools", toolsFlag)
	}

	if req.Stream {
		args = append(args, "--output-format", "stream-json", "--verbose", "--include-partial-messages")
	} else {
		args = append(args, "--output-format", "json")
	}

	if isResume && sessionID != "" {
		args = append(args, "--resume", sessionID)
	} else if sessionID != "" {
		args = append(args, "--session-id", sessionID)
	} else {
		args = append(args, "--no-session-persistence")
	}

	return args
}
