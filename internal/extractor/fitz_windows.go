//go:build short || windows
// +build short windows

package extractor

import (
	"errors"
	"fmt"
)

var errPDFLibraryNotAvailable = errors.New("PDF library not available: run without -short flag or ensure libmupdf.dll is installed")

func (e *PDFExtractor) ExtractText(path string) (string, error) {
	return "", fmt.Errorf("%w: %s", errPDFLibraryNotAvailable, path)
}
