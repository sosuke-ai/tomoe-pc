package platform

import (
	"fmt"

	"github.com/sosuke-ai/tomoe-pc/internal/audio"
	"github.com/sosuke-ai/tomoe-pc/internal/clipboard"
	"github.com/sosuke-ai/tomoe-pc/internal/config"
	"github.com/sosuke-ai/tomoe-pc/internal/hotkey"
	"github.com/sosuke-ai/tomoe-pc/internal/notify"
)

// Services aggregates all platform-dependent services.
type Services struct {
	Notifier  notify.Notifier
	Hotkey    hotkey.Listener
	Clipboard clipboard.Writer
	Audio     audio.Capturer
}

// New constructs all platform services from config.
func New(cfg *config.Config) (*Services, error) {
	notifier := notify.NewNotifier()

	hk, err := hotkey.NewListener(cfg.Hotkey.Binding)
	if err != nil {
		return nil, fmt.Errorf("creating hotkey listener: %w", err)
	}

	clip := clipboard.NewWriter()

	capturer, err := audio.NewCapturer(cfg.Audio.Device, audio.Input)
	if err != nil {
		return nil, fmt.Errorf("creating audio capturer: %w", err)
	}

	return &Services{
		Notifier:  notifier,
		Hotkey:    hk,
		Clipboard: clip,
		Audio:     capturer,
	}, nil
}

// Close releases all platform service resources.
func (s *Services) Close() {
	if s.Hotkey != nil {
		_ = s.Hotkey.Unregister()
	}
	if s.Audio != nil {
		s.Audio.Close()
	}
}
