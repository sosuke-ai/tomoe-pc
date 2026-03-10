package session

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSessionJSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	sess := &Session{
		ID:        "test-123",
		Title:     "Team Standup",
		CreatedAt: now,
		EndedAt:   now.Add(30 * time.Minute),
		Duration:  1800.0,
		Sources:   []string{"mic", "monitor"},
		Segments: []Segment{
			{
				ID:        "seg-1",
				Speaker:   "You",
				Text:      "Good morning everyone.",
				StartTime: 0.0,
				EndTime:   2.5,
				Source:    "mic",
			},
			{
				ID:        "seg-2",
				Speaker:   "Person 1",
				Text:      "Hey, good morning.",
				StartTime: 3.0,
				EndTime:   4.5,
				Source:    "monitor",
			},
		},
		AudioPath: "/tmp/test/audio.mp3",
	}

	data, err := json.Marshal(sess)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var loaded Session
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if loaded.ID != sess.ID {
		t.Errorf("ID = %q, want %q", loaded.ID, sess.ID)
	}
	if loaded.Title != sess.Title {
		t.Errorf("Title = %q, want %q", loaded.Title, sess.Title)
	}
	if !loaded.CreatedAt.Equal(sess.CreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", loaded.CreatedAt, sess.CreatedAt)
	}
	if loaded.Duration != sess.Duration {
		t.Errorf("Duration = %v, want %v", loaded.Duration, sess.Duration)
	}
	if len(loaded.Sources) != len(sess.Sources) {
		t.Fatalf("Sources len = %d, want %d", len(loaded.Sources), len(sess.Sources))
	}
	if len(loaded.Segments) != 2 {
		t.Fatalf("Segments len = %d, want 2", len(loaded.Segments))
	}
	if loaded.Segments[0].Speaker != "You" {
		t.Errorf("Segments[0].Speaker = %q, want %q", loaded.Segments[0].Speaker, "You")
	}
	if loaded.Segments[1].Speaker != "Person 1" {
		t.Errorf("Segments[1].Speaker = %q, want %q", loaded.Segments[1].Speaker, "Person 1")
	}
	if loaded.AudioPath != sess.AudioPath {
		t.Errorf("AudioPath = %q, want %q", loaded.AudioPath, sess.AudioPath)
	}
}

func TestSessionEmptySegments(t *testing.T) {
	sess := &Session{
		ID:        "empty-1",
		Title:     "Empty Session",
		CreatedAt: time.Now().Truncate(time.Second),
	}

	data, err := json.Marshal(sess)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var loaded Session
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if loaded.Segments != nil {
		t.Errorf("Segments should be nil for empty session, got %v", loaded.Segments)
	}
}

func TestSegmentJSONFields(t *testing.T) {
	seg := Segment{
		ID:        "s1",
		Speaker:   "You",
		Text:      "Hello world",
		StartTime: 1.5,
		EndTime:   3.75,
		Source:    "mic",
	}

	data, err := json.Marshal(seg)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Verify JSON field names
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal to map error: %v", err)
	}

	expectedKeys := []string{"id", "speaker", "text", "start_time", "end_time", "source"}
	for _, key := range expectedKeys {
		if _, ok := m[key]; !ok {
			t.Errorf("missing JSON key %q", key)
		}
	}
}
