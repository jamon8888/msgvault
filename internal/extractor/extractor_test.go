package extractor

import (
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
