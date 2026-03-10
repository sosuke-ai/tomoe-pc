package backend

import (
	"testing"

	"github.com/sosuke-ai/tomoe-pc/internal/session"
)

func TestNewApp(t *testing.T) {
	app := NewApp()
	if app == nil {
		t.Fatal("NewApp() returned nil")
	}
}

func TestIsRecordingDefault(t *testing.T) {
	app := NewApp()
	if app.IsRecording() {
		t.Error("IsRecording() should be false initially")
	}
}

func TestExportSessionUnknownFormat(t *testing.T) {
	app := NewApp()
	app.store = session.NewStore(t.TempDir())
	_, err := app.ExportSession("nonexistent", "unknown-format")
	if err == nil {
		t.Error("ExportSession() should fail for nonexistent session")
	}
}

func TestBytesWriter(t *testing.T) {
	var buf []byte
	w := &bytesWriter{buf: &buf}

	n, err := w.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	if n != 5 {
		t.Errorf("Write returned %d, want 5", n)
	}

	_, err = w.Write([]byte(" world"))
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	if string(buf) != "hello world" {
		t.Errorf("buf = %q, want %q", string(buf), "hello world")
	}
}
