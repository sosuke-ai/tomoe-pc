package meeting

import (
	"testing"
	"time"
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

func TestEventTypeHelpers(t *testing.T) {
	facSink := int(paFacilitySinkInput)
	facSource := int(paFacilitySourceOutput)
	evtNew := int(paEventNew)
	evtRemove := int(paEventRemove)

	tests := []struct {
		name     string
		fn       func(int, int) bool
		facility int
		evtType  int
		want     bool
	}{
		{"isSourceOutputNew/match", isSourceOutputNew, facSource, evtNew, true},
		{"isSourceOutputNew/wrong_facility", isSourceOutputNew, facSink, evtNew, false},
		{"isSourceOutputNew/wrong_type", isSourceOutputNew, facSource, evtRemove, false},
		{"isSourceOutputRemove/match", isSourceOutputRemove, facSource, evtRemove, true},
		{"isSourceOutputRemove/wrong_type", isSourceOutputRemove, facSource, evtNew, false},
		{"isSinkInputNew/match", isSinkInputNew, facSink, evtNew, true},
		{"isSinkInputNew/wrong_facility", isSinkInputNew, facSource, evtNew, false},
		{"isSinkInputNew/wrong_type", isSinkInputNew, facSink, evtRemove, false},
		{"isSinkInputRemove/match", isSinkInputRemove, facSink, evtRemove, true},
		{"isSinkInputRemove/wrong_facility", isSinkInputRemove, facSource, evtRemove, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn(tt.facility, tt.evtType)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOnSubscribeEventDispatch(t *testing.T) {
	// Verify that onSubscribeEvent doesn't panic and respects the stopped flag.
	d := NewDetector()

	// Should not panic when called with any event type
	d.onSubscribeEvent(int(paFacilitySourceOutput), int(paEventNew), 0)
	d.onSubscribeEvent(int(paFacilitySinkInput), int(paEventNew), 0)
	d.onSubscribeEvent(int(paFacilitySourceOutput), int(paEventRemove), 0)
	d.onSubscribeEvent(int(paFacilitySinkInput), int(paEventRemove), 0)

	// After marking stopped, events should be ignored
	d.mu.Lock()
	d.stopped = true
	d.mu.Unlock()

	// Should return immediately without spawning goroutines
	d.onSubscribeEvent(int(paFacilitySourceOutput), int(paEventNew), 0)
	d.onSubscribeEvent(int(paFacilitySinkInput), int(paEventNew), 0)
}

func TestDetectorEventChannel(t *testing.T) {
	d := NewDetector()
	ch := d.Events()
	if ch == nil {
		t.Fatal("Events() returned nil channel")
	}

	// Channel should be buffered (cap 4)
	if cap(ch) != 4 {
		t.Errorf("Events() channel capacity = %d, want 4", cap(ch))
	}
}

func TestCheckForMeetingSkipsWhenActive(t *testing.T) {
	d := NewDetector()

	// Set an active meeting — checkForMeeting should return early
	d.mu.Lock()
	d.active = &trackedMeeting{pid: 12345, platform: PlatformTeams}
	d.mu.Unlock()

	// Should not panic or block
	d.checkForMeeting()
}

func TestCheckForMeetingEndNoActive(t *testing.T) {
	d := NewDetector()

	// No active meeting — checkForMeetingEnd should return early
	d.checkForMeetingEnd()
}

func TestDebounceDelay(t *testing.T) {
	if debounceDelay != 2*time.Second {
		t.Errorf("debounceDelay = %v, want 2s", debounceDelay)
	}
}

func TestHealthCheckInterval(t *testing.T) {
	if healthCheckInterval != 30*time.Second {
		t.Errorf("healthCheckInterval = %v, want 30s", healthCheckInterval)
	}
}
