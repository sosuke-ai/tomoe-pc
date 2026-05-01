//go:build darwin

package meeting

import (
	"context"
	"fmt"
)

// darwinDetector is a stub Detector for darwin builds.
// Phase 0 stub — replaced in Phase 2.7 with CoreAudio process enumeration
// (kAudioHardwarePropertyProcessObjectList + kAudioProcessPropertyIsRunningInput/Output).
type darwinDetector struct {
	events chan MeetingEvent
}

// newDetector returns the darwin stub Detector.
func newDetector() Detector {
	return &darwinDetector{
		events: make(chan MeetingEvent, 4),
	}
}

func (d *darwinDetector) Events() <-chan MeetingEvent {
	return d.events
}

func (d *darwinDetector) Start(ctx context.Context) error {
	return fmt.Errorf("meeting: darwin detector not yet wired (Phase 2.7: CoreAudio process enumeration)")
}

func (d *darwinDetector) Stop() {}
