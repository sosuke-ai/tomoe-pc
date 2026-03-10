# Developer Setup — Ubuntu 24.04+

## System Dependencies

```bash
# Build toolchain (Go cgo requires C/C++ compiler)
sudo apt install build-essential pkg-config

# X11 development headers (global hotkey via golang-design/hotkey)
sudo apt install libx11-dev libxtst-dev libxkbcommon-dev

# Audio development libraries (portaudio for mic capture)
sudo apt install libasound-dev portaudio19-dev libportaudio2 libpulse-dev

# Clipboard and auto-paste tools
sudo apt install xclip xdotool

# Wayland clipboard/paste equivalents (best-effort support)
sudo apt install wl-clipboard wtype

# Desktop notifications
sudo apt install libnotify-bin

# Audio format conversion (FLAC, OGG → WAV for transcription; WAV → MP3 for session recording)
sudo apt install ffmpeg

# GUI dependencies (Wails v2 + system tray)
sudo apt install libwebkit2gtk-4.1-dev libappindicator3-dev libgtk-3-dev

# Node.js 18+ (React frontend build)
sudo apt install nodejs npm
```

### One-liner

```bash
make dev-deps
```

Or manually:

```bash
sudo apt install build-essential pkg-config \
  libx11-dev libxtst-dev libxkbcommon-dev \
  libasound-dev portaudio19-dev libportaudio2 libpulse-dev \
  xclip xdotool wl-clipboard wtype libnotify-bin ffmpeg \
  libwebkit2gtk-4.1-dev libappindicator3-dev libgtk-3-dev \
  nodejs npm
```

## Go

Go 1.22+ is required. Install via snap or from https://go.dev/dl/:

```bash
sudo snap install go --classic
```

Verify cgo is enabled (should be `1` by default on Linux):

```bash
go env CGO_ENABLED
```

## Go Development Tools

```bash
make dev-tools
```

This installs:
- `golangci-lint` v2.11.3 — linter
- `goimports` — import organizer
- `wails` — GUI framework CLI

## sherpa-onnx (Transcription Engine)

The Go bindings (`github.com/k2-fsa/sherpa-onnx-go`) bundle pre-built shared libraries for Linux — no manual installation of ONNX Runtime or sherpa-onnx C API is needed. Dependencies are pulled automatically via `go get`.

## GPU Acceleration (NVIDIA CUDA)

GPU acceleration requires an NVIDIA GPU with the proprietary driver installed. The driver provides CUDA support — verify with:

```bash
nvidia-smi   # Should show "CUDA Version: 12.x" or higher
```

### Install GPU support

```bash
make install-gpu
```

This will:

1. **Install CUDA 12.8 toolkit + cuDNN 9** via the NVIDIA apt repository (requires `sudo`)
2. **Download sherpa-onnx GPU provider libraries** (~235MB download) to `~/.local/share/tomoe/lib/`

The GPU provider `.so` files are loaded at runtime by ONNX Runtime when `gpu_enabled = true` in `~/.config/tomoe/config.toml`. The tomoe binary prepends `~/.local/share/tomoe/lib/` to `LD_LIBRARY_PATH` automatically.

### Prerequisites

- NVIDIA driver 525+ (check with `nvidia-smi`)
- Ubuntu 24.04+ (the CUDA apt repo targets this release)
- ~3GB disk space (CUDA toolkit + cuDNN + GPU provider libs)

### Verify GPU is working

After `make install-gpu`, ensure your config has:

```toml
[transcription]
gpu_enabled = true
```

Then run `tomoe status` — it should show GPU info and the transcription engine using CUDA EP.

If you see `Fallback to cpu!` in the log output, the CUDA libraries are not being found. Check:

```bash
ls ~/.local/share/tomoe/lib/libonnxruntime_providers_cuda.so
ldconfig -p | grep libcublas
```

## Build & Run

```bash
make build          # Build CLI + GUI (if webkit2gtk available)
make test           # Run tests
make install        # Install to $GOPATH/bin
make download-model # Download transcription models (~350MB)
make install-gpu    # Install CUDA toolkit + GPU provider libs
```

### Development mode (GUI with hot-reload)

```bash
make dev-gui
```

This runs `wails dev` which rebuilds the frontend and backend on file changes.

## Project Layout

| Directory | Purpose |
|-----------|---------|
| `cmd/tomoe/` | CLI entry point |
| `cmd/tomoe-gui/` | Wails GUI entry point |
| `internal/audio/` | Audio capture + monitor source detection |
| `internal/backend/` | Wails backend (bindings, events, tray, hotkey) |
| `internal/clipboard/` | Clipboard write |
| `internal/config/` | TOML config |
| `internal/daemon/` | CLI daemon (hotkey → capture → transcribe → clipboard) |
| `internal/gpu/` | GPU detection |
| `internal/hotkey/` | Global hotkey registration |
| `internal/live/` | Live transcription coordinator |
| `internal/models/` | Model download and verification |
| `internal/notify/` | Desktop notifications |
| `internal/platform/` | Platform services aggregation |
| `internal/session/` | Session storage and export |
| `internal/speaker/` | Speaker embedding + clustering |
| `internal/transcribe/` | sherpa-onnx transcription engine |
| `frontend/` | React + TypeScript + Vite frontend |
