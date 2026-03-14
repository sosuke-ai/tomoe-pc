package meeting

/*
#cgo pkg-config: libpulse
#include <pulse/pulseaudio.h>
#include <stdlib.h>
#include <string.h>

// ─── Globals ─────────────────────────────────────────────────────────────────
// Single PulseAudio connection for the process, following the same pattern
// as internal/hotkey/hotkey_linux.go's static X11 display connection.

static pa_mainloop *pa_ml = NULL;
static pa_context *pa_ctx = NULL;
static volatile int pa_connected = 0;
static volatile int pa_quit = 0;

// ─── Forward declarations for Go callbacks ───────────────────────────────────

extern void goPulseSubscribeCallback(int facility, int eventType, unsigned int idx);
extern void goPulseSinkInputInfo(unsigned int idx, char *appName, char *pid, int eol);
extern void goPulseSourceOutputInfo(unsigned int idx, char *appName, char *pid, int eol);

// ─── C callbacks ─────────────────────────────────────────────────────────────

static void subscribe_cb(pa_context *c, pa_subscription_event_type_t t,
                         uint32_t idx, void *userdata) {
    int facility = (int)(t & PA_SUBSCRIPTION_EVENT_FACILITY_MASK);
    int type = (int)(t & PA_SUBSCRIPTION_EVENT_TYPE_MASK);
    goPulseSubscribeCallback(facility, type, idx);
}

static void sink_input_info_cb(pa_context *c, const pa_sink_input_info *i,
                                int eol, void *userdata) {
    if (eol || !i) {
        goPulseSinkInputInfo(0, (char*)"", (char*)"", 1);
        return;
    }
    const char *app = pa_proplist_gets(i->proplist, PA_PROP_APPLICATION_NAME);
    const char *pid = pa_proplist_gets(i->proplist, PA_PROP_APPLICATION_PROCESS_ID);
    goPulseSinkInputInfo(i->index, (char*)(app ? app : ""), (char*)(pid ? pid : ""), 0);
}

static void source_output_info_cb(pa_context *c, const pa_source_output_info *i,
                                   int eol, void *userdata) {
    if (eol || !i) {
        goPulseSourceOutputInfo(0, (char*)"", (char*)"", 1);
        return;
    }
    const char *app = pa_proplist_gets(i->proplist, PA_PROP_APPLICATION_NAME);
    const char *pid = pa_proplist_gets(i->proplist, PA_PROP_APPLICATION_PROCESS_ID);
    goPulseSourceOutputInfo(i->index, (char*)(app ? app : ""), (char*)(pid ? pid : ""), 0);
}

// ─── Context state callback ─────────────────────────────────────────────────

static void context_state_cb(pa_context *c, void *userdata) {
    pa_context_state_t state = pa_context_get_state(c);
    switch (state) {
    case PA_CONTEXT_READY:
        pa_connected = 1;
        break;
    case PA_CONTEXT_FAILED:
    case PA_CONTEXT_TERMINATED:
        pa_connected = 0;
        pa_quit = 1;
        break;
    default:
        break;
    }
}

// ─── Lifecycle functions ─────────────────────────────────────────────────────

static int pulse_init_connection(void) {
    if (pa_ml != NULL) return 0; // already initialized

    pa_ml = pa_mainloop_new();
    if (!pa_ml) return -1;

    pa_mainloop_api *api = pa_mainloop_get_api(pa_ml);
    pa_ctx = pa_context_new(api, "tomoe-meeting-detector");
    if (!pa_ctx) {
        pa_mainloop_free(pa_ml);
        pa_ml = NULL;
        return -2;
    }

    pa_context_set_state_callback(pa_ctx, context_state_cb, NULL);

    if (pa_context_connect(pa_ctx, NULL, PA_CONTEXT_NOFLAGS, NULL) < 0) {
        pa_context_unref(pa_ctx);
        pa_ctx = NULL;
        pa_mainloop_free(pa_ml);
        pa_ml = NULL;
        return -3;
    }

    // Block until connected or failed
    pa_quit = 0;
    while (!pa_connected && !pa_quit) {
        if (pa_mainloop_iterate(pa_ml, 1, NULL) < 0) {
            pa_context_disconnect(pa_ctx);
            pa_context_unref(pa_ctx);
            pa_ctx = NULL;
            pa_mainloop_free(pa_ml);
            pa_ml = NULL;
            return -4;
        }
    }

    if (!pa_connected) {
        pa_context_unref(pa_ctx);
        pa_ctx = NULL;
        pa_mainloop_free(pa_ml);
        pa_ml = NULL;
        return -5;
    }

    return 0;
}

static int pulse_subscribe_events(void) {
    if (!pa_ctx || !pa_connected) return -1;

    pa_context_set_subscribe_callback(pa_ctx, subscribe_cb, NULL);

    pa_operation *op = pa_context_subscribe(pa_ctx,
        PA_SUBSCRIPTION_MASK_SINK_INPUT | PA_SUBSCRIPTION_MASK_SOURCE_OUTPUT,
        NULL, NULL);
    if (op) {
        pa_operation_unref(op);
    }
    return 0;
}

// Run one iteration of the mainloop. Returns 0 on success, negative on error.
static int pulse_iterate_once(int block) {
    if (!pa_ml) return -1;
    int ret = 0;
    if (pa_mainloop_iterate(pa_ml, block, &ret) < 0) return -1;
    return ret;
}

// Request sink-input info list (async — results arrive via callback).
static void pulse_request_sink_inputs(void) {
    if (!pa_ctx || !pa_connected) return;
    pa_operation *op = pa_context_get_sink_input_info_list(pa_ctx, sink_input_info_cb, NULL);
    if (op) pa_operation_unref(op);
}

// Request source-output info list (async — results arrive via callback).
static void pulse_request_source_outputs(void) {
    if (!pa_ctx || !pa_connected) return;
    pa_operation *op = pa_context_get_source_output_info_list(pa_ctx, source_output_info_cb, NULL);
    if (op) pa_operation_unref(op);
}

// Wake a blocking pa_mainloop_iterate so the Go event loop can exit.
static void pulse_request_quit(void) {
    if (pa_ml) {
        pa_mainloop_quit(pa_ml, 0);
    }
}

static void pulse_cleanup_connection(void) {
    if (pa_ctx) {
        pa_context_disconnect(pa_ctx);
        pa_context_unref(pa_ctx);
        pa_ctx = NULL;
    }
    if (pa_ml) {
        pa_mainloop_free(pa_ml);
        pa_ml = NULL;
    }
    pa_connected = 0;
    pa_quit = 0;
}
*/
import "C"

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"time"
	"unsafe"
)

// ─── Active detector routing ─────────────────────────────────────────────────
// PulseAudio callbacks are global (C function pointers can't carry Go closures),
// so we route events to the active Detector instance via a package-level variable.

var (
	activeDetectorMu sync.Mutex
	activeDetector   *Detector
)

func setActiveDetector(d *Detector) {
	activeDetectorMu.Lock()
	activeDetector = d
	activeDetectorMu.Unlock()
}

// ─── Callback data collection ────────────────────────────────────────────────
// PulseAudio info callbacks fire multiple times (once per item + final eol).
// We collect items into slices and signal completion via channels.

var (
	// pulseQueryMu serializes all PA info queries to prevent concurrent
	// callers from racing on the global collector state.
	pulseQueryMu sync.Mutex

	sinkInputMu      sync.Mutex
	sinkInputItems   []streamInfo
	sinkInputDone    = make(chan struct{}, 1)
	sourceOutputMu   sync.Mutex
	sourceOutputItems []streamInfo
	sourceOutputDone  = make(chan struct{}, 1)
)

//export goPulseSubscribeCallback
func goPulseSubscribeCallback(facility, eventType C.int, idx C.uint) {
	activeDetectorMu.Lock()
	d := activeDetector
	activeDetectorMu.Unlock()
	if d != nil {
		d.onSubscribeEvent(int(facility), int(eventType), uint32(idx))
	}
}

//export goPulseSinkInputInfo
func goPulseSinkInputInfo(idx C.uint, appName, pid *C.char, eol C.int) {
	if eol != 0 {
		select {
		case sinkInputDone <- struct{}{}:
		default:
		}
		return
	}
	pidStr := C.GoString(pid)
	pidNum, _ := strconv.Atoi(pidStr)
	si := streamInfo{
		Index:   uint32(idx),
		AppName: C.GoString(appName),
		PID:     pidNum,
	}
	sinkInputMu.Lock()
	sinkInputItems = append(sinkInputItems, si)
	sinkInputMu.Unlock()
}

//export goPulseSourceOutputInfo
func goPulseSourceOutputInfo(idx C.uint, appName, pid *C.char, eol C.int) {
	if eol != 0 {
		select {
		case sourceOutputDone <- struct{}{}:
		default:
		}
		return
	}
	pidStr := C.GoString(pid)
	pidNum, _ := strconv.Atoi(pidStr)
	si := streamInfo{
		Index:   uint32(idx),
		AppName: C.GoString(appName),
		PID:     pidNum,
	}
	sourceOutputMu.Lock()
	sourceOutputItems = append(sourceOutputItems, si)
	sourceOutputMu.Unlock()
}

// Ensure C strings are used (suppress unused import warning for "unsafe").
var _ = unsafe.Pointer(nil)

// ─── Go wrappers ─────────────────────────────────────────────────────────────

func pulseInit() error {
	ret := C.pulse_init_connection()
	if ret != 0 {
		return fmt.Errorf("pulse_init_connection returned %d", ret)
	}
	return nil
}

func pulseSubscribe() error {
	ret := C.pulse_subscribe_events()
	if ret != 0 {
		return fmt.Errorf("pulse_subscribe_events returned %d", ret)
	}
	return nil
}

func pulseCleanup() {
	C.pulse_cleanup_connection()
}

// pulseQuit wakes a blocked pa_mainloop_iterate so the event loop exits.
func pulseQuit() {
	C.pulse_request_quit()
}

// pulseEventLoop runs the PulseAudio mainloop on a locked OS thread.
// Blocks until ctx is cancelled.
func pulseEventLoop(ctx context.Context) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	fmt.Println("meeting: PulseAudio event loop started")

	for {
		select {
		case <-ctx.Done():
			fmt.Println("meeting: PulseAudio event loop stopping")
			return
		default:
		}
		// Block in PA mainloop until an event arrives. This avoids
		// busy-spinning at 100% CPU with non-blocking iterate(0).
		ret := C.pulse_iterate_once(1)
		if ret < 0 {
			fmt.Println("meeting: PulseAudio mainloop error, stopping")
			return
		}
	}
}

// pulseListSinkInputs requests sink-input info and waits for results.
// Must NOT be called from the PulseAudio event loop thread.
func pulseListSinkInputs() []streamInfo {
	pulseQueryMu.Lock()
	defer pulseQueryMu.Unlock()

	sinkInputMu.Lock()
	sinkInputItems = nil
	sinkInputMu.Unlock()

	// Drain any stale done signal
	select {
	case <-sinkInputDone:
	default:
	}

	C.pulse_request_sink_inputs()

	// Wait for callback to signal completion (with timeout)
	select {
	case <-sinkInputDone:
	case <-waitTimeout(3):
	}

	sinkInputMu.Lock()
	result := make([]streamInfo, len(sinkInputItems))
	copy(result, sinkInputItems)
	sinkInputMu.Unlock()
	return result
}

// pulseListSourceOutputs requests source-output info and waits for results.
// Must NOT be called from the PulseAudio event loop thread.
func pulseListSourceOutputs() []streamInfo {
	pulseQueryMu.Lock()
	defer pulseQueryMu.Unlock()

	sourceOutputMu.Lock()
	sourceOutputItems = nil
	sourceOutputMu.Unlock()

	// Drain any stale done signal
	select {
	case <-sourceOutputDone:
	default:
	}

	C.pulse_request_source_outputs()

	// Wait for callback to signal completion (with timeout)
	select {
	case <-sourceOutputDone:
	case <-waitTimeout(3):
	}

	sourceOutputMu.Lock()
	result := make([]streamInfo, len(sourceOutputItems))
	copy(result, sourceOutputItems)
	sourceOutputMu.Unlock()
	return result
}

func waitTimeout(seconds int) <-chan time.Time {
	return time.After(time.Duration(seconds) * time.Second)
}

// ─── PulseAudio event type helpers ───────────────────────────────────────────

const (
	paFacilitySinkInput    = C.PA_SUBSCRIPTION_EVENT_SINK_INPUT
	paFacilitySourceOutput = C.PA_SUBSCRIPTION_EVENT_SOURCE_OUTPUT
	paEventNew             = C.PA_SUBSCRIPTION_EVENT_NEW
	paEventRemove          = C.PA_SUBSCRIPTION_EVENT_REMOVE
)

func isSourceOutputNew(facility, eventType int) bool {
	return facility == int(paFacilitySourceOutput) && eventType == int(paEventNew)
}

func isSourceOutputRemove(facility, eventType int) bool {
	return facility == int(paFacilitySourceOutput) && eventType == int(paEventRemove)
}

func isSinkInputRemove(facility, eventType int) bool {
	return facility == int(paFacilitySinkInput) && eventType == int(paEventRemove)
}
