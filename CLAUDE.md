# Tomoe PC — Project Instructions

## Project Overview

Local-first speech-to-text desktop application for Linux. Captures microphone audio, transcribes locally using NVIDIA Parakeet TDT 0.6B v3 (INT8, 25 European languages) via sherpa-onnx, and delivers text to clipboard. Written in Go with cgo dependencies.

- **License:** GPLv3
- **Language:** Go 1.22+
- **Target OS:** Ubuntu Linux 24.04+ (X11 primary, Wayland best-effort)
- **Audio:** PipeWire (with PulseAudio compat layer)
- **GPU:** NVIDIA CUDA (recommended), CPU fallback automatic

## Project Structure

```
tomoe-pc/
├── cmd/tomoe/                  # Main entry point
├── internal/
│   ├── audio/                  # Audio capture (gen2brain/malgo)
│   ├── transcribe/             # sherpa-onnx / Parakeet TDT integration
│   ├── clipboard/              # Clipboard write (atotto/clipboard)
│   ├── hotkey/                 # Global hotkey (golang-design/hotkey)
│   ├── gpu/                    # GPU detection, ONNX Runtime EP selection
│   └── models/                 # Model download and management
├── frontend/                   # Wails v2 frontend (Phase 2)
├── docs/                       # Tech specs and documentation
├── go.mod
├── go.sum
└── Makefile
```

## Tech Spec

The authoritative tech spec is at `docs/speech-to-text-tech-brief.md`. Always validate implementation plans and completed work against it.

## Build & Run

```bash
make build            # Build CLI binary (CPU)
make build-cuda       # Build with CUDA support
make test             # Run tests
make download-model   # Download Parakeet TDT v3 INT8 model + Silero VAD
make install          # Install to /usr/local/bin
```

### Build Requirements

- Go 1.22+
- C/C++ toolchain (gcc/g++ for cgo)
- ONNX Runtime shared library (≥1.17.0)
- sherpa-onnx C API headers and library

## Key Dependencies

| Package | Purpose | cgo |
|---|---|---|
| `k2-fsa/sherpa-onnx` | Transcription engine (ONNX Runtime + Parakeet TDT) | Yes |
| `gen2brain/malgo` | Audio capture (miniaudio bindings) | Yes |
| `golang-design/hotkey` | Global hotkey registration | Yes (X11) |
| `atotto/clipboard` | Clipboard write | No |
| `pelletier/go-toml` | Config file parsing (TOML) | No |
| `schollz/progressbar` | CLI progress bars | No |

## Inference Stack

```
Go binary → cgo → sherpa-onnx C API → ONNX Runtime (CUDA EP / CPU EP)
  → Parakeet TDT 0.6B v3 INT8 (encoder + decoder + joiner, 25 languages)
  → Silero VAD (~2MB)
```

## Configuration Paths

- Config: `~/.config/tomoe/config.toml`
- Models: `~/.local/share/tomoe/models/`
- Sessions: `~/.local/share/tomoe/sessions/` (Phase 2)

## Development Phases

- **Phase 1 (current):** CLI dictation mode — hotkey toggle, mic capture, local transcription, clipboard output, auto-init
- **Phase 2:** Meeting transcription GUI via Wails v2 — system audio loopback, live transcript, session management

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
- Model download: ~350MB (INT8 archive) + ~2MB (Silero VAD)
- VRAM during inference: ~1–1.5GB (INT8)
