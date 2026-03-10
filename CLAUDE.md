# Tomoe PC — Project Instructions

## Project Overview

Local-first speech-to-text desktop application for Linux. Captures microphone audio and system audio, transcribes locally using NVIDIA Parakeet TDT 0.6B v3 (INT8, 25 European languages) via sherpa-onnx, and delivers text to clipboard or a live GUI transcript. Written in Go with cgo dependencies.

- **License:** GPLv3
- **Language:** Go 1.22+
- **Target OS:** Ubuntu Linux 24.04+ (X11 primary, Wayland best-effort)
- **Audio:** PipeWire (with PulseAudio compat layer)
- **GPU:** NVIDIA CUDA (recommended), CPU fallback automatic
- **GUI Framework:** Wails v2 (Phase 2)

## Project Structure

```
tomoe-pc/
├── cmd/
│   ├── tomoe/                     # CLI entry point (Phase 1)
│   └── tomoe-gui/                 # Wails GUI entry point (Phase 2)
├── internal/
│   ├── audio/                     # Audio capture, streaming, monitor sources
│   ├── backend/                   # Wails Go backend (app, events, tray, hotkey)
│   ├── clipboard/                 # Clipboard write (atotto/clipboard)
│   ├── config/                    # TOML config (Phase 1 + Phase 2 meeting config)
│   ├── daemon/                    # CLI daemon orchestration
│   ├── gpu/                       # GPU detection, ONNX Runtime EP selection
│   ├── hotkey/                    # Global hotkey (golang-design/hotkey)
│   ├── live/                      # Live transcription coordinator (Phase 2)
│   ├── models/                    # Model download and management
│   ├── notify/                    # Desktop notifications
│   ├── platform/                  # Services aggregation layer
│   ├── session/                   # Session data model, storage, export, audio
│   ├── speaker/                   # Speaker embedding + clustering (Phase 2)
│   └── transcribe/                # sherpa-onnx / Parakeet TDT integration
├── frontend/                      # React + TypeScript + Vite frontend
│   ├── src/
│   │   ├── components/            # React components
│   │   ├── hooks/                 # Custom React hooks
│   │   ├── types.ts               # TypeScript type definitions
│   │   ├── App.tsx                # Root component
│   │   └── main.tsx               # Entry point
│   └── wailsjs/                   # Wails runtime bindings
├── docs/                          # Tech specs and documentation
├── wails.json                     # Wails project configuration
├── go.mod
├── go.sum
└── Makefile
```

## Tech Spec

The authoritative tech spec is at `docs/speech-to-text-tech-brief.md`. Always validate implementation plans and completed work against it.

## Build & Run

```bash
make build            # Build CLI + GUI (if webkit2gtk available)
make build-gui        # Build GUI binary only
make build-cuda       # Build with CUDA support
make test             # Run tests
make dev-gui          # Wails dev mode with hot-reload
make download-model   # Download Parakeet TDT v3 INT8 + Silero VAD + Speaker Embedding
make install          # Install to GOPATH/bin
```

### Build Requirements

- Go 1.22+
- C/C++ toolchain (gcc/g++ for cgo)
- ONNX Runtime shared library (≥1.17.0)
- sherpa-onnx C API headers and library
- Node.js 18+ (for frontend)
- `libwebkit2gtk-4.1-dev` (for GUI)

## Key Dependencies

| Package | Purpose | cgo |
|---|---|---|
| `k2-fsa/sherpa-onnx` | Transcription engine (ONNX Runtime + Parakeet TDT) | Yes |
| `gen2brain/malgo` | Audio capture (miniaudio bindings) | Yes |
| `golang-design/hotkey` | Global hotkey registration | Yes (X11) |
| `wailsapp/wails/v2` | Desktop GUI framework | Yes |
| `fyne.io/systray` | System tray (AppIndicator3 on Linux) | Yes |
| `atotto/clipboard` | Clipboard write | No |
| `pelletier/go-toml` | Config file parsing (TOML) | No |
| `google/uuid` | Session IDs | No |
| `schollz/progressbar` | CLI progress bars | No |

## Inference Stack

```
Go binary → cgo → sherpa-onnx C API → ONNX Runtime (CUDA EP / CPU EP)
  → Parakeet TDT 0.6B v3 INT8 (encoder + decoder + joiner, 25 languages)
  → Silero VAD (~2MB)
  → 3D-Speaker embedding model (~25MB) [Phase 2]
```

## Configuration Paths

- Config: `~/.config/tomoe/config.toml`
- Models: `~/.local/share/tomoe/models/`
- Sessions: `~/.local/share/tomoe/sessions/`

## Development Phases

- **Phase 1:** CLI dictation mode — hotkey toggle, mic capture, local transcription, clipboard output, auto-init
- **Phase 2 (current):** Meeting transcription GUI via Wails v2 — system audio loopback, live transcript, speaker ID, session management, export

## Coding Conventions

- Use `internal/` for all non-main packages — nothing is exported outside the module
- Platform-specific code uses `_linux.go` build tag suffix
- Audio format: 16kHz mono PCM float32 (Parakeet TDT native input)
- 25 European languages with automatic language detection
- Model quantization: INT8 (v3)
- Config format: TOML via `pelletier/go-toml`

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

## Non-Functional Targets

- Transcription latency (30s clip): <1s GPU, <5s CPU
- Idle daemon memory: <50MB RSS
- Binary size: <30MB (excluding model + ONNX Runtime .so)
- Model download: ~350MB (INT8 archive) + ~2MB (Silero VAD) + ~25MB (Speaker Embedding)
- VRAM during inference: ~1–1.5GB (INT8)
