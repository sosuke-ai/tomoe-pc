package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const sessionFileName = "session.json"

// Store manages session persistence on disk.
// Each session is stored in its own directory: <baseDir>/<session-id>/session.json
type Store struct {
	baseDir string
}

// NewStore creates a Store rooted at the given directory.
func NewStore(baseDir string) *Store {
	return &Store{baseDir: baseDir}
}

// Save persists a session to disk.
func (s *Store) Save(sess *Session) error {
	dir := filepath.Join(s.baseDir, sess.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating session directory: %w", err)
	}

	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling session: %w", err)
	}

	path := filepath.Join(dir, sessionFileName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing session file: %w", err)
	}

	return nil
}

// Load reads a session from disk by ID. Supports unique prefix matching
// (e.g., "9237d6bd" matches "9237d6bd-755b-4ef7-948c-3ae5a5c61597").
func (s *Store) Load(id string) (*Session, error) {
	resolvedID, err := s.resolveID(id)
	if err != nil {
		return nil, err
	}

	path := filepath.Join(s.baseDir, resolvedID, sessionFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading session file: %w", err)
	}

	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, fmt.Errorf("parsing session file: %w", err)
	}

	return &sess, nil
}

// resolveID resolves a full or prefix session ID to the actual directory name.
func (s *Store) resolveID(id string) (string, error) {
	// Try exact match first
	exact := filepath.Join(s.baseDir, id, sessionFileName)
	if _, err := os.Stat(exact); err == nil {
		return id, nil
	}

	// Try prefix match
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return "", fmt.Errorf("reading session directory: %w", err)
	}

	var matches []string
	for _, entry := range entries {
		if entry.IsDir() && len(entry.Name()) >= len(id) && entry.Name()[:len(id)] == id {
			matches = append(matches, entry.Name())
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no session found matching %q", id)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("ambiguous session ID %q matches %d sessions", id, len(matches))
	}
}

// List returns all stored sessions, sorted by creation time (newest first).
func (s *Store) List() ([]*Session, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading session directory: %w", err)
	}

	var sessions []*Session
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		sess, err := s.Load(entry.Name())
		if err != nil {
			continue // skip corrupt sessions
		}
		sessions = append(sessions, sess)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].CreatedAt.After(sessions[j].CreatedAt)
	})

	return sessions, nil
}

// Delete removes a session and its directory from disk.
// Supports prefix matching like Load.
func (s *Store) Delete(id string) error {
	resolvedID, err := s.resolveID(id)
	if err != nil {
		return err
	}
	dir := filepath.Join(s.baseDir, resolvedID)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("deleting session: %w", err)
	}
	return nil
}
