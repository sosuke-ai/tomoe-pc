package meeting

import "context"

// Platform identifies the meeting application.
type Platform string

const (
	PlatformTeams   Platform = "Teams"
	PlatformMeet    Platform = "Meet"
	PlatformZoom    Platform = "Zoom"
	PlatformWebex   Platform = "Webex"
	PlatformSlack   Platform = "Slack"
	PlatformUnknown Platform = "Unknown"
)

// EventType distinguishes start from stop events.
type EventType int

const (
	MeetingStarted EventType = iota
	MeetingStopped
)

// MeetingEvent represents a detected meeting start or stop.
type MeetingEvent struct {
	Type     EventType
	Platform Platform
}

// Detector monitors the system for active meetings — defined as a single
// process simultaneously using both microphone and speaker. Implementations
// are platform-specific: Linux uses libpulse stream introspection; macOS
// uses CoreAudio process objects (kAudioHardwarePropertyProcessObjectList).
type Detector interface {
	Events() <-chan MeetingEvent
	Start(ctx context.Context) error
	Stop()
}

// NewDetector returns the platform-appropriate Detector implementation.
func NewDetector() Detector {
	return newDetector()
}
