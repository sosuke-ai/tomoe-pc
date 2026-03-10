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

# Audio format conversion (FLAC, OGG → WAV for transcription)
sudo apt install ffmpeg
```

### One-liner

```bash
sudo apt install --allow-downgrades build-essential pkg-config \
  libx11-dev libxtst-dev libxkbcommon-dev \
  libasound-dev portaudio19-dev libportaudio2 libpulse-dev \
  xclip xdotool wl-clipboard wtype libnotify-bin ffmpeg
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

## sherpa-onnx (Transcription Engine)

The Go bindings (`github.com/k2-fsa/sherpa-onnx-go`) bundle pre-built shared libraries for Linux — no manual installation of ONNX Runtime or sherpa-onnx C API is needed. Dependencies are pulled automatically via `go get`.
