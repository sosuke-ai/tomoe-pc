//go:build darwin

package main

/*
#cgo LDFLAGS: -framework CoreAudio -framework CoreFoundation
#include <stdlib.h>
#include <string.h>
#include <sys/types.h>
#include <CoreAudio/CoreAudio.h>
#include <CoreAudio/AudioHardware.h>
#include <CoreFoundation/CoreFoundation.h>

// Selectors introduced in macOS 14.4 for per-process audio attribution.
// Some SDKs guard these behind macOS version checks; we declare them
// unconditionally so the spike compiles even if the build SDK is older.
#ifndef kAudioHardwarePropertyProcessObjectList
#define kAudioHardwarePropertyProcessObjectList 'pro#'
#endif
#ifndef kAudioProcessPropertyPID
#define kAudioProcessPropertyPID 'ppid'
#endif
#ifndef kAudioProcessPropertyBundleID
#define kAudioProcessPropertyBundleID 'pbid'
#endif
#ifndef kAudioProcessPropertyIsRunning
#define kAudioProcessPropertyIsRunning 'pir?'
#endif
#ifndef kAudioProcessPropertyIsRunningInput
#define kAudioProcessPropertyIsRunningInput 'piri'
#endif
#ifndef kAudioProcessPropertyIsRunningOutput
#define kAudioProcessPropertyIsRunningOutput 'piro'
#endif

extern void goProcessListChanged(void);

static OSStatus processListListenerProc(AudioObjectID inObjectID,
                                         UInt32 inNumberAddresses,
                                         const AudioObjectPropertyAddress *inAddresses,
                                         void *inClientData) {
    (void)inObjectID;
    (void)inNumberAddresses;
    (void)inAddresses;
    (void)inClientData;
    goProcessListChanged();
    return noErr;
}

// addProcessListListener registers a property listener so the kernel notifies us
// whenever a process appears in or disappears from the audio HAL.
static OSStatus addProcessListListener(void) {
    AudioObjectPropertyAddress addr = {
        kAudioHardwarePropertyProcessObjectList,
        kAudioObjectPropertyScopeGlobal,
        kAudioObjectPropertyElementMain
    };
    return AudioObjectAddPropertyListener(kAudioObjectSystemObject, &addr,
                                          processListListenerProc, NULL);
}

// processObjectList writes up to maxIDs AudioObjectIDs into the caller's buffer.
// Returns the number written, or -1 on error. Caller-allocated buffer avoids
// malloc/free across the cgo boundary.
static int processObjectList(AudioObjectID *ids, int maxIDs, OSStatus *outStatus) {
    AudioObjectPropertyAddress addr = {
        kAudioHardwarePropertyProcessObjectList,
        kAudioObjectPropertyScopeGlobal,
        kAudioObjectPropertyElementMain
    };
    UInt32 dataSize = 0;
    OSStatus s = AudioObjectGetPropertyDataSize(kAudioObjectSystemObject, &addr,
                                                 0, NULL, &dataSize);
    if (outStatus) *outStatus = s;
    if (s != noErr) return -1;
    int count = (int)(dataSize / sizeof(AudioObjectID));
    if (count > maxIDs) count = maxIDs;
    UInt32 actualSize = (UInt32)(count * sizeof(AudioObjectID));
    s = AudioObjectGetPropertyData(kAudioObjectSystemObject, &addr,
                                    0, NULL, &actualSize, ids);
    if (outStatus) *outStatus = s;
    if (s != noErr) return -1;
    return count;
}

static int32_t processPID(AudioObjectID id) {
    pid_t pid = 0;
    UInt32 size = sizeof(pid);
    AudioObjectPropertyAddress addr = {
        kAudioProcessPropertyPID,
        kAudioObjectPropertyScopeGlobal,
        kAudioObjectPropertyElementMain
    };
    if (AudioObjectGetPropertyData(id, &addr, 0, NULL, &size, &pid) != noErr) {
        return -1;
    }
    return (int32_t)pid;
}

// processBundleID returns the bundle identifier as a malloc'd C string, or NULL.
// Caller must free.
static char *processBundleID(AudioObjectID id) {
    CFStringRef cfStr = NULL;
    UInt32 size = sizeof(cfStr);
    AudioObjectPropertyAddress addr = {
        kAudioProcessPropertyBundleID,
        kAudioObjectPropertyScopeGlobal,
        kAudioObjectPropertyElementMain
    };
    if (AudioObjectGetPropertyData(id, &addr, 0, NULL, &size, &cfStr) != noErr || cfStr == NULL) {
        return NULL;
    }
    CFIndex maxLen = CFStringGetMaximumSizeForEncoding(CFStringGetLength(cfStr),
                                                        kCFStringEncodingUTF8) + 1;
    char *buf = (char *)malloc((size_t)maxLen);
    if (buf == NULL) {
        CFRelease(cfStr);
        return NULL;
    }
    if (!CFStringGetCString(cfStr, buf, maxLen, kCFStringEncodingUTF8)) {
        free(buf);
        CFRelease(cfStr);
        return NULL;
    }
    CFRelease(cfStr);
    return buf;
}

static int processBoolProperty(AudioObjectID id, AudioObjectPropertySelector sel) {
    UInt32 v = 0;
    UInt32 size = sizeof(v);
    AudioObjectPropertyAddress addr = {
        sel,
        kAudioObjectPropertyScopeGlobal,
        kAudioObjectPropertyElementMain
    };
    if (AudioObjectGetPropertyData(id, &addr, 0, NULL, &size, &v) != noErr) {
        return -1;
    }
    return v != 0 ? 1 : 0;
}

static int processIsRunning(AudioObjectID id) {
    return processBoolProperty(id, kAudioProcessPropertyIsRunning);
}
static int processIsRunningInput(AudioObjectID id) {
    return processBoolProperty(id, kAudioProcessPropertyIsRunningInput);
}
static int processIsRunningOutput(AudioObjectID id) {
    return processBoolProperty(id, kAudioProcessPropertyIsRunningOutput);
}
*/
import "C"

import (
	"fmt"
	"os/exec"
	"strings"
	"unsafe"
)

const maxProcesses = 512

// processInfo is a snapshot of one audio-process object.
type processInfo struct {
	ObjectID        uint32
	PID             int
	BundleID        string
	IsRunning       bool
	IsRunningInput  bool
	IsRunningOutput bool
}

// listProcesses enumerates every process registered with the audio HAL
// and reads its PID, bundle ID, and three running-state flags.
func listProcesses() ([]processInfo, error) {
	var ids [maxProcesses]C.AudioObjectID
	var status C.OSStatus
	n := C.processObjectList(&ids[0], C.int(maxProcesses), &status)
	if n < 0 {
		return nil, fmt.Errorf("AudioObjectGetPropertyData(ProcessObjectList) failed: OSStatus=%d (likely macOS <14.2 — public process API requires 14.2+, ideally 14.4+)", int32(status))
	}
	out := make([]processInfo, 0, int(n))
	for i := 0; i < int(n); i++ {
		id := ids[i]
		var info processInfo
		info.ObjectID = uint32(id)

		pid := int32(C.processPID(id))
		if pid < 0 {
			info.PID = -1
		} else {
			info.PID = int(pid)
		}

		if cBundle := C.processBundleID(id); cBundle != nil {
			info.BundleID = C.GoString(cBundle)
			C.free(unsafe.Pointer(cBundle))
		}

		if v := C.processIsRunning(id); v == 1 {
			info.IsRunning = true
		}
		if v := C.processIsRunningInput(id); v == 1 {
			info.IsRunningInput = true
		}
		if v := C.processIsRunningOutput(id); v == 1 {
			info.IsRunningOutput = true
		}

		out = append(out, info)
	}
	return out, nil
}

// listChangedCh is signaled when the HAL's process-object list changes
// (a process appeared or disappeared). Buffered to coalesce bursts.
var listChangedCh = make(chan struct{}, 1)

//export goProcessListChanged
func goProcessListChanged() {
	select {
	case listChangedCh <- struct{}{}:
	default:
	}
}

// startProcessListListener registers a CoreAudio property listener that
// pushes onto listChangedCh whenever the process list changes.
func startProcessListListener() error {
	if status := C.addProcessListListener(); status != 0 {
		return fmt.Errorf("AudioObjectAddPropertyListener failed: OSStatus=%d", int32(status))
	}
	return nil
}

// processListEvents returns the channel that fires on process-list changes.
func processListEvents() <-chan struct{} {
	return listChangedCh
}

// macOSVersion returns the user's macOS version as reported by sw_vers.
func macOSVersion() string {
	out, err := exec.Command("sw_vers", "-productVersion").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

// macOSBuild returns the macOS build number (e.g. "23F79").
func macOSBuild() string {
	out, err := exec.Command("sw_vers", "-buildVersion").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

// hardwareModel returns the hardware model identifier (e.g. "MacBookPro18,3").
func hardwareModel() string {
	out, err := exec.Command("sysctl", "-n", "hw.model").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}
