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

// Load reads a session from disk by ID.
func (s *Store) Load(id string) (*Session, error) {
	path := filepath.Join(s.baseDir, id, sessionFileName)
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
func (s *Store) Delete(id string) error {
	dir := filepath.Join(s.baseDir, id)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("deleting session: %w", err)
	}
	return nil
}
