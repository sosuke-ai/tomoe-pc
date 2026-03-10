package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/sosuke-ai/tomoe-pc/internal/audio"
	"github.com/sosuke-ai/tomoe-pc/internal/config"
	"github.com/sosuke-ai/tomoe-pc/internal/daemon"
	"github.com/sosuke-ai/tomoe-pc/internal/gpu"
	"github.com/sosuke-ai/tomoe-pc/internal/models"
	"github.com/sosuke-ai/tomoe-pc/internal/platform"
	"github.com/sosuke-ai/tomoe-pc/internal/transcribe"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "tomoe",
	Short: "Local-first speech-to-text for Linux",
	Long:  "Tomoe captures microphone audio, transcribes locally using Parakeet TDT via sherpa-onnx, and delivers text to clipboard.",
	RunE:  runStart,
}

func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(modelCmd)
	rootCmd.AddCommand(transcribeCmd)
	rootCmd.AddCommand(devicesCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(versionCmd)
}

// startCmd is an alias for the root command.
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the daemon (alias for running tomoe with no arguments)",
	RunE:  runStart,
}

func runStart(cmd *cobra.Command, args []string) error {
	if !config.Exists() {
		fmt.Println("First run detected, running auto-init...")
		if err := runInit(cmd, args); err != nil {
			return err
		}
	}

	// Check if already running
	if daemon.IsRunning() {
		return fmt.Errorf("daemon already running (PID %d)", daemon.ReadPID())
	}

	// Load config
	cfg, err := config.Load(config.Path())
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Check models
	mgr := models.NewManager(cfg.Transcription.ModelPath)
	status := mgr.Check()
	if !status.Ready() {
		return fmt.Errorf("models not downloaded (run 'tomoe init' or 'tomoe model download')")
	}

	// Create transcription engine
	engine, err := transcribe.NewEngine(transcribe.Config{
		EncoderPath: status.EncoderPath,
		DecoderPath: status.DecoderPath,
		JoinerPath:  status.JoinerPath,
		TokensPath:  status.TokensPath,
		VADPath:     status.VADPath,
		UseGPU:      cfg.Transcription.GPUEnabled,
	})
	if err != nil {
		return fmt.Errorf("creating transcription engine: %w", err)
	}
	defer engine.Close()

	// Create platform services
	svc, err := platform.New(cfg)
	if err != nil {
		return fmt.Errorf("initializing platform services: %w", err)
	}
	defer svc.Close()

	// Run daemon
	d := daemon.New(cfg, engine, svc)
	return d.Run(context.Background())
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the running daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !daemon.IsRunning() {
			fmt.Println("Daemon is not running.")
			return nil
		}
		if err := daemon.StopRemote(); err != nil {
			return fmt.Errorf("stopping daemon: %w", err)
		}
		fmt.Println("Stop signal sent to daemon.")
		return nil
	},
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Detect system, generate config, and download model",
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	fmt.Println("=== Tomoe Auto-Init ===")
	fmt.Println()

	// Detect GPU
	fmt.Println("Detecting GPU...")
	gpuInfo := gpu.Detect()
	fmt.Println(gpuInfo)
	fmt.Println()

	// Detect display server
	displayServer := os.Getenv("XDG_SESSION_TYPE")
	if displayServer == "" {
		displayServer = "unknown"
	}
	fmt.Printf("Display server: %s\n", displayServer)
	fmt.Println()

	// Generate config
	cfg := config.DefaultConfig()
	cfg.Transcription.GPUEnabled = gpuInfo.Available && gpuInfo.Sufficient

	cfgPath := config.Path()
	if err := config.Save(cfg, cfgPath); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}
	fmt.Printf("Config written to: %s\n", cfgPath)
	fmt.Println()

	// Download models
	mgr := models.NewManager(cfg.Transcription.ModelPath)
	if err := mgr.Download(false); err != nil {
		return fmt.Errorf("downloading models: %w", err)
	}
	fmt.Println()

	// Summary
	modelStatus := mgr.Check()
	fmt.Println(modelStatus)
	fmt.Println()

	fmt.Println("=== Init complete ===")
	return nil
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show daemon status, system info, model info, and GPU detection",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("=== Tomoe Status ===")
		fmt.Println()

		// GPU
		gpuInfo := gpu.Detect()
		fmt.Println(gpuInfo)
		fmt.Println()

		// Config
		cfgPath := config.Path()
		if config.Exists() {
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			fmt.Printf("Config: %s\n", cfgPath)
			fmt.Printf("  Hotkey:      %s\n", cfg.Hotkey.Binding)
			fmt.Printf("  Audio:       %s\n", cfg.Audio.Device)
			fmt.Printf("  GPU enabled: %v\n", cfg.Transcription.GPUEnabled)
			fmt.Printf("  Model path:  %s\n", cfg.Transcription.ModelPath)
			fmt.Printf("  Clipboard:   %v\n", cfg.Output.Clipboard)
			fmt.Printf("  Auto-paste:  %v\n", cfg.Output.AutoPaste)
		} else {
			fmt.Printf("Config: not found (run 'tomoe init')\n")
		}
		fmt.Println()

		// Models
		mgr := models.NewManager(config.ModelDir())
		modelStatus := mgr.Check()
		fmt.Println(modelStatus)
		fmt.Println()

		// Audio devices
		devices, err := audio.ListDevices()
		if err != nil {
			fmt.Printf("Audio: error listing devices (%v)\n", err)
		} else if len(devices) == 0 {
			fmt.Println("Audio: no capture devices found")
		} else {
			fmt.Printf("Audio: %d capture device(s)\n", len(devices))
			for _, d := range devices {
				def := ""
				if d.IsDefault {
					def = " *"
				}
				fmt.Printf("  - %s%s\n", d.Name, def)
			}
		}
		fmt.Println()

		// Display server
		ds := os.Getenv("XDG_SESSION_TYPE")
		if ds == "" {
			ds = "unknown"
		}
		fmt.Printf("Display: %s\n", ds)
		fmt.Println()

		// Daemon
		if daemon.IsRunning() {
			fmt.Printf("Daemon: running (PID %d)\n", daemon.ReadPID())
		} else {
			fmt.Println("Daemon: not running")
		}

		return nil
	},
}

var modelCmd = &cobra.Command{
	Use:   "model",
	Short: "Manage transcription models",
}

func init() {
	modelCmd.AddCommand(modelDownloadCmd)
	modelCmd.AddCommand(modelStatusCmd)
}

var modelDownloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download or re-download Parakeet TDT INT8 model and Silero VAD",
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")
		mgr := models.NewManager(config.ModelDir())
		return mgr.Download(force)
	},
}

func init() {
	modelDownloadCmd.Flags().Bool("force", false, "Force re-download even if models exist")
}

var modelStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show downloaded model info and integrity check",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := models.NewManager(config.ModelDir())
		status := mgr.Check()
		fmt.Println(status)
		return nil
	},
}

var transcribeCmd = &cobra.Command{
	Use:   "transcribe <file>",
	Short: "Transcribe an audio file (WAV, FLAC, OGG)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]

		if !transcribe.IsSupportedFormat(filePath) {
			return fmt.Errorf("unsupported audio format: %s (supported: .wav, .flac, .ogg)", filePath)
		}

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", filePath)
		}

		// Load config or use defaults
		var cfg *config.Config
		if config.Exists() {
			var err error
			cfg, err = config.Load(config.Path())
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
		} else {
			cfg = config.DefaultConfig()
		}

		// Check model status
		mgr := models.NewManager(cfg.Transcription.ModelPath)
		status := mgr.Check()
		if !status.Ready() {
			return fmt.Errorf("models not downloaded (run 'tomoe init' or 'tomoe model download')")
		}

		// Create transcription engine
		engine, err := transcribe.NewEngine(transcribe.Config{
			EncoderPath: status.EncoderPath,
			DecoderPath: status.DecoderPath,
			JoinerPath:  status.JoinerPath,
			TokensPath:  status.TokensPath,
			VADPath:     status.VADPath,
			UseGPU:      cfg.Transcription.GPUEnabled,
		})
		if err != nil {
			return fmt.Errorf("creating transcription engine: %w", err)
		}
		defer engine.Close()

		// Transcribe
		result, err := engine.TranscribeFile(filePath)
		if err != nil {
			return fmt.Errorf("transcription failed: %w", err)
		}

		if result.Text == "" {
			fmt.Println("(no speech detected)")
			return nil
		}

		fmt.Println(result.Text)

		if lang := result.Language; lang != "" {
			fmt.Fprintf(os.Stderr, "Language: %s | Duration: %.1fs\n", lang, result.Duration)
		} else {
			fmt.Fprintf(os.Stderr, "Duration: %.1fs\n", result.Duration)
		}

		return nil
	},
}

var devicesCmd = &cobra.Command{
	Use:   "devices",
	Short: "List audio input devices",
	RunE: func(cmd *cobra.Command, args []string) error {
		devices, err := audio.ListDevices()
		if err != nil {
			return fmt.Errorf("listing audio devices: %w", err)
		}

		if len(devices) == 0 {
			fmt.Println("No audio capture devices found.")
			return nil
		}

		for _, d := range devices {
			def := ""
			if d.IsDefault {
				def = " (default)"
			}
			fmt.Printf("  %s%s\n", d.Name, def)
		}
		return nil
	},
}

// Version is set at build time via -ldflags.
var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("tomoe %s\n", Version)
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Print current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath := config.Path()
		if !config.Exists() {
			fmt.Println("No config file found. Run 'tomoe init' to generate one.")
			return nil
		}

		data, err := os.ReadFile(cfgPath)
		if err != nil {
			return fmt.Errorf("reading config: %w", err)
		}

		fmt.Print(string(data))
		return nil
	},
}
