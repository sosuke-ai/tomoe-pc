package langid

import (
	"fmt"

	sherpa "github.com/k2-fsa/sherpa-onnx-go/sherpa_onnx"

	"github.com/sosuke-ai/tomoe-pc/internal/sigfix"
)

const sampleRate = 16000

// Identifier detects the spoken language of audio samples using Whisper tiny.
// When AllowedLangs is set, only those languages are returned; any other
// detection result is mapped to DefaultLang.
type Identifier struct {
	slid        *sherpa.SpokenLanguageIdentification
	allowed     map[string]bool // nil = accept all
	defaultLang string
}

// NewIdentifier creates a language identifier from Whisper tiny encoder/decoder models.
// allowedLangs constrains detection to only those languages (e.g. ["en", "bn"]).
// If empty, all Whisper-supported languages are accepted.
// defaultLang is returned when the detected language is not in the allowed set.
func NewIdentifier(encoderPath, decoderPath string, allowedLangs []string, defaultLang string) (*Identifier, error) {
	config := &sherpa.SpokenLanguageIdentificationConfig{
		Whisper: sherpa.SpokenLanguageIdentificationWhisperConfig{
			Encoder:      encoderPath,
			Decoder:      decoderPath,
			TailPaddings: -1,
		},
		NumThreads: 2,
		Provider:   "cpu",
	}

	slid := sherpa.NewSpokenLanguageIdentification(config)
	if slid == nil {
		return nil, fmt.Errorf("failed to create language identifier (check model paths: %s, %s)", encoderPath, decoderPath)
	}
	sigfix.AfterSherpa()

	var allowed map[string]bool
	if len(allowedLangs) > 0 {
		allowed = make(map[string]bool, len(allowedLangs))
		for _, lang := range allowedLangs {
			allowed[lang] = true
		}
	}

	if defaultLang == "" {
		defaultLang = "en"
	}

	return &Identifier{
		slid:        slid,
		allowed:     allowed,
		defaultLang: defaultLang,
	}, nil
}

// Detect returns the ISO 639-1 language code for the given audio samples (16kHz mono float32).
// If the detected language is not in the allowed set, returns the default language.
func (id *Identifier) Detect(samples []float32) string {
	if len(samples) == 0 {
		return id.defaultLang
	}

	stream := id.slid.CreateStream()
	if stream == nil {
		return id.defaultLang
	}
	defer sherpa.DeleteOfflineStream(stream)

	stream.AcceptWaveform(sampleRate, samples)

	result := id.slid.Compute(stream)
	if result == nil {
		return id.defaultLang
	}

	lang := result.Lang
	if lang == "" {
		return id.defaultLang
	}

	// If an allowed set is configured, reject languages not in it
	if id.allowed != nil && !id.allowed[lang] {
		return id.defaultLang
	}

	return lang
}

// Close releases the underlying native resources.
func (id *Identifier) Close() {
	if id.slid != nil {
		sherpa.DeleteSpokenLanguageIdentification(id.slid)
		id.slid = nil
	}
}
