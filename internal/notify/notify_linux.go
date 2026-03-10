package notify

import (
	"fmt"
	"os/exec"
)

// linuxNotifier sends notifications via notify-send.
type linuxNotifier struct {
	appName string
	runner  func(name string, args ...string) error
}

// NewNotifier creates a Notifier that uses notify-send on Linux.
func NewNotifier() Notifier {
	return &linuxNotifier{
		appName: "Tomoe",
		runner:  defaultRunner,
	}
}

func defaultRunner(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}

func (n *linuxNotifier) Send(title, body string) error {
	err := n.runner("notify-send", "--app-name", n.appName, title, body)
	if err != nil {
		return fmt.Errorf("notify-send: %w", err)
	}
	return nil
}
