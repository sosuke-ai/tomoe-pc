package models

import (
	"archive/tar"
	"compress/bzip2"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/schollz/progressbar/v3"
)

// Status describes the state of downloaded models.
type Status struct {
	ParakeetReady bool
	VADReady      bool
	ModelDir      string
	EncoderPath   string
	DecoderPath   string
	JoinerPath    string
	TokensPath    string
	VADPath       string
}

// Manager handles model download, extraction, and verification.
type Manager struct {
	modelDir string
}

// NewManager creates a Manager for the given model directory.
func NewManager(modelDir string) *Manager {
	return &Manager{modelDir: modelDir}
}

// ModelDir returns the model storage directory.
func (m *Manager) ModelDir() string {
	return m.modelDir
}

// Check inspects the model directory and reports what's present.
func (m *Manager) Check() *Status {
	parakeetDir := filepath.Join(m.modelDir, ParakeetSubdir)

	s := &Status{
		ModelDir:    m.modelDir,
		EncoderPath: filepath.Join(parakeetDir, encoderFile),
		DecoderPath: filepath.Join(parakeetDir, decoderFile),
		JoinerPath:  filepath.Join(parakeetDir, joinerFile),
		TokensPath:  filepath.Join(parakeetDir, tokensFile),
		VADPath:     filepath.Join(m.modelDir, SileroVADFile),
	}

	s.ParakeetReady = allFilesExist(
		s.EncoderPath, s.DecoderPath, s.JoinerPath, s.TokensPath,
	)
	s.VADReady = fileExists(s.VADPath)

	return s
}

// Ready reports whether all required models are present.
func (s *Status) Ready() bool {
	return s.ParakeetReady && s.VADReady
}

// String returns a human-readable summary.
func (s *Status) String() string {
	parakeet := "not downloaded"
	if s.ParakeetReady {
		parakeet = "ready"
	}
	vad := "not downloaded"
	if s.VADReady {
		vad = "ready"
	}
	return fmt.Sprintf("Model dir: %s\nParakeet TDT INT8: %s\nSilero VAD: %s",
		s.ModelDir, parakeet, vad)
}

// Download downloads and extracts all required models.
// If force is true, existing models are re-downloaded.
func (m *Manager) Download(force bool) error {
	if err := os.MkdirAll(m.modelDir, 0o755); err != nil {
		return fmt.Errorf("creating model directory: %w", err)
	}

	status := m.Check()

	// Download Parakeet TDT archive
	if force || !status.ParakeetReady {
		fmt.Println("Downloading Parakeet TDT 0.6B v3 INT8 model...")
		if err := m.downloadAndExtractArchive(ParakeetArchiveURL); err != nil {
			return fmt.Errorf("downloading Parakeet model: %w", err)
		}
		fmt.Println("Parakeet TDT model downloaded and extracted.")
	} else {
		fmt.Println("Parakeet TDT model already present, skipping.")
	}

	// Download Silero VAD
	if force || !status.VADReady {
		fmt.Println("Downloading Silero VAD model...")
		vadPath := filepath.Join(m.modelDir, SileroVADFile)
		if err := downloadFile(SileroVADURL, vadPath); err != nil {
			return fmt.Errorf("downloading Silero VAD: %w", err)
		}
		fmt.Println("Silero VAD model downloaded.")
	} else {
		fmt.Println("Silero VAD model already present, skipping.")
	}

	// Verify
	final := m.Check()
	if !final.Ready() {
		return fmt.Errorf("model verification failed after download")
	}

	return nil
}

// downloadAndExtractArchive downloads a tar.bz2 archive and extracts it to the model directory.
func (m *Manager) downloadAndExtractArchive(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP GET: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	bar := progressbar.DefaultBytes(resp.ContentLength, "downloading")
	reader := io.TeeReader(resp.Body, bar)

	return extractTarBz2(reader, m.modelDir)
}

// extractTarBz2 extracts a tar.bz2 stream to the destination directory.
func extractTarBz2(r io.Reader, destDir string) error {
	bzReader := bzip2.NewReader(r)
	tarReader := tar.NewReader(bzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar: %w", err)
		}

		// Sanitize path to prevent directory traversal
		cleanName := filepath.Clean(header.Name)
		if strings.Contains(cleanName, "..") {
			return fmt.Errorf("tar contains path traversal: %s", header.Name)
		}

		target := filepath.Join(destDir, cleanName)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("creating directory %s: %w", target, err)
			}

		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("creating parent directory for %s: %w", target, err)
			}

			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("creating file %s: %w", target, err)
			}

			if _, err := io.Copy(f, tarReader); err != nil {
				f.Close()
				return fmt.Errorf("writing file %s: %w", target, err)
			}
			f.Close()
		}
	}

	return nil
}

// downloadFile downloads a URL to a local file with a progress bar.
func downloadFile(url, destPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP GET: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer f.Close()

	bar := progressbar.DefaultBytes(resp.ContentLength, "downloading")
	if _, err := io.Copy(io.MultiWriter(f, bar), resp.Body); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func allFilesExist(paths ...string) bool {
	for _, p := range paths {
		if !fileExists(p) {
			return false
		}
	}
	return true
}
