package meeting

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// knownBrowserBinaries maps browser application names (as reported by PulseAudio)
// to their xdotool WM class names for window title lookup.
var knownBrowserBinaries = map[string]bool{
	"Google Chrome":        true,
	"Google Chrome input":  true,
	"Chromium":             true,
	"Chromium input":       true,
	"Firefox":              true,
	"Firefox input":        true,
	"Brave Browser":        true,
	"Brave Browser input":  true,
	"Microsoft Edge":       true,
	"Microsoft Edge input": true,
}

// nativeAppPlatforms maps PulseAudio application names of native meeting
// apps to their platform. These can be identified without window title lookup.
var nativeAppPlatforms = map[string]Platform{
	"ZOOM VoiceEngine": PlatformZoom,
	"zoom":             PlatformZoom,
	"Slack":            PlatformSlack,
	"slack":            PlatformSlack,
}

// identifyPlatform determines the meeting platform from PulseAudio metadata
// and, for browser-based meetings, from the X11 window title.
func identifyPlatform(appName string, pid int) Platform {
	// Check native app names first (no window title needed)
	if p, ok := nativeAppPlatforms[appName]; ok {
		return p
	}

	// For browser-based apps, look up the window title
	if knownBrowserBinaries[appName] {
		title := getWindowTitleByPID(pid)
		if title != "" {
			return matchPlatformFromTitle(title)
		}
	}

	return PlatformUnknown
}

// matchPlatformFromTitle matches meeting platform keywords in a window title.
func matchPlatformFromTitle(title string) Platform {
	lower := strings.ToLower(title)

	switch {
	case strings.Contains(lower, "microsoft teams"):
		return PlatformTeams
	case strings.Contains(lower, "meet -") || strings.Contains(lower, "meet.google.com"):
		return PlatformMeet
	case strings.Contains(lower, "zoom"):
		return PlatformZoom
	case strings.Contains(lower, "webex"):
		return PlatformWebex
	case strings.Contains(lower, "slack"):
		return PlatformSlack
	}

	return PlatformUnknown
}

// getWindowTitleByPID uses xdotool to find window titles for a given PID.
// Returns the first matching window title, or "" if none found.
// On Wayland or if xdotool is not available, returns "".
func getWindowTitleByPID(pid int) string {
	xdotool, err := exec.LookPath("xdotool")
	if err != nil {
		return "" // xdotool not available (Wayland or not installed)
	}

	// Search for windows belonging to this PID
	out, err := exec.Command(xdotool, "search", "--pid", fmt.Sprintf("%d", pid), "--name", ".").Output()
	if err != nil {
		return ""
	}

	// xdotool search returns window IDs, one per line
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, wid := range lines {
		wid = strings.TrimSpace(wid)
		if wid == "" {
			continue
		}
		// Get the window name for this ID
		nameOut, err := exec.Command(xdotool, "getwindowname", wid).Output()
		if err != nil {
			continue
		}
		name := strings.TrimSpace(string(nameOut))
		if name != "" {
			return name
		}
	}

	return ""
}

// processExists checks if a process with the given PID is still running.
func processExists(pid int) bool {
	_, err := os.Stat(fmt.Sprintf("/proc/%d", pid))
	return err == nil
}
