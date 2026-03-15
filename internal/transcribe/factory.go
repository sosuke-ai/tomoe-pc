package transcribe

import (
	"fmt"

	"github.com/sosuke-ai/tomoe-pc/internal/config"
	"github.com/sosuke-ai/tomoe-pc/internal/models"
)

// NewEngineSetFromConfig creates an EngineSet based on config and model status.
// If multilingual is enabled and models are available, returns an EngineSet with
// per-language engines. Otherwise returns an EngineSet with just the default
// (Parakeet) engine.
func NewEngineSetFromConfig(cfg Config, status *models.Status, multiCfg *config.MultilingualConfig) (*EngineSet, error) {
	// Create base Parakeet engine
	parakeet, err := NewEngine(cfg)
	if err != nil {
		return nil, err
	}

	defaultLang := "en"
	if multiCfg != nil && multiCfg.DefaultLang != "" {
		defaultLang = multiCfg.DefaultLang
	}

	engines := map[string]Engine{defaultLang: parakeet}

	// If multilingual enabled, create additional engines
	if multiCfg != nil && multiCfg.Enabled {
		for _, lang := range multiCfg.Languages {
			if lang == defaultLang {
				continue // already have Parakeet
			}
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
	}

	es, err := NewEngineSet(engines, defaultLang)
	if err != nil {
		parakeet.Close()
		return nil, err
	}

	if len(engines) > 1 {
		fmt.Printf("Multilingual: %d engines available (default: %s)\n", len(engines), defaultLang)
	}

	return es, nil
}
