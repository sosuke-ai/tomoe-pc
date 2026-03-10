package audio

import "testing"

func TestDeviceTypeConstants(t *testing.T) {
	if Input != 0 {
		t.Errorf("Input = %d, want 0", Input)
	}
	if Monitor != 1 {
		t.Errorf("Monitor = %d, want 1", Monitor)
	}
}

func TestDeviceInfoDeviceType(t *testing.T) {
	mic := DeviceInfo{
		ID:         "1",
		Name:       "Built-in Microphone",
		IsDefault:  true,
		DeviceType: Input,
	}
	monitor := DeviceInfo{
		ID:         "2",
		Name:       "alsa_output.pci-0000_00_1f.3.analog-stereo.monitor",
		IsDefault:  false,
		DeviceType: Monitor,
	}

	if mic.DeviceType != Input {
		t.Error("mic DeviceType should be Input")
	}
	if monitor.DeviceType != Monitor {
		t.Error("monitor DeviceType should be Monitor")
	}
}
