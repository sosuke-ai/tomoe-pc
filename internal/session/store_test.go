package session

import (
	"testing"
	"time"
)

func TestStoreSaveAndLoad(t *testing.T) {
	store := NewStore(t.TempDir())

	sess := &Session{
		ID:        "sess-001",
		Title:     "Test Session",
		CreatedAt: time.Now().Truncate(time.Second),
		Duration:  120.0,
		Sources:   []string{"mic"},
		Segments: []Segment{
			{ID: "s1", Speaker: "You", Text: "Hello", StartTime: 0, EndTime: 1.5, Source: "mic"},
		},
	}

	if err := store.Save(sess); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := store.Load("sess-001")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if loaded.ID != sess.ID {
		t.Errorf("ID = %q, want %q", loaded.ID, sess.ID)
	}
	if loaded.Title != sess.Title {
		t.Errorf("Title = %q, want %q", loaded.Title, sess.Title)
	}
	if len(loaded.Segments) != 1 {
		t.Fatalf("Segments len = %d, want 1", len(loaded.Segments))
	}
	if loaded.Segments[0].Text != "Hello" {
		t.Errorf("Segments[0].Text = %q, want %q", loaded.Segments[0].Text, "Hello")
	}
}

func TestStoreList(t *testing.T) {
	store := NewStore(t.TempDir())

	now := time.Now().Truncate(time.Second)
	sess1 := &Session{ID: "older", Title: "Older", CreatedAt: now.Add(-1 * time.Hour)}
	sess2 := &Session{ID: "newer", Title: "Newer", CreatedAt: now}

	if err := store.Save(sess1); err != nil {
		t.Fatalf("Save(sess1) error: %v", err)
	}
	if err := store.Save(sess2); err != nil {
		t.Fatalf("Save(sess2) error: %v", err)
	}

	list, err := store.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	if len(list) != 2 {
		t.Fatalf("List() len = %d, want 2", len(list))
	}

	// Newest first
	if list[0].ID != "newer" {
		t.Errorf("List()[0].ID = %q, want %q", list[0].ID, "newer")
	}
	if list[1].ID != "older" {
		t.Errorf("List()[1].ID = %q, want %q", list[1].ID, "older")
	}
}

func TestStoreDelete(t *testing.T) {
	store := NewStore(t.TempDir())

	sess := &Session{
		ID:        "to-delete",
		Title:     "Delete Me",
		CreatedAt: time.Now().Truncate(time.Second),
	}

	if err := store.Save(sess); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	if err := store.Delete("to-delete"); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	_, err := store.Load("to-delete")
	if err == nil {
		t.Error("Load() should fail after Delete()")
	}
}

func TestStoreLoadNonexistent(t *testing.T) {
	store := NewStore(t.TempDir())

	_, err := store.Load("nonexistent")
	if err == nil {
		t.Error("Load() should fail for nonexistent session")
	}
}

func TestStoreListEmptyDir(t *testing.T) {
	store := NewStore(t.TempDir())

	list, err := store.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("List() len = %d, want 0", len(list))
	}
}

func TestStoreListNonexistentDir(t *testing.T) {
	store := NewStore("/nonexistent/path/sessions")

	list, err := store.List()
	if err != nil {
		t.Fatalf("List() should not error for nonexistent dir, got: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("List() len = %d, want 0", len(list))
	}
}

func TestStoreOverwrite(t *testing.T) {
	store := NewStore(t.TempDir())

	sess := &Session{
		ID:        "update-me",
		Title:     "Original",
		CreatedAt: time.Now().Truncate(time.Second),
	}
	if err := store.Save(sess); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	sess.Title = "Updated"
	if err := store.Save(sess); err != nil {
		t.Fatalf("Save() updated error: %v", err)
	}

	loaded, err := store.Load("update-me")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if loaded.Title != "Updated" {
		t.Errorf("Title = %q, want %q", loaded.Title, "Updated")
	}
}
