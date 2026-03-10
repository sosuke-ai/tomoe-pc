package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/sosuke-ai/tomoe-pc/internal/config"
	"github.com/sosuke-ai/tomoe-pc/internal/gpu"
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

	fmt.Println("Daemon mode not yet implemented.")
	return nil
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

	// Model download placeholder
	fmt.Println("Model download: not yet implemented")
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

		// Daemon
		fmt.Println("Daemon: not running (daemon mode not yet implemented)")

		return nil
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
