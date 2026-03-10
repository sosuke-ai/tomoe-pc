package daemon

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sosuke-ai/tomoe-pc/internal/audio"
	"github.com/sosuke-ai/tomoe-pc/internal/config"
	"github.com/sosuke-ai/tomoe-pc/internal/platform"
	"github.com/sosuke-ai/tomoe-pc/internal/transcribe"
)

// Daemon orchestrates the hotkey → capture → transcribe → clipboard pipeline.
type Daemon struct {
	cfg    *config.Config
	engine transcribe.Engine
	svc    *platform.Services
}

// New creates a Daemon with the given dependencies.
func New(cfg *config.Config, engine transcribe.Engine, svc *platform.Services) *Daemon {
	return &Daemon{
		cfg:    cfg,
		engine: engine,
		svc:    svc,
	}
}

// Run starts the daemon main loop. Blocks until SIGTERM/SIGINT or context cancel.
func (d *Daemon) Run(ctx context.Context) error {
	// Write PID file
	if err := WritePID(); err != nil {
		return fmt.Errorf("writing PID file: %w", err)
	}
	defer RemovePID()

	// Register hotkey
	if err := d.svc.Hotkey.Register(); err != nil {
		return fmt.Errorf("registering hotkey: %w", err)
	}

	// Signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(sigCh)

	_ = d.svc.Notifier.Send("Tomoe", fmt.Sprintf("Ready — press %s to record", d.cfg.Hotkey.Binding))
	fmt.Printf("Tomoe daemon started. Press %s to toggle recording.\n", d.cfg.Hotkey.Binding)

	recording := false

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Shutting down...")
			return nil

		case sig := <-sigCh:
			fmt.Printf("Received %s, shutting down...\n", sig)
			if recording {
				_ = d.svc.Audio.Stop()
			}
			return nil

		case <-d.svc.Hotkey.Keydown():
			if !recording {
				// Start recording
				recording = true
				d.svc.Audio.Reset()
				if err := d.svc.Audio.Start(); err != nil {
					_ = d.svc.Notifier.Send("Tomoe", "Failed to start recording")
					fmt.Printf("Error starting recording: %v\n", err)
					recording = false
					continue
				}
				_ = d.svc.Notifier.Send("Tomoe", "Recording...")
				fmt.Println("Recording started...")
			} else {
				// Stop recording and transcribe
				recording = false
				if err := d.svc.Audio.Stop(); err != nil {
					fmt.Printf("Error stopping recording: %v\n", err)
				}

				samples := d.svc.Audio.Samples()
				if len(samples) == 0 {
					_ = d.svc.Notifier.Send("Tomoe", "No audio captured")
					fmt.Println("No audio captured.")
					continue
				}

				// Apply DSP pipeline
				samples = audio.ProcessPipeline(samples, 16000, -40)

				_ = d.svc.Notifier.Send("Tomoe", "Transcribing...")
				fmt.Printf("Transcribing %.1fs of audio...\n", float64(len(samples))/16000.0)

				start := time.Now()
				result, err := d.engine.TranscribeSamples(samples)
				elapsed := time.Since(start)

				if err != nil {
					_ = d.svc.Notifier.Send("Tomoe", fmt.Sprintf("Transcription failed: %v", err))
					fmt.Printf("Transcription error: %v\n", err)
					continue
				}

				if result.Text == "" {
					_ = d.svc.Notifier.Send("Tomoe", "No speech detected")
					fmt.Println("No speech detected.")
					continue
				}

				// Copy to clipboard
				if d.cfg.Output.Clipboard {
					if err := d.svc.Clipboard.Write(result.Text); err != nil {
						fmt.Printf("Clipboard error: %v\n", err)
					}
				}

				// Auto-paste
				if d.cfg.Output.AutoPaste {
					if err := d.svc.Clipboard.AutoPaste(); err != nil {
						// Non-fatal — auto-paste may not be available
						fmt.Printf("Auto-paste error: %v\n", err)
					}
				}

				lang := ""
				if result.Language != "" {
					lang = fmt.Sprintf(" [%s]", result.Language)
				}
				_ = d.svc.Notifier.Send("Tomoe", fmt.Sprintf("Done%s — copied to clipboard", lang))
				fmt.Printf("Transcribed in %s%s: %s\n", elapsed.Round(time.Millisecond), lang, result.Text)
			}
		}
	}
}
