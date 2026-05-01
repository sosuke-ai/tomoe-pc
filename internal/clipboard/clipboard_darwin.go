//go:build darwin

package clipboard

import (
	"fmt"

	atotto "github.com/atotto/clipboard"
)

// darwinWriter is a stub Writer for darwin builds.
// Write uses atotto/clipboard (which works on macOS via pbcopy);
// TypeText is unimplemented until Phase 2.5 (CGEventPost synthetic Cmd+V).
type darwinWriter struct{}

// NewWriter returns the darwin Writer.
func NewWriter() Writer {
	return &darwinWriter{}
}

func (w *darwinWriter) Write(text string) error {
	return atotto.WriteAll(text)
}

func (w *darwinWriter) TypeText(text string) error {
	return fmt.Errorf("clipboard: darwin TypeText not yet wired (Phase 2.5: CGEventPost)")
}
