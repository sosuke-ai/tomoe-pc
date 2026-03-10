package transcribe

// Result holds the transcription output.
type Result struct {
	Text       string
	Tokens     []string
	Timestamps []float32
	Duration   float64 // audio duration in seconds
	Language   string  // detected language code
}

// Engine wraps speech recognition capabilities.
type Engine interface {
	// TranscribeSamples transcribes raw float32 PCM audio at 16kHz.
	// Uses VAD segmentation if configured.
	TranscribeSamples(samples []float32) (*Result, error)

	// TranscribeDirect transcribes pre-segmented audio without VAD.
	// Use this when audio has already been segmented (e.g., by the live pipeline).
	TranscribeDirect(samples []float32) (*Result, error)

	// TranscribeFile transcribes an audio file (WAV, FLAC, or OGG).
	TranscribeFile(path string) (*Result, error)

	// Close releases all native resources.
	Close()
}

// Config holds paths and settings for creating an Engine.
type Config struct {
	EncoderPath string
	DecoderPath string
	JoinerPath  string
	TokensPath  string
	VADPath     string
	UseGPU      bool
	NumThreads  int
}
