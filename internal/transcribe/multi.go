package transcribe

import (
	"fmt"
	"strings"

	"github.com/sosuke-ai/tomoe-pc/internal/langid"
)

// MultiEngine implements Engine by detecting the language of each audio segment
// and routing to the appropriate per-language engine.
//
// For 2-language setups, when lang-id detects the default language, a dual-engine
// validation pass runs both engines and uses script detection to pick the correct
// result. This compensates for Whisper tiny's unreliable detection of some languages
// (e.g., Bengali) on short utterances.
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

// resolveEngine uses lang-id to pick the engine. Returns the engine and language code.
func (m *MultiEngine) resolveEngine(samples []float32) (Engine, string) {
	lang := m.langID.Detect(samples)
	detected := lang
	if lang == "" {
		lang = m.defaultLang
	}
	engine, ok := m.engines[lang]
	if !ok {
		fmt.Printf("lang-id: detected %q, no engine available, falling back to %q\n", lang, m.defaultLang)
		engine = m.engines[m.defaultLang]
		lang = m.defaultLang
	} else {
		fmt.Printf("lang-id: detected %q (raw: %q), routing to %s engine\n", lang, detected, lang)
	}
	return engine, lang
}

// alternateEngine returns the non-default engine and its language code.
// Only meaningful for 2-engine setups.
func (m *MultiEngine) alternateEngine() (Engine, string) {
	for lang, engine := range m.engines {
		if lang != m.defaultLang {
			return engine, lang
		}
	}
	return nil, ""
}

// transcribeWithValidation runs the default engine first, then validates with
// the alternate engine. If the alternate engine produces text in its expected
// script, it's preferred over the default (which is likely gibberish from
// misdetected audio).
func (m *MultiEngine) transcribeWithValidation(samples []float32, method string) (*Result, error) {
	altEngine, altLang := m.alternateEngine()
	if altEngine == nil {
		return nil, fmt.Errorf("no alternate engine available")
	}

	// Run the alternate (non-default) engine
	var altResult *Result
	var altErr error
	if method == "samples" {
		altResult, altErr = altEngine.TranscribeSamples(samples)
	} else {
		altResult, altErr = altEngine.TranscribeDirect(samples)
	}

	// If alternate engine produced text in the expected script, prefer it
	if altErr == nil && altResult != nil {
		altText := strings.TrimSpace(altResult.Text)
		if altText != "" && containsScript(altText, altLang) {
			fmt.Printf("lang-id: override %s → %s (script validation: %s text detected)\n",
				m.defaultLang, altLang, altLang)
			altResult.Language = altLang
			return altResult, nil
		}
	}

	// Alternate engine didn't produce expected script — use default engine
	var result *Result
	var err error
	if method == "samples" {
		result, err = m.engines[m.defaultLang].TranscribeSamples(samples)
	} else {
		result, err = m.engines[m.defaultLang].TranscribeDirect(samples)
	}
	if err != nil {
		return nil, err
	}
	result.Language = m.defaultLang
	fmt.Printf("lang-id: confirmed %s (alternate engine produced no %s script)\n",
		m.defaultLang, altLang)
	return result, nil
}

// TranscribeSamples detects language then transcribes with the appropriate engine.
func (m *MultiEngine) TranscribeSamples(samples []float32) (*Result, error) {
	engine, lang := m.resolveEngine(samples)

	// For 2-language setups: when lang-id says default, validate with dual-engine
	if lang == m.defaultLang && len(m.engines) == 2 {
		return m.transcribeWithValidation(samples, "samples")
	}

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

	// For 2-language setups: when lang-id says default, validate with dual-engine
	if lang == m.defaultLang && len(m.engines) == 2 {
		return m.transcribeWithValidation(samples, "direct")
	}

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

// scriptRanges maps language codes to their primary Unicode script ranges.
// Used to validate that an engine produced output in the expected writing system.
var scriptRanges = map[string][2]rune{
	"bn": {0x0980, 0x09FF}, // Bengali
	"hi": {0x0900, 0x097F}, // Devanagari (Hindi)
	"ta": {0x0B80, 0x0BFF}, // Tamil
	"te": {0x0C00, 0x0C7F}, // Telugu
	"gu": {0x0A80, 0x0AFF}, // Gujarati
	"kn": {0x0C80, 0x0CFF}, // Kannada
	"ml": {0x0D00, 0x0D7F}, // Malayalam
	"pa": {0x0A00, 0x0A7F}, // Gurmukhi (Punjabi)
	"or": {0x0B00, 0x0B7F}, // Odia
	"si": {0x0D80, 0x0DFF}, // Sinhala
	"th": {0x0E00, 0x0E7F}, // Thai
	"my": {0x1000, 0x109F}, // Myanmar
	"ka": {0x10A0, 0x10FF}, // Georgian
	"am": {0x1200, 0x137F}, // Ethiopic (Amharic)
	"km": {0x1780, 0x17FF}, // Khmer
	"lo": {0x0E80, 0x0EFF}, // Lao
	"ja": {0x3040, 0x309F}, // Hiragana (Japanese)
	"ko": {0xAC00, 0xD7AF}, // Hangul (Korean)
	"zh": {0x4E00, 0x9FFF}, // CJK Unified (Chinese)
	"ar": {0x0600, 0x06FF}, // Arabic
	"he": {0x0590, 0x05FF}, // Hebrew
}

// containsScript checks whether the text contains characters from the expected
// Unicode script range for the given language.
func containsScript(text string, lang string) bool {
	r, ok := scriptRanges[lang]
	if !ok {
		return false // no script range defined (e.g., Latin-script languages)
	}
	for _, ch := range text {
		if ch >= r[0] && ch <= r[1] {
			return true
		}
	}
	return false
}
