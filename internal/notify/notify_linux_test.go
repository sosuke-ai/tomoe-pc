package notify

import (
	"strings"
	"testing"
)

func TestLinuxNotifierCommandArgs(t *testing.T) {
	var capturedName string
	var capturedArgs []string

	n := &linuxNotifier{
		appName: "Tomoe",
		runner: func(name string, args ...string) error {
			capturedName = name
			capturedArgs = args
			return nil
		},
	}

	if err := n.Send("Test Title", "Test body text"); err != nil {
		t.Fatalf("Send() error: %v", err)
	}

	if capturedName != "notify-send" {
		t.Errorf("command = %q, want %q", capturedName, "notify-send")
	}

	want := []string{"--app-name", "Tomoe", "Test Title", "Test body text"}
	if len(capturedArgs) != len(want) {
		t.Fatalf("args = %v, want %v", capturedArgs, want)
	}
	for i, arg := range capturedArgs {
		if arg != want[i] {
			t.Errorf("args[%d] = %q, want %q", i, arg, want[i])
		}
	}
}

func TestLinuxNotifierError(t *testing.T) {
	n := &linuxNotifier{
		appName: "Tomoe",
		runner: func(name string, args ...string) error {
			return &fakeError{msg: "command not found"}
		},
	}

	err := n.Send("Title", "Body")
	if err == nil {
		t.Fatal("Send() should return error when runner fails")
	}
	if !strings.Contains(err.Error(), "notify-send") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "notify-send")
	}
}

type fakeError struct {
	msg string
}

func (e *fakeError) Error() string {
	return e.msg
}
