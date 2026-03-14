package langid

import (
	"fmt"

	sherpa "github.com/k2-fsa/sherpa-onnx-go/sherpa_onnx"

	"github.com/sosuke-ai/tomoe-pc/internal/sigfix"
)

const sampleRate = 16000

// Identifier detects the spoken language of audio samples using Whisper tiny.
type Identifier struct {
	slid *sherpa.SpokenLanguageIdentification
}

// NewIdentifier creates a language identifier from Whisper tiny encoder/decoder models.
func NewIdentifier(encoderPath, decoderPath string) (*Identifier, error) {
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

	return &Identifier{slid: slid}, nil
}

// Detect returns the ISO 639-1 language code for the given audio samples (16kHz mono float32).
// Returns codes like "en", "bn", "de", "fr", etc.
func (id *Identifier) Detect(samples []float32) string {
	if len(samples) == 0 {
		return ""
	}

	stream := id.slid.CreateStream()
	if stream == nil {
		return ""
	}
	defer sherpa.DeleteOfflineStream(stream)

	stream.AcceptWaveform(sampleRate, samples)

	result := id.slid.Compute(stream)
	if result == nil {
		return ""
	}
	return result.Lang
}

// Close releases the underlying native resources.
func (id *Identifier) Close() {
	if id.slid != nil {
		sherpa.DeleteSpokenLanguageIdentification(id.slid)
		id.slid = nil
	}
}
