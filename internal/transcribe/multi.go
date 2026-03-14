package transcribe

import (
	"fmt"

	"github.com/sosuke-ai/tomoe-pc/internal/langid"
)

// MultiEngine implements Engine by detecting the language of each audio segment
// and routing to the appropriate per-language engine.
type MultiEngine struct {
	langID      *langid.Identifier
	engines     map[string]Engine // language code → engine
	defaultLang string
}

// MultiEngineConfig holds configuration for creating a MultiEngine.
type MultiEngineConfig struct {
	LangID      *langid.Identifier
	Engines     map[string]Engine
	DefaultLang string
}

// NewMultiEngine creates a multi-engine that routes audio to per-language engines.
func NewMultiEngine(cfg MultiEngineConfig) (*MultiEngine, error) {
	if cfg.LangID == nil {
		return nil, fmt.Errorf("language identifier is required")
	}
	if len(cfg.Engines) == 0 {
		return nil, fmt.Errorf("at least one engine is required")
	}
	if cfg.DefaultLang == "" {
		cfg.DefaultLang = "en"
	}
	if _, ok := cfg.Engines[cfg.DefaultLang]; !ok {
		return nil, fmt.Errorf("no engine for default language %q", cfg.DefaultLang)
	}
	return &MultiEngine{
		langID:      cfg.LangID,
		engines:     cfg.Engines,
		defaultLang: cfg.DefaultLang,
	}, nil
}

func (m *MultiEngine) resolveEngine(samples []float32) (Engine, string) {
	lang := m.langID.Detect(samples)
	if lang == "" {
		lang = m.defaultLang
	}
	engine, ok := m.engines[lang]
	if !ok {
		engine = m.engines[m.defaultLang]
		lang = m.defaultLang
	}
	return engine, lang
}

// TranscribeSamples detects language then transcribes with the appropriate engine.
func (m *MultiEngine) TranscribeSamples(samples []float32) (*Result, error) {
	engine, lang := m.resolveEngine(samples)
	result, err := engine.TranscribeSamples(samples)
	if err != nil {
		return nil, err
	}
	result.Language = lang
	return result, nil
}

// TranscribeDirect detects language then transcribes pre-segmented audio.
func (m *MultiEngine) TranscribeDirect(samples []float32) (*Result, error) {
	engine, lang := m.resolveEngine(samples)
	result, err := engine.TranscribeDirect(samples)
	if err != nil {
		return nil, err
	}
	result.Language = lang
	return result, nil
}

// TranscribeFile transcribes a file using the default engine.
// Language detection is not applied for file transcription since
// the file may contain multiple languages.
func (m *MultiEngine) TranscribeFile(path string) (*Result, error) {
	return m.engines[m.defaultLang].TranscribeFile(path)
}

// Close releases all engines and the language identifier.
func (m *MultiEngine) Close() {
	for _, engine := range m.engines {
		engine.Close()
	}
	if m.langID != nil {
		m.langID.Close()
		m.langID = nil
	}
}
