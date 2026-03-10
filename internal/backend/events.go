package backend

import (
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// emitSegments reads segments from the coordinator and emits them to the frontend.
// Must be called as a goroutine. Runs until the coordinator's segment channel is closed.
func (a *App) emitSegments() {
	if a.coordinator == nil {
		return
	}

	for seg := range a.coordinator.Segments() {
		// Append to current session
		a.mu.Lock()
		if a.currentSess != nil {
			a.currentSess.Segments = append(a.currentSess.Segments, seg)
		}
		a.mu.Unlock()

		// Emit to frontend
		wailsRuntime.EventsEmit(a.ctx, "transcript:segment", seg)
	}
}
