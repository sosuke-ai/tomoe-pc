package hotkey

/*
#cgo LDFLAGS: -lX11
#include <stdio.h>
#include <X11/Xlib.h>
#include <X11/keysym.h>

// Lock modifiers that must be masked out for grabs to work
// regardless of NumLock / CapsLock / ScrollLock state.
#define NUMLOCK_MASK   Mod2Mask
#define CAPSLOCK_MASK  LockMask
#define SCROLLLOCK_MASK Mod3Mask
#define LOCK_MASK_ALL  (NUMLOCK_MASK | CAPSLOCK_MASK | SCROLLLOCK_MASK)

static Display *hk_display = NULL;
static int hk_grab_error = 0;

static int hk_error_handler(Display *d, XErrorEvent *e) {
	char buf[256];
	XGetErrorText(d, e->error_code, buf, sizeof(buf));
	fprintf(stderr, "hotkey: X11 error request=%d error=%d (%s)\n",
		e->request_code, e->error_code, buf);
	if (e->request_code == 33) { // X_GrabKey
		hk_grab_error = 1;
	}
	return 0;
}

static int hk_init() {
	if (hk_display != NULL) return 0;
	XInitThreads(); // required: grab/ungrab from Go goroutines, event loop on locked thread
	hk_display = XOpenDisplay(NULL);
	if (hk_display == NULL) return -1;
	XSetErrorHandler(hk_error_handler);
	return 0;
}

// Grab a key with all combinations of lock modifiers so the grab works
// regardless of NumLock / CapsLock / ScrollLock state.
// Returns 0 on success, -1 if display unavailable, -2 if grab failed.
static int hk_grab(unsigned int mod, unsigned int keycode) {
	if (hk_display == NULL) return -1;
	hk_grab_error = 0;
	Window root = DefaultRootWindow(hk_display);
	unsigned int locks[] = {0, NUMLOCK_MASK, CAPSLOCK_MASK, SCROLLLOCK_MASK,
		NUMLOCK_MASK|CAPSLOCK_MASK, NUMLOCK_MASK|SCROLLLOCK_MASK,
		CAPSLOCK_MASK|SCROLLLOCK_MASK, NUMLOCK_MASK|CAPSLOCK_MASK|SCROLLLOCK_MASK};
	for (int i = 0; i < 8; i++) {
		XGrabKey(hk_display, keycode, mod | locks[i], root,
			False, GrabModeAsync, GrabModeAsync);
	}
	XSync(hk_display, False);
	return hk_grab_error ? -2 : 0;
}

static int hk_ungrab(unsigned int mod, unsigned int keycode) {
	if (hk_display == NULL) return -1;
	Window root = DefaultRootWindow(hk_display);
	unsigned int locks[] = {0, NUMLOCK_MASK, CAPSLOCK_MASK, SCROLLLOCK_MASK,
		NUMLOCK_MASK|CAPSLOCK_MASK, NUMLOCK_MASK|SCROLLLOCK_MASK,
		CAPSLOCK_MASK|SCROLLLOCK_MASK, NUMLOCK_MASK|CAPSLOCK_MASK|SCROLLLOCK_MASK};
	for (int i = 0; i < 8; i++) {
		XUngrabKey(hk_display, keycode, mod | locks[i], root);
	}
	XSync(hk_display, False);
	return 0;
}

// Blocks until next KeyPress event. Returns the keycode and clean modifier state
// (lock modifiers stripped). Returns 1 on success, -1 on error.
static int hk_next_key_event(unsigned int *out_keycode, unsigned int *out_mod) {
	if (hk_display == NULL) return -1;
	XEvent ev;
	while (1) {
		XNextEvent(hk_display, &ev);
		if (ev.type == KeyPress) {
			*out_keycode = ev.xkey.keycode;
			*out_mod = ev.xkey.state & ~LOCK_MASK_ALL;
			return 1;
		}
	}
}

static unsigned int hk_keysym_to_keycode(unsigned int keysym) {
	if (hk_display == NULL) return 0;
	return XKeysymToKeycode(hk_display, keysym);
}
*/
import "C"

import (
	"fmt"
	"runtime"
	"sync"
)

func init() {
	if C.hk_init() != 0 {
		// X11 not available — hotkeys won't work but don't crash
		return
	}
}

// registryKey identifies a grabbed key combination.
type registryKey struct {
	keycode uint32
	mod     uint32
}

// Global listener registry and single dispatch loop.
var (
	registryMu   sync.Mutex
	registry     = make(map[registryKey]*linuxListener)
	dispatchOnce sync.Once
)

// linuxListener implements Listener using direct X11 key grabs.
// All listeners share a single X11 display connection and event dispatch loop.
type linuxListener struct {
	mod     uint32
	keysym  uint32
	keycode C.uint
	keydown chan struct{}
	mu      sync.Mutex
	running bool
}

// NewListener creates a Listener for the given binding string.
func NewListener(bindingStr string) (Listener, error) {
	binding, err := ParseBinding(bindingStr)
	if err != nil {
		return nil, err
	}

	var mod uint32
	for _, m := range binding.Modifiers {
		switch m {
		case "Super":
			mod |= C.Mod4Mask
		case "Ctrl":
			mod |= C.ControlMask
		case "Shift":
			mod |= C.ShiftMask
		case "Alt":
			mod |= C.Mod1Mask
		default:
			return nil, fmt.Errorf("unsupported modifier: %s", m)
		}
	}

	keysym, err := keyToKeysym(binding.Key)
	if err != nil {
		return nil, err
	}

	keycode := C.hk_keysym_to_keycode(C.uint(keysym))
	if keycode == 0 {
		return nil, fmt.Errorf("could not map key %s to X11 keycode", binding.Key)
	}

	return &linuxListener{
		mod:     mod,
		keysym:  keysym,
		keycode: keycode,
		keydown: make(chan struct{}, 1),
	}, nil
}

func (l *linuxListener) Register() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.running {
		return fmt.Errorf("hotkey already registered")
	}

	ret := C.hk_grab(C.uint(l.mod), l.keycode)
	if ret == -1 {
		return fmt.Errorf("failed to grab hotkey (X11 display not available)")
	}
	if ret == -2 {
		return fmt.Errorf("XGrabKey failed (BadAccess — another app may hold this grab)")
	}

	l.running = true

	// Add to registry and ensure dispatch loop is running
	key := registryKey{keycode: uint32(l.keycode), mod: l.mod}
	registryMu.Lock()
	registry[key] = l
	registryMu.Unlock()

	fmt.Printf("hotkey: registered keycode=%d mod=0x%x\n", uint32(l.keycode), l.mod)

	dispatchOnce.Do(func() {
		go dispatchLoop()
	})

	return nil
}

func (l *linuxListener) Keydown() <-chan struct{} {
	return l.keydown
}

func (l *linuxListener) Unregister() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.running {
		return nil
	}
	l.running = false

	// Remove from registry immediately — dispatch loop won't route to us anymore
	key := registryKey{keycode: uint32(l.keycode), mod: l.mod}
	registryMu.Lock()
	delete(registry, key)
	registryMu.Unlock()

	C.hk_ungrab(C.uint(l.mod), l.keycode)
	return nil
}

// ReGrabAll re-grabs all registered hotkeys on the X11 server.
// Call this after operations that may interfere with X11 key grabs
// (e.g., audio device init/close, GTK operations).
func ReGrabAll() {
	registryMu.Lock()
	defer registryMu.Unlock()
	for key := range registry {
		ret := C.hk_grab(C.uint(key.mod), C.uint(key.keycode))
		if ret != 0 {
			fmt.Printf("hotkey: re-grab FAILED keycode=%d mod=0x%x ret=%d\n", key.keycode, key.mod, ret)
		} else {
			fmt.Printf("hotkey: re-grabbed keycode=%d mod=0x%x\n", key.keycode, key.mod)
		}
	}
}

// dispatchLoop is the single X11 event loop. It reads KeyPress events and
// dispatches them to the correct listener based on keycode + modifiers.
// Started once on first Register() call, runs for the lifetime of the process.
func dispatchLoop() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	fmt.Println("hotkey: dispatch loop started")

	for {
		var keycode, mod C.uint
		if C.hk_next_key_event(&keycode, &mod) != 1 {
			fmt.Println("hotkey: dispatch loop exiting (display error)")
			return
		}

		key := registryKey{keycode: uint32(keycode), mod: uint32(mod)}
		registryMu.Lock()
		l, ok := registry[key]
		n := len(registry)
		registryMu.Unlock()

		if ok {
			select {
			case l.keydown <- struct{}{}:
				fmt.Printf("hotkey: dispatched keycode=%d mod=0x%x\n", key.keycode, key.mod)
			default:
				fmt.Printf("hotkey: dropped keycode=%d mod=0x%x (channel full)\n", key.keycode, key.mod)
			}
		} else {
			fmt.Printf("hotkey: unmatched event keycode=%d mod=0x%x (registry has %d entries)\n",
				key.keycode, key.mod, n)
		}
	}
}

// keyToKeysym converts a key name to an X11 keysym.
func keyToKeysym(key string) (uint32, error) {
	switch key {
	case "A":
		return 0x0061, nil
	case "B":
		return 0x0062, nil
	case "C":
		return 0x0063, nil
	case "D":
		return 0x0064, nil
	case "E":
		return 0x0065, nil
	case "F":
		return 0x0066, nil
	case "G":
		return 0x0067, nil
	case "H":
		return 0x0068, nil
	case "I":
		return 0x0069, nil
	case "J":
		return 0x006a, nil
	case "K":
		return 0x006b, nil
	case "L":
		return 0x006c, nil
	case "M":
		return 0x006d, nil
	case "N":
		return 0x006e, nil
	case "O":
		return 0x006f, nil
	case "P":
		return 0x0070, nil
	case "Q":
		return 0x0071, nil
	case "R":
		return 0x0072, nil
	case "S":
		return 0x0073, nil
	case "T":
		return 0x0074, nil
	case "U":
		return 0x0075, nil
	case "V":
		return 0x0076, nil
	case "W":
		return 0x0077, nil
	case "X":
		return 0x0078, nil
	case "Y":
		return 0x0079, nil
	case "Z":
		return 0x007a, nil
	case "0":
		return 0x0030, nil
	case "1":
		return 0x0031, nil
	case "2":
		return 0x0032, nil
	case "3":
		return 0x0033, nil
	case "4":
		return 0x0034, nil
	case "5":
		return 0x0035, nil
	case "6":
		return 0x0036, nil
	case "7":
		return 0x0037, nil
	case "8":
		return 0x0038, nil
	case "9":
		return 0x0039, nil
	case "SPACE":
		return 0x0020, nil
	case "RETURN", "ENTER":
		return 0xff0d, nil
	case "ESCAPE", "ESC":
		return 0xff1b, nil
	case "TAB":
		return 0xff09, nil
	case "DELETE":
		return 0xffff, nil
	case "LEFT":
		return 0xff51, nil
	case "RIGHT":
		return 0xff53, nil
	case "UP":
		return 0xff52, nil
	case "DOWN":
		return 0xff54, nil
	case "F1":
		return 0xffbe, nil
	case "F2":
		return 0xffbf, nil
	case "F3":
		return 0xffc0, nil
	case "F4":
		return 0xffc1, nil
	case "F5":
		return 0xffc2, nil
	case "F6":
		return 0xffc3, nil
	case "F7":
		return 0xffc4, nil
	case "F8":
		return 0xffc5, nil
	case "F9":
		return 0xffc6, nil
	case "F10":
		return 0xffc7, nil
	case "F11":
		return 0xffc8, nil
	case "F12":
		return 0xffc9, nil
	}
	return 0, fmt.Errorf("unsupported key: %s", key)
}
