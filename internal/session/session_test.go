package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManager_NewSession(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)

	id, sessionDir, err := m.NewSession()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty session ID")
	}

	expectedDir := filepath.Join(dir, id)
	if sessionDir != expectedDir {
		t.Errorf("expected dir %q, got %q", expectedDir, sessionDir)
	}

	info, err := os.Stat(sessionDir)
	if err != nil {
		t.Fatalf("session dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("session path is not a directory")
	}
}

func TestManager_ResolveSession_Exists(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)

	sessionDir := filepath.Join(dir, "test-session")
	os.MkdirAll(sessionDir, 0755)

	resolved, err := m.ResolveSession("test-session")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != sessionDir {
		t.Errorf("expected %q, got %q", sessionDir, resolved)
	}
}

func TestManager_ResolveSession_NotFound(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)

	_, err := m.ResolveSession("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}

func TestManager_GetOrCreateSession_New(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)

	id, sessionDir, isResume, err := m.GetOrCreateSession("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty session ID")
	}
	if isResume {
		t.Error("expected isResume=false for new session")
	}

	info, err := os.Stat(sessionDir)
	if err != nil {
		t.Fatalf("session dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("session path is not a directory")
	}
}

func TestManager_GetOrCreateSession_Resume(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)

	sessionDir := filepath.Join(dir, "existing-id")
	os.MkdirAll(sessionDir, 0755)

	id, resolved, isResume, err := m.GetOrCreateSession("existing-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "existing-id" {
		t.Errorf("expected id 'existing-id', got %q", id)
	}
	if resolved != sessionDir {
		t.Errorf("expected dir %q, got %q", sessionDir, resolved)
	}
	if !isResume {
		t.Error("expected isResume=true for existing session")
	}
}

func TestManager_GetOrCreateSession_ResumeNotFound(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)

	_, _, _, err := m.GetOrCreateSession("does-not-exist")
	if err == nil {
		t.Error("expected error for nonexistent session resume")
	}
}
