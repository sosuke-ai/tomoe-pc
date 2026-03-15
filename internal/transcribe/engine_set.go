package transcribe

import (
	"fmt"
	"sort"
)

// EngineSet holds a set of per-language transcription engines.
// Callers must explicitly pick a language — EngineSet does NOT implement Engine.
type EngineSet struct {
	engines     map[string]Engine
	defaultLang string
}

// NewEngineSet creates an engine set from a language-to-engine map.
func NewEngineSet(engines map[string]Engine, defaultLang string) (*EngineSet, error) {
	if len(engines) == 0 {
		return nil, fmt.Errorf("at least one engine is required")
	}
	if defaultLang == "" {
		defaultLang = "en"
	}
	if _, ok := engines[defaultLang]; !ok {
		return nil, fmt.Errorf("no engine for default language %q", defaultLang)
	}
	return &EngineSet{
		engines:     engines,
		defaultLang: defaultLang,
	}, nil
}

// Get returns the engine for the given language, falling back to the default
// engine if the language is not available.
func (es *EngineSet) Get(lang string) Engine {
	if eng, ok := es.engines[lang]; ok {
		return eng
	}
	return es.engines[es.defaultLang]
}

// Default returns the default language engine.
func (es *EngineSet) Default() Engine {
	return es.engines[es.defaultLang]
}

// DefaultLang returns the default language code.
func (es *EngineSet) DefaultLang() string {
	return es.defaultLang
}

// Languages returns a sorted list of available language codes.
func (es *EngineSet) Languages() []string {
	langs := make([]string, 0, len(es.engines))
	for lang := range es.engines {
		langs = append(langs, lang)
	}
	sort.Strings(langs)
	return langs
}

// Close releases all engines.
func (es *EngineSet) Close() {
	for _, eng := range es.engines {
		eng.Close()
	}
}
