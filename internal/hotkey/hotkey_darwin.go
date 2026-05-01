//go:build darwin

package hotkey

import "fmt"

// darwinListener is a stub Listener for darwin builds.
// Real implementation will use Carbon RegisterEventHotKey via golang.design/x/hotkey
// (see plan Phase 2.4). Calling Register at runtime returns ErrNotImplemented.
type darwinListener struct {
	binding *Binding
	keydown chan struct{}
}

// NewListener returns a stub Listener that returns an error on Register.
// Phase 0 stub — replaced in Phase 2.4 with Carbon RegisterEventHotKey.
func NewListener(bindingStr string) (Listener, error) {
	binding, err := ParseBinding(bindingStr)
	if err != nil {
		return nil, err
	}
	return &darwinListener{
		binding: binding,
		keydown: make(chan struct{}, 1),
	}, nil
}

func (l *darwinListener) Register() error {
	return fmt.Errorf("hotkey: darwin implementation not yet wired (binding=%s)", l.binding)
}

func (l *darwinListener) Keydown() <-chan struct{} {
	return l.keydown
}

func (l *darwinListener) Unregister() error {
	return nil
}

// ReGrabAll is a no-op on darwin (no shared display connection to refresh).
func ReGrabAll() {}
