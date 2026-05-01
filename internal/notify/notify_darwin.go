//go:build darwin

package notify

import "fmt"

// darwinNotifier is a stub Notifier for darwin builds.
// Phase 0 stub — replaced in Phase 2.3 with osascript shell-out.
type darwinNotifier struct{}

// NewNotifier returns the darwin Notifier.
func NewNotifier() Notifier {
	return &darwinNotifier{}
}

func (n *darwinNotifier) Send(title, body string) error {
	return fmt.Errorf("notify: darwin not yet wired (Phase 2.3: osascript) — title=%q body=%q", title, body)
}
