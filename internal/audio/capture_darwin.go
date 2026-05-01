//go:build darwin

package audio

import "fmt"

// listDevices returns an error stub on darwin.
// Phase 0 stub — replaced in Phase 2.6 with malgo CoreAudio backend.
func listDevices() ([]DeviceInfo, error) {
	return nil, fmt.Errorf("audio: darwin capture not yet implemented")
}

// newCapturer returns an error stub on darwin.
// Phase 0 stub — replaced in Phase 2.6 with malgo CoreAudio backend.
func newCapturer(device string, deviceType DeviceType) (Capturer, error) {
	return nil, fmt.Errorf("audio: darwin capture not yet implemented (device=%q type=%v)", device, deviceType)
}
