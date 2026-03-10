package audio

// ListMonitorSources returns audio devices that are PulseAudio/PipeWire monitor sources.
// Monitor sources have names ending in ".monitor" and capture system audio output.
func ListMonitorSources() ([]DeviceInfo, error) {
	devices, err := ListDevices()
	if err != nil {
		return nil, err
	}

	var monitors []DeviceInfo
	for _, d := range devices {
		if d.DeviceType == Monitor {
			monitors = append(monitors, d)
		}
	}
	return monitors, nil
}

// DefaultMonitorDevice returns the first available monitor source, or empty string if none found.
func DefaultMonitorDevice() string {
	monitors, err := ListMonitorSources()
	if err != nil || len(monitors) == 0 {
		return ""
	}
	// Prefer the default monitor if available
	for _, m := range monitors {
		if m.IsDefault {
			return m.Name
		}
	}
	return monitors[0].Name
}
