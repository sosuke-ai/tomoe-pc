# Tomoe macOS Spike — Tester Instructions

Thank you for helping validate Tomoe's macOS port. This is a small probe that
**does not record audio, send anything over the network, or modify your
system.** It only reads metadata about which apps are using the microphone
and speakers, then writes that information to a text log on disk.

The log is what the developer uses to confirm the meeting auto-detection
approach works correctly on real hardware.

## Requirements

- A Mac with **Apple Silicon** (M1, M2, M3, M4, etc.)
- **macOS 14.4 or later** (`Apple → About This Mac`)

If your Mac is older than 14.4 the spike will print a clear error and exit
without doing any harm. (The CoreAudio API it depends on landed in 14.4.)

## Run it

1. **Download** the `tomoe-mac-spike-arm64` artifact zip from the developer's
   GitHub Actions link. Unzip it. You should have a single file called
   `tomoe-mac-spike`.

2. **Open Terminal** (Applications → Utilities → Terminal) and navigate to
   wherever the file is, e.g.:

   ```
   cd ~/Downloads
   ```

3. **Strip the quarantine attribute** so macOS doesn't block it (you're
   getting it directly from the developer; this is safe):

   ```
   xattr -cr tomoe-mac-spike
   chmod +x tomoe-mac-spike
   ```

4. **Run it:**

   ```
   ./tomoe-mac-spike
   ```

   The spike will print its findings to your screen and also save a log
   file in the same directory (named `tomoe-spike-<timestamp>.log`).

5. **Use your computer normally for at least 30 minutes.** The most useful
   thing is to take 1-3 real meeting calls during this window — Zoom,
   Google Meet, Microsoft Teams, FaceTime, Webex, Discord voice, anything
   you'd normally use. The spike doesn't care which platform.

   Background activity (Spotify, browser tabs, voice memos, Siri) is fine
   too — the developer wants to see those signals in the log to tune
   false-positive filtering.

6. **Press `Ctrl+C`** in the Terminal window to stop the spike. It'll
   print a `=== SUMMARY ===` section showing what it observed.

7. **Send the log file back** to the developer. The file is named
   `tomoe-spike-YYYY-MM-DD-HHMMSS.log` and lives in whatever directory
   you ran the spike from (probably `~/Downloads`). Email or drop it
   into chat — it's plain text, usually under 100 KB.

## What the spike does NOT do

- It does not request microphone permission. It only asks the system
  *who else* is using the mic, not the audio data itself.
- It does not request screen recording, accessibility, or any other
  permission. macOS may show no prompts at all.
- It does not connect to the internet.
- It does not write to anything outside the directory you ran it from.

## What the log contains

A header (your macOS version, hardware model — to confirm we're testing
the right environment), a list of every app currently registered with
the audio system, and time-stamped events as apps start/stop using the
mic and speakers. Roughly the level of detail Activity Monitor's
"Microphone" tab shows, but with timestamps.

## If something goes wrong

- **"cannot be opened because the developer cannot be verified"** —
  rerun step 3 (`xattr -cr tomoe-mac-spike`). If that doesn't help,
  open `System Settings → Privacy & Security`, scroll to the bottom,
  and click "Open Anyway" next to the Tomoe Spike entry.
- **"AudioObjectGetPropertyData(ProcessObjectList) failed"** — your
  macOS version is probably older than 14.2. Check `Apple → About This
  Mac`. If you're on 14.0-14.3 the developer will adjust the approach.
- **Anything else** — please send the log file regardless. Even a
  failed run is useful information.

Thanks again!
