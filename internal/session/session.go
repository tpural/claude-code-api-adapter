// Package session manages on-disk session directories that provide
// a persistent working directory and session ID for the Claude Code CLI.
package session

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// Manager manages session directories under a base path.
type Manager struct {
	BaseDir string
}

// NewManager returns a Manager rooted at baseDir.
func NewManager(baseDir string) *Manager {
	return &Manager{BaseDir: baseDir}
}

func (m *Manager) EnsureBaseDir() error {
	return os.MkdirAll(m.BaseDir, 0755)
}

func (m *Manager) NewSession() (id string, dir string, err error) {
	id = uuid.New().String()
	dir = filepath.Join(m.BaseDir, id)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return "", "", fmt.Errorf("create session dir: %w", err)
	}
	return id, dir, nil
}

// ResolveSession returns the directory for an existing session, or an error
// if the session does not exist.
func (m *Manager) ResolveSession(sessionID string) (string, error) {
	dir := filepath.Join(m.BaseDir, sessionID)
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("session %q not found", sessionID)
		}
		return "", fmt.Errorf("stat session dir: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("session path %q is not a directory", dir)
	}
	return dir, nil
}

// GetOrCreateSession creates a new session when sessionID is empty, or
// resolves the existing one.
func (m *Manager) GetOrCreateSession(sessionID string) (sid string, dir string, isResume bool, err error) {
	if err = m.EnsureBaseDir(); err != nil {
		return "", "", false, err
	}

	if sessionID == "" {
		sid, dir, err = m.NewSession()
		return sid, dir, false, err
	}

	dir, err = m.ResolveSession(sessionID)
	if err != nil {
		return "", "", false, err
	}
	return sessionID, dir, true, nil
}
