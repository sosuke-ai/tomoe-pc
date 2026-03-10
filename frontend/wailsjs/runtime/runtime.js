// Wails runtime bridge.
// In production, the real Wails runtime is injected as window.runtime.
// These functions delegate to the real runtime when available,
// falling back to no-ops for standalone dev/test.

export function EventsOn(eventName, callback) {
  if (window.runtime && window.runtime.EventsOn) {
    return window.runtime.EventsOn(eventName, callback);
  }
  return () => {};
}

export function EventsEmit(eventName, ...data) {
  if (window.runtime && window.runtime.EventsEmit) {
    window.runtime.EventsEmit(eventName, ...data);
  }
}

export function EventsOff(eventName, ...additionalEventNames) {
  if (window.runtime && window.runtime.EventsOff) {
    window.runtime.EventsOff(eventName, ...additionalEventNames);
  }
}

export function WindowShow() {
  if (window.runtime && window.runtime.WindowShow) {
    window.runtime.WindowShow();
  }
}

export function WindowHide() {
  if (window.runtime && window.runtime.WindowHide) {
    window.runtime.WindowHide();
  }
}

export function Quit() {
  if (window.runtime && window.runtime.Quit) {
    window.runtime.Quit();
  }
}
