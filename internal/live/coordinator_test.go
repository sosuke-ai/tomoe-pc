package live

import (
	"testing"

	"github.com/sosuke-ai/tomoe-pc/internal/session"
	"github.com/sosuke-ai/tomoe-pc/internal/speaker"
)

func TestAssignSpeakerMic(t *testing.T) {
	c := &Coordinator{
		cfg: Config{},
	}

	label := c.assignSpeaker(SourceMic, []float32{0.1, 0.2})
	if label != "You" {
		t.Errorf("mic speaker = %q, want %q", label, "You")
	}
}

func TestAssignSpeakerMonitorNoEmbedder(t *testing.T) {
	c := &Coordinator{
		cfg: Config{},
	}

	label := c.assignSpeaker(SourceMonitor, []float32{0.1, 0.2})
	if label != "Other" {
		t.Errorf("monitor speaker without embedder = %q, want %q", label, "Other")
	}
}

func TestNextSegID(t *testing.T) {
	c := &Coordinator{}

	id1 := c.nextSegID()
	id2 := c.nextSegID()

	if id1 != "seg-1" {
		t.Errorf("first ID = %q, want %q", id1, "seg-1")
	}
	if id2 != "seg-2" {
		t.Errorf("second ID = %q, want %q", id2, "seg-2")
	}
}

func TestSourceTypeConstants(t *testing.T) {
	if SourceMic != "mic" {
		t.Errorf("SourceMic = %q, want %q", SourceMic, "mic")
	}
	if SourceMonitor != "monitor" {
		t.Errorf("SourceMonitor = %q, want %q", SourceMonitor, "monitor")
	}
}

func TestNewCoordinator(t *testing.T) {
	c := New(Config{SegmentBufferSize: 32})
	if c == nil {
		t.Fatal("New() returned nil")
	}
	if cap(c.segmentCh) != 32 {
		t.Errorf("segment channel cap = %d, want 32", cap(c.segmentCh))
	}
}

func TestNewCoordinatorDefaultBufferSize(t *testing.T) {
	c := New(Config{})
	if cap(c.segmentCh) != 64 {
		t.Errorf("default segment channel cap = %d, want 64", cap(c.segmentCh))
	}
}

func TestSegmentStructure(t *testing.T) {
	seg := session.Segment{
		ID:        "seg-1",
		Speaker:   "You",
		Text:      "Hello world",
		StartTime: 1.0,
		EndTime:   2.5,
		Source:    "mic",
	}

	if seg.Speaker != "You" {
		t.Errorf("Speaker = %q, want %q", seg.Speaker, "You")
	}
}

func TestTrackerIntegration(t *testing.T) {
	tracker := speaker.NewTracker(0.7)

	// Simulate what the coordinator does with speaker tracking
	emb1 := []float32{1, 0, 0, 0}
	emb2 := []float32{0, 1, 0, 0}

	label1 := tracker.Assign(emb1)
	label2 := tracker.Assign(emb2)
	label3 := tracker.Assign(emb1) // Same as emb1

	if label1 != "Person 1" {
		t.Errorf("label1 = %q, want %q", label1, "Person 1")
	}
	if label2 != "Person 2" {
		t.Errorf("label2 = %q, want %q", label2, "Person 2")
	}
	if label3 != "Person 1" {
		t.Errorf("label3 = %q, want %q (same as emb1)", label3, "Person 1")
	}
}
