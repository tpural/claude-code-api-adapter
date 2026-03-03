// Package main is the entry point for claude-code-api-adapter, an
// OpenAI-compatible API server backed by the Claude Code CLI.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/tpural/claude-code-api-adapter/internal/claude"
	"github.com/tpural/claude-code-api-adapter/internal/handler"
	"github.com/tpural/claude-code-api-adapter/internal/session"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	sessionsDir := os.Getenv("SESSIONS_DIR")
	if sessionsDir == "" {
		sessionsDir = "./sessions"
	}

	sm := session.NewManager(sessionsDir)
	executor := &claude.RealExecutor{}
	h := handler.New(executor, sm)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/chat/completions", h.ChatCompletions)
	mux.HandleFunc("GET /v1/models", h.ListModels)

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	log.Printf("claude-code-api-adapter listening on :%s", port)
	return fmt.Errorf("server exited: %w", http.ListenAndServe(":"+port, mux))
}
