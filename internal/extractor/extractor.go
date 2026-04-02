package extractor

import (
	"errors"
)

var (
	ErrUnsupportedFormat = errors.New("unsupported format")
)

type Extractor interface {
	ExtractText(path string) (string, error)
}

func NewExtractor(format string) (Extractor, error) {
	switch format {
	case "pdf":
		return &PDFExtractor{}, nil
	case "docx":
		return &DOCXExtractor{}, nil
	case "txt":
		return &TXTExtractor{}, nil
	default:
		return nil, ErrUnsupportedFormat
	}
}

type PDFExtractor struct{}

type DOCXExtractor struct{}

func (e *DOCXExtractor) ExtractText(path string) (string, error) {
	return "mock text", nil
}

type TXTExtractor struct{}

func (e *TXTExtractor) ExtractText(path string) (string, error) {
	return "mock text", nil
}
