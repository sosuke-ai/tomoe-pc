package transcribe

import (
	"fmt"

	"github.com/sosuke-ai/tomoe-pc/internal/config"
	"github.com/sosuke-ai/tomoe-pc/internal/langid"
	"github.com/sosuke-ai/tomoe-pc/internal/models"
)

// NewEngineFromConfig creates an Engine based on config and model status.
// If multilingual is enabled and models are available, returns a MultiEngine
// that detects language per-segment and routes to the appropriate model.
// Otherwise returns a plain Parakeet engine.
func NewEngineFromConfig(cfg Config, status *models.Status, multiCfg *config.MultilingualConfig) (Engine, error) {
	// Create base Parakeet engine
	parakeet, err := NewEngine(cfg)
	if err != nil {
		return nil, err
	}

	// If multilingual disabled or not configured, return Parakeet directly
	if multiCfg == nil || !multiCfg.Enabled {
		return parakeet, nil
	}

	// Build per-language engine map
	defaultLang := multiCfg.DefaultLang
	if defaultLang == "" {
		defaultLang = "en"
	}

	// Check if lang-id models are available
	if !status.LangIDReady {
		fmt.Println("Warning: multilingual enabled but lang-id models not downloaded, using Parakeet only")
		return parakeet, nil
	}

	// Create language identifier constrained to configured languages
	identifier, err := langid.NewIdentifier(status.LangIDEncoderPath, status.LangIDDecoderPath, multiCfg.Languages, defaultLang)
	if err != nil {
		fmt.Printf("Warning: failed to create language identifier: %v, using Parakeet only\n", err)
		return parakeet, nil
	}

	engines := map[string]Engine{}

	// Parakeet handles all Parakeet-supported languages (mapped to default)
	engines[defaultLang] = parakeet

	// Create Bengali engine if configured and available
	for _, lang := range multiCfg.Languages {
		if lang == "bn" && status.BengaliReady {
			bengali, err := NewBengaliEngine(BengaliConfig{
				EncoderPath: status.BengaliEncoderPath,
				DecoderPath: status.BengaliDecoderPath,
				JoinerPath:  status.BengaliJoinerPath,
				TokensPath:  status.BengaliTokensPath,
			})
			if err != nil {
				fmt.Printf("Warning: failed to create Bengali engine: %v\n", err)
				continue
			}
			engines["bn"] = bengali
			fmt.Println("Multilingual: Bengali Zipformer engine loaded")
		}
	}

	// If we only have the default engine, no point in using MultiEngine
	if len(engines) <= 1 {
		identifier.Close()
		fmt.Println("Warning: multilingual enabled but no additional language engines available")
		return parakeet, nil
	}

	multi, err := NewMultiEngine(MultiEngineConfig{
		LangID:      identifier,
		Engines:     engines,
		DefaultLang: defaultLang,
	})
	if err != nil {
		identifier.Close()
		for lang, eng := range engines {
			if lang != defaultLang {
				eng.Close()
			}
		}
		return parakeet, nil
	}

	fmt.Printf("Multilingual: active with %d engines (default: %s)\n", len(engines), defaultLang)
	return multi, nil
}
