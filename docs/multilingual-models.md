# Multilingual Model Integration Guide

Tomoe supports multiple transcription models for different languages. Each VAD-segmented audio chunk is first language-detected, then routed to the appropriate model for transcription.

## Architecture Overview

```
Audio → VAD → Speech Segment → Language Detection (Whisper tiny) → Route
                                                                     │
                                        ┌────────────────────────────┤
                                        │                            │
                                  lang="en"                    lang="bn"
                                        │                            │
                               Parakeet TDT 0.6B v3         Bengali Zipformer
                               (OfflineRecognizer)           (OnlineRecognizer)
                                        │                            │
                                        └────────────┬───────────────┘
                                                      │
                                              Result{text, lang}
```

The `MultiEngine` implements the same `Engine` interface as individual engines, making it transparent to all callers (live pipeline, daemon, GUI backend).

## Current Models

| Model | Languages | Architecture | Size | API |
|-------|-----------|-------------|------|-----|
| Parakeet TDT 0.6B v3 INT8 | 25 (European) | NeMo Transducer | ~350MB | OfflineRecognizer |
| Bengali Zipformer (vosk) | Bengali | Zipformer2 Transducer | ~87MB | OnlineRecognizer |
| Whisper tiny INT8 | 99 (lang-id only) | Whisper encoder+decoder | ~98MB | SpokenLanguageIdentification |

## Adding a New Language

### Step 1: Find a Model

Browse the [sherpa-onnx model zoo](https://github.com/k2-fsa/sherpa-onnx/releases/tag/asr-models) for a model supporting your target language. Models come in two types:

- **Offline models** (OfflineRecognizer) — process entire audio segments at once. Used for Parakeet.
- **Online/streaming models** (OnlineRecognizer) — process audio in chunks. Used for Bengali Zipformer.

Both work for our use case since we transcribe complete VAD segments.

### Step 2: Add URL Constants

In `internal/models/urls.go`, add download URL and file constants:

```go
const (
    // Hindi Zipformer transducer
    HindiArchiveURL = "https://github.com/k2-fsa/sherpa-onnx/releases/download/asr-models/sherpa-onnx-streaming-zipformer-hi-vosk-YYYY-MM-DD.tar.bz2"
    HindiSubdir     = "sherpa-onnx-streaming-zipformer-hi-vosk-YYYY-MM-DD"
    hindiEncoderFile = "encoder.onnx"
    hindiDecoderFile = "decoder.onnx"
    hindiJoinerFile  = "joiner.onnx"
    hindiTokensFile  = "tokens.txt"
)
```

### Step 3: Add Readiness Check

In `internal/models/manager.go`, extend the `Status` struct:

```go
HindiReady       bool
HindiEncoderPath string
HindiDecoderPath string
HindiJoinerPath  string
HindiTokensPath  string
```

Update `Check()` to set paths and verify files exist.

### Step 4: Create Engine Wrapper

Create `internal/transcribe/hindi.go` following the pattern in `bengali.go`:

- For **streaming models** (zipformer2), use `OnlineRecognizer` with `OnlineTransducerModelConfig`
- For **offline models** (like Parakeet), use `OfflineRecognizer` with `OfflineTransducerModelConfig`

The engine must implement the `Engine` interface:
```go
type Engine interface {
    TranscribeSamples(samples []float32) (*Result, error)
    TranscribeDirect(samples []float32) (*Result, error)
    TranscribeFile(path string) (*Result, error)
    Close()
}
```

### Step 5: Register in Factory

In `internal/transcribe/factory.go`, add a case in the language loop:

```go
if lang == "hi" && status.HindiReady {
    hindi, err := NewHindiEngine(HindiConfig{...})
    if err == nil {
        engines["hi"] = hindi
    }
}
```

### Step 6: Update Config

Add `"hi"` to the languages list in `config.toml`:

```toml
[multilingual]
enabled = true
languages = ['en', 'bn', 'hi']
default_lang = 'en'
```

### Step 7: Download Models

Add download logic to `DownloadMultilingual()` in `manager.go`, or use:
```bash
tomoe model download --multilingual
```

## Supported Model Architectures

### Transducer (Parakeet, Zipformer)

Transducer models have three components: encoder, decoder, joiner. They're the most common architecture in sherpa-onnx and support hotword boosting via `modified_beam_search`.

- **NeMo Transducer** (Parakeet): `ModelType: "nemo_transducer"`, uses `OfflineRecognizer`
- **Zipformer2 Transducer**: `ModelType: "zipformer2"`, uses `OnlineRecognizer` for streaming models

### CTC

CTC models have a single model file. Some sherpa-onnx models use CTC (e.g., `OnlineZipformer2CtcModelConfig`). These can be wrapped with `OnlineRecognizer` using the CTC config.

### Whisper

Whisper models have encoder + decoder. Used primarily for language identification in Tomoe, but can also be used for transcription via `OfflineRecognizer` with `OfflineWhisperModelConfig`.

## Language Detection

Language detection uses Whisper tiny (multilingual variant, ~98MB). It supports 99 languages. Performance:

- **Latency**: ~100-150ms per 2-3 second segment on CPU
- **Accuracy**: High for segments >1 second; may be unreliable for very short (<1s) segments

For short segments, the `MultiEngine` falls back to the default language.

## Hotword Boosting

Hotword boosting improves recognition of specific words/phrases that are often misrecognized. It works with transducer models using `modified_beam_search` decoding.

### Configuration

```toml
[transcription]
decoding_method = 'modified_beam_search'
hotwords_file = '/home/user/.config/tomoe/hotwords.txt'
hotwords_score = 1.5
max_active_paths = 4
```

### Hotwords File Format

One word or phrase per line:

```
claude
haiku
sonnet
opus
anthropic
kubernetes
terraform
```

### Tuning

- **hotwords_score**: Higher values boost hotwords more aggressively. Start with 1.5 and adjust.
- **max_active_paths**: Number of beam search paths. Higher values are more accurate but slower. Default: 4.
- `modified_beam_search` is slightly slower than `greedy_search` but necessary for hotword support.

## Performance Considerations

- **Model loading**: Each model takes 1-3 seconds to load. All models load at startup.
- **Memory**: ~540MB total for Parakeet + Whisper tiny + Bengali (acceptable for desktop).
- **Per-segment overhead**: ~100-150ms for language detection + normal transcription time.
- **GPU**: Language detection always runs on CPU (fast enough). Parakeet can use CUDA. Bengali runs on CPU.
