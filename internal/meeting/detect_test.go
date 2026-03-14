package meeting

import (
	"testing"
)

func TestMatchPlatformFromTitle(t *testing.T) {
	tests := []struct {
		title    string
		expected Platform
	}{
		{"(10) Chat | Workflow Agent PR | Microsoft Teams - Google Chrome", PlatformTeams},
		{"Microsoft Teams", PlatformTeams},
		{"Meet - fqx-wprh-axx - Google Chrome", PlatformMeet},
		{"meet.google.com/abc-defg-hij - Google Chrome", PlatformMeet},
		{"Zoom Meeting", PlatformZoom},
		{"Zoom Workplace - Free account", PlatformZoom},
		{"Webex Meeting Center", PlatformWebex},
		{"Slack | #general", PlatformSlack},
		{"YouTube - Google Chrome", PlatformUnknown},
		{"Google Gemini - Google Chrome", PlatformUnknown},
		{"", PlatformUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			got := matchPlatformFromTitle(tt.title)
			if got != tt.expected {
				t.Errorf("matchPlatformFromTitle(%q) = %q, want %q", tt.title, got, tt.expected)
			}
		})
	}
}

func TestIdentifyPlatformNativeApps(t *testing.T) {
	tests := []struct {
		appName  string
		expected Platform
	}{
		{"ZOOM VoiceEngine", PlatformZoom},
		{"zoom", PlatformZoom},
		{"Slack", PlatformSlack},
		{"slack", PlatformSlack},
	}

	for _, tt := range tests {
		t.Run(tt.appName, func(t *testing.T) {
			// PID 0 means xdotool lookup will fail, so we're testing native path only
			got := identifyPlatform(tt.appName, 0)
			if got != tt.expected {
				t.Errorf("identifyPlatform(%q, 0) = %q, want %q", tt.appName, got, tt.expected)
			}
		})
	}
}

func TestMeetingCorrelation(t *testing.T) {
	// Simulate: source-outputs and sink-inputs with same PID = meeting
	sourceOutputs := []streamInfo{
		{Index: 826, AppName: "Google Chrome input", PID: 249501},
	}
	sinkInputs := []streamInfo{
		{Index: 834, AppName: "Google Chrome", PID: 249501},
	}

	sinkPIDs := make(map[int]string)
	for _, si := range sinkInputs {
		if si.PID > 0 {
			sinkPIDs[si.PID] = si.AppName
		}
	}

	var matchedPID int
	for _, so := range sourceOutputs {
		if so.PID > 0 {
			if _, hasSink := sinkPIDs[so.PID]; hasSink {
				matchedPID = so.PID
				break
			}
		}
	}

	if matchedPID != 249501 {
		t.Errorf("expected matched PID 249501, got %d", matchedPID)
	}
}

func TestMeetingCorrelationNoMatch(t *testing.T) {
	// Source-output and sink-input from different PIDs = no meeting
	sourceOutputs := []streamInfo{
		{Index: 826, AppName: "OBS Studio", PID: 12345},
	}
	sinkInputs := []streamInfo{
		{Index: 834, AppName: "Google Chrome", PID: 99999},
	}

	sinkPIDs := make(map[int]string)
	for _, si := range sinkInputs {
		if si.PID > 0 {
			sinkPIDs[si.PID] = si.AppName
		}
	}

	var matchedPID int
	for _, so := range sourceOutputs {
		if so.PID > 0 {
			if _, hasSink := sinkPIDs[so.PID]; hasSink {
				matchedPID = so.PID
				break
			}
		}
	}

	if matchedPID != 0 {
		t.Errorf("expected no match, got PID %d", matchedPID)
	}
}

func TestKnownBrowserBinaries(t *testing.T) {
	browsers := []string{
		"Google Chrome", "Google Chrome input",
		"Firefox", "Firefox input",
		"Brave Browser", "Brave Browser input",
		"Microsoft Edge", "Microsoft Edge input",
		"Chromium", "Chromium input",
	}

	for _, b := range browsers {
		if !knownBrowserBinaries[b] {
			t.Errorf("%q should be recognized as a browser", b)
		}
	}

	nonBrowsers := []string{"ZOOM VoiceEngine", "Spotify", "VLC media player"}
	for _, nb := range nonBrowsers {
		if knownBrowserBinaries[nb] {
			t.Errorf("%q should NOT be recognized as a browser", nb)
		}
	}
}

func TestPlatformConstants(t *testing.T) {
	// Verify platform strings are user-friendly
	platforms := []Platform{PlatformTeams, PlatformMeet, PlatformZoom, PlatformWebex, PlatformSlack, PlatformUnknown}
	for _, p := range platforms {
		if p == "" {
			t.Error("platform constant should not be empty")
		}
	}
}
