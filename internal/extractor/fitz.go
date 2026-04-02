//go:build !short && !windows
// +build !short,!windows

package extractor

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gen2brain/go-fitz"
)

var errPDFLibraryNotAvailable = errors.New("PDF library not available")

func (e *PDFExtractor) ExtractText(path string) (string, error) {
	doc, err := fitz.New(path)
	if err != nil {
		if strings.Contains(err.Error(), "libmupdf") || strings.Contains(err.Error(), "dll") {
			return "", fmt.Errorf("%w (ensure libmupdf.dll is in PATH): %v", errPDFLibraryNotAvailable, err)
		}
		return "", err
	}
	defer doc.Close()

	var text strings.Builder
	for i := 0; i < doc.NumPage(); i++ {
		pageText, err := doc.Text(i)
		if err != nil {
			continue
		}
		text.WriteString(pageText)
		text.WriteString("\n")
	}
	return text.String(), nil
}
