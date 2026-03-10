package notify

// Notifier sends desktop notifications.
type Notifier interface {
	// Send displays a notification with the given title and body.
	Send(title, body string) error
}
