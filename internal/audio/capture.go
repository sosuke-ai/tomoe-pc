package audio

// DeviceInfo describes an audio input device.
type DeviceInfo struct {
	ID        string
	Name      string
	IsDefault bool
}

// Capturer captures audio from a microphone.
type Capturer interface {
	// Start begins capturing audio.
	Start() error

	// Stop stops capturing audio.
	Stop() error

	// Samples returns the captured audio as float32 PCM at 16kHz mono.
	Samples() []float32

	// Reset clears the captured audio buffer.
	Reset()

	// Close releases all resources.
	Close()
}

// ListDevices returns available audio capture devices.
func ListDevices() ([]DeviceInfo, error) {
	return listDevices()
}

// NewCapturer creates a Capturer for the specified device.
// Pass "default" or empty string for the default device.
func NewCapturer(device string) (Capturer, error) {
	return newCapturer(device)
}
