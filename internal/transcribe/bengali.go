package transcribe

import (
	"fmt"
	"os"
	"strings"

	sherpa "github.com/k2-fsa/sherpa-onnx-go/sherpa_onnx"

	"github.com/sosuke-ai/tomoe-pc/internal/sigfix"
)

// BengaliConfig holds paths for the Bengali Zipformer streaming model.
type BengaliConfig struct {
	EncoderPath string
	DecoderPath string
	JoinerPath  string
	TokensPath  string
	NumThreads  int
}

// bengaliEngine implements Engine using sherpa-onnx online (streaming) recognition
// for the Bengali Zipformer transducer model.
type bengaliEngine struct {
	recognizer *sherpa.OnlineRecognizer
}

// NewBengaliEngine creates a transcription engine for Bengali using the
// streaming Zipformer transducer model.
func NewBengaliEngine(cfg BengaliConfig) (Engine, error) {
	numThreads := cfg.NumThreads
	if numThreads <= 0 {
		numThreads = 2
	}

	config := &sherpa.OnlineRecognizerConfig{
		FeatConfig: sherpa.FeatureConfig{
			SampleRate: sampleRate,
			FeatureDim: 80,
		},
		ModelConfig: sherpa.OnlineModelConfig{
			Transducer: sherpa.OnlineTransducerModelConfig{
				Encoder: cfg.EncoderPath,
				Decoder: cfg.DecoderPath,
				Joiner:  cfg.JoinerPath,
			},
			Tokens:     cfg.TokensPath,
			NumThreads: numThreads,
			Provider:   "cpu",
			ModelType:  "zipformer2",
		},
		DecodingMethod: "greedy_search",
		EnableEndpoint: 0,
	}

	recognizer := sherpa.NewOnlineRecognizer(config)
	if recognizer == nil {
		return nil, fmt.Errorf("failed to create Bengali online recognizer (check model paths)")
	}
	sigfix.AfterSherpa()

	return &bengaliEngine{recognizer: recognizer}, nil
}

// TranscribeSamples transcribes audio using the streaming recognizer.
// For pre-segmented audio (our use case), we feed all samples then decode.
func (e *bengaliEngine) TranscribeSamples(samples []float32) (*Result, error) {
	return e.TranscribeDirect(samples)
}

// TranscribeDirect transcribes pre-segmented audio.
func (e *bengaliEngine) TranscribeDirect(samples []float32) (*Result, error) {
	if len(samples) == 0 {
		return &Result{}, nil
	}

	duration := float64(len(samples)) / float64(sampleRate)

	stream := sherpa.NewOnlineStream(e.recognizer)
	if stream == nil {
		return nil, fmt.Errorf("failed to create online stream")
	}
	defer sherpa.DeleteOnlineStream(stream)

	stream.AcceptWaveform(sampleRate, samples)
	stream.InputFinished()

	// Decode until no more frames are ready
	for e.recognizer.IsReady(stream) {
		e.recognizer.Decode(stream)
	}

	result := e.recognizer.GetResult(stream)
	if result == nil {
		return &Result{Duration: duration, Language: "bn"}, nil
	}

	return &Result{
		Text:     strings.TrimSpace(result.Text),
		Duration: duration,
		Language: "bn",
	}, nil
}

// TranscribeFile transcribes an audio file.
func (e *bengaliEngine) TranscribeFile(path string) (*Result, error) {
	if !IsSupportedFormat(path) {
		return nil, fmt.Errorf("unsupported audio format: %s", path)
	}

	wavPath := path
	if needsConversion(path) {
		converted, err := convertToWAV(path)
		if err != nil {
			return nil, fmt.Errorf("converting %s to WAV: %w", path, err)
		}
		defer func() { _ = os.Remove(converted) }()
		wavPath = converted
	}

	wave := sherpa.ReadWave(wavPath)
	if wave == nil {
		return nil, fmt.Errorf("failed to read audio file: %s", wavPath)
	}

	return e.TranscribeSamples(wave.Samples)
}

// Close releases all native resources.
func (e *bengaliEngine) Close() {
	if e.recognizer != nil {
		sherpa.DeleteOnlineRecognizer(e.recognizer)
		e.recognizer = nil
	}
}
