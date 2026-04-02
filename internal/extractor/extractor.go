package extractor

import (
	"errors"
	"strings"

	"github.com/gomutex/godocx"
	"github.com/gomutex/godocx/docx"
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
	doc, err := godocx.OpenDocument(path)
	if err != nil {
		return "", err
	}
	defer doc.Close()

	var text strings.Builder
	if doc.Document != nil && doc.Document.Body != nil {
		for _, child := range doc.Document.Body.Children {
			if child.Para != nil {
				text.WriteString(extractParagraphText(child.Para))
				text.WriteString("\n")
			}
		}
	}
	return text.String(), nil
}

func extractParagraphText(p *docx.Paragraph) string {
	if p == nil {
		return ""
	}
	var result strings.Builder
	for _, child := range p.GetCT().Children {
		if child.Run != nil {
			for _, runChild := range child.Run.Children {
				if runChild.Text != nil {
					result.WriteString(runChild.Text.Text)
				}
			}
		}
	}
	return result.String()
}

type TXTExtractor struct{}

func (e *TXTExtractor) ExtractText(path string) (string, error) {
	return "mock text", nil
}
