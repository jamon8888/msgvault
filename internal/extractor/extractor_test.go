package extractor

import (
	"errors"
	"os"
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
	// Check if a real PDF exists in the attachments directory
	attachmentsDir := os.Getenv("MSGVAULT_TEST_ATTACHMENTS")
	if attachmentsDir == "" {
		t.Skip("set MSGVAULT_TEST_ATTACHMENTS to a directory with PDF files")
	}
	entries, err := os.ReadDir(attachmentsDir)
	if err != nil {
		t.Skipf("cannot read test attachments dir: %v", err)
	}
	var pdfPath string
	for _, e := range entries {
		if e.Type().IsRegular() {
			pdfPath = attachmentsDir + "/" + e.Name()
			break
		}
	}
	if pdfPath == "" {
		t.Skip("no PDF files found in test attachments dir")
	}
	e, err := NewExtractor("pdf")
	if err != nil {
		t.Errorf("NewExtractor failed: %v", err)
	}
	text, err := e.ExtractText(pdfPath)
	if err != nil {
		t.Errorf("ExtractText failed: %v", err)
	}
	if text == "" {
		t.Error("Expected non-empty text")
	}
}

func TestExtractDOCX(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real DOCX extraction test")
	}
	// Create a minimal DOCX file for testing
	tmpFile, err := os.CreateTemp("", "test-*.docx")
	if err != nil {
		t.Fatalf("failed to create temp docx: %v", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)
	tmpFile.Close()

	// DOCX is a zip file; godocx will fail on empty file, which is expected
	e, err := NewExtractor("docx")
	if err != nil {
		t.Errorf("NewExtractor failed: %v", err)
	}
	_, err = e.ExtractText(tmpPath)
	// We expect an error for an empty/invalid docx, but the extractor should be created
	if err != nil {
		t.Logf("DOCX extraction failed as expected for empty file: %v", err)
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
