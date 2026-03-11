# Tomoe PC

Local-first speech-to-text desktop application for Linux. Captures microphone and system audio, transcribes locally using NVIDIA Parakeet TDT 0.6B v3 (INT8, 25 languages) via sherpa-onnx, and delivers text to clipboard or a live GUI transcript with speaker identification.

## Features

- **CLI dictation mode** — global hotkey triggers mic capture, transcribes speech, pastes result into the focused window (detects terminals for Ctrl+Shift+V)
- **Meeting transcription GUI** — Wails v2 desktop app with live scrolling transcript, mic + system audio capture, speaker identification, session management
- **Speaker identification** — mic audio labeled "You", system audio speakers clustered via 3D-Speaker embeddings ("Person 1", "Person 2", etc.)
- **Session management** — save, load, export (Markdown, plain text, SRT), delete sessions with recorded audio (MP3)
- **GPU acceleration** — NVIDIA CUDA via ONNX Runtime with automatic CPU fallback
- **25 languages** — automatic language detection via Parakeet TDT v3 INT8
- **System tray** — background operation with AppIndicator3 tray icon

## Quick Start

```bash
# Install system dependencies (Ubuntu 24.04+)
make dev-deps

# Install Go tools (golangci-lint, wails)
make dev-tools

# Build CLI + GUI
make build

# First run — auto-detects system, creates config, downloads models (~375MB)
./tomoe

# Or launch the GUI
./tomoe-gui
```

## Build Requirements

- Go 1.22+
- C/C++ toolchain (gcc/g++ for cgo)
- ONNX Runtime shared library (>=1.17.0) + sherpa-onnx C API
- Node.js 18+ (for frontend)
- `libwebkit2gtk-4.1-dev` (for GUI)
- `xdotool`, `xprop` (for clipboard auto-paste on X11)

## Make Targets

```bash
make build            # Build CLI + GUI (if webkit2gtk available)
make build-gui        # Build GUI binary only
make build-cuda       # Build with CUDA support
make test             # Run unit tests
make vet              # Run go vet
make lint             # Run golangci-lint
make dev-gui          # Wails dev mode with hot-reload
make download-model   # Download Parakeet TDT v3 INT8 + Silero VAD + Speaker Embedding
make install          # Install to $GOPATH/bin
make install-gpu      # Install CUDA toolkit + sherpa-onnx GPU libraries
make clean            # Remove build artifacts
```

## CLI Commands

```
tomoe                     # Start daemon (auto-init on first run)
tomoe start               # Alias for above
tomoe init                # Manual system detection + config generation + model download
tomoe stop                # Stop daemon
tomoe status              # Show daemon/system/model/GPU info
tomoe transcribe <file>   # Transcribe audio file (WAV, FLAC, OGG)
tomoe model download      # Force re-download model
tomoe model status        # Show model info + integrity check
tomoe devices             # List audio input devices
tomoe config              # Print current config
```

## Default Hotkeys

| Hotkey | Action |
|--------|--------|
| `Super+Shift+S` | Toggle dictation (CLI + GUI) |
| `Super+Shift+X` | Toggle meeting recording (CLI + GUI) |

Configurable in `~/.config/tomoe/config.toml`.

## Configuration

```toml
# ~/.config/tomoe/config.toml

[hotkey]
binding = 'Super+Shift+S'
meeting_binding = 'Super+Shift+X'

[audio]
device = 'default'

[transcription]
gpu_enabled = true
model_path = '~/.local/share/tomoe/models'

[output]
auto_paste = true
clipboard = true
silence_timeout = 5.0

[meeting]
default_sources = 'both'
monitor_device = ''
speaker_threshold = 0.65
max_speech_duration = 30.0
min_silence_duration = 0.5
auto_save = true
```

## Architecture

```
tomoe-pc/
├── cmd/
│   ├── tomoe/              # CLI entry point
│   └── tomoe-gui/          # Wails GUI entry point
├── internal/
│   ├── audio/              # Audio capture, streaming, monitor sources
│   ├── backend/            # Wails Go backend (app, events, tray, hotkey)
│   ├── clipboard/          # Clipboard write + terminal-aware auto-paste
│   ├── config/             # TOML config
│   ├── daemon/             # CLI daemon orchestration
│   ├── gpu/                # GPU detection
│   ├── hotkey/             # Global hotkey (X11 key grabs)
│   ├── live/               # Live transcription coordinator
│   ├── models/             # Model download and management
│   ├── notify/             # Desktop notifications
│   ├── platform/           # Services aggregation layer
│   ├── session/            # Session storage, export, audio recording
│   ├── sigfix/             # ONNX Runtime signal handler fix
│   ├── speaker/            # Speaker embedding + clustering
│   └── transcribe/         # sherpa-onnx / Parakeet TDT integration
├── frontend/               # React + TypeScript + Vite
└── Makefile
```

### Inference Stack

```
Go binary → cgo → sherpa-onnx C API → ONNX Runtime (CUDA EP / CPU EP)
  → Parakeet TDT 0.6B v3 INT8 (encoder + decoder + joiner, 25 languages)
  → Silero VAD (~2MB)
  → 3D-Speaker embedding model (~25MB)
```

### Data Flow (Meeting Mode)

```
Mic Capturer → StreamCapturer → VAD → Transcribe → Segment{speaker:"You"}
                                                            │
Monitor Capturer → StreamCapturer → VAD → Embed → Cluster → Segment{speaker:"Person N"}
                                            │                       │
                                      Transcribe              EventsEmit
                                                                    │
                                                             React Frontend
```

## Data Paths

| Path | Purpose |
|------|---------|
| `~/.config/tomoe/config.toml` | Configuration |
| `~/.local/share/tomoe/models/` | ONNX models |
| `~/.local/share/tomoe/sessions/` | Saved sessions (JSON + MP3) |
| `~/.local/share/tomoe/lib/` | GPU libraries (if installed) |

## License

GPLv3 — see [LICENSE](LICENSE).
