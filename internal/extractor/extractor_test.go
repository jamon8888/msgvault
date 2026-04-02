package extractor

import (
	"errors"
	"testing"
)

type MockExtractor struct{}

func (m *MockExtractor) ExtractText(path string) (string, error) {
	return "", nil
}

func TestExtractorInterface(t *testing.T) {
	var _ Extractor = (*MockExtractor)(nil)
}

func TestExtractPDF(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real PDF extraction test")
	}
	e, err := NewExtractor("pdf")
	if err != nil {
		t.Errorf("NewExtractor failed: %v", err)
	}
	text, err := e.ExtractText("test.pdf")
	if err != nil {
		t.Errorf("ExtractText failed: %v", err)
	}
	if text == "" {
		t.Error("Expected non-empty text")
	}
}

func TestExtractDOCX(t *testing.T) {
	e, err := NewExtractor("docx")
	if err != nil {
		t.Errorf("NewExtractor failed: %v", err)
	}
	text, err := e.ExtractText("test.docx")
	if err != nil {
		t.Errorf("ExtractText failed: %v", err)
	}
	if text == "" {
		t.Error("Expected non-empty text")
	}
}

func TestExtractTXT(t *testing.T) {
	e, err := NewExtractor("txt")
	if err != nil {
		t.Errorf("NewExtractor failed: %v", err)
	}
	text, err := e.ExtractText("test.txt")
	if err != nil {
		t.Errorf("ExtractText failed: %v", err)
	}
	if text == "" {
		t.Error("Expected non-empty text")
	}
}

func TestUnsupportedFormat(t *testing.T) {
	_, err := NewExtractor("xyz")
	if err == nil {
		t.Error("Expected error for unsupported format")
	}
	if !errors.Is(err, ErrUnsupportedFormat) {
		t.Errorf("Expected ErrUnsupportedFormat, got: %v", err)
	}
}
