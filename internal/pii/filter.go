// Package pii provides personally identifiable information (PII) filtering
// for msgvault's MCP interface to prevent leakage of sensitive data.
package pii

import (
	"context"

	"github.com/taoq-ai/wuming"
)

// Config controls PII filtering behavior.
type Config struct {
	// LegalMode enables legal-specific PII detectors (case numbers, lawyer IDs, etc.)
	LegalMode bool
	// NERMode enables named entity recognition via prose (PERSON, ORG, MONEY, etc.)
	NERMode bool
	// Jurisdictions for legal detectors: "fr", "uk", "us", "de". Empty = all.
	Jurisdictions []string
}

// Filter provides PII detection and redaction via a 3-pass pipeline:
//
//	Pass 1: legal regex — legal-specific patterns (case numbers, bar refs, etc.)
//	Pass 2: wuming — structured PII (email, phone, IBAN, NIR, etc.)
//	Pass 3: prose NER — named entities (PERSON, ORG, GPE, MONEY, etc.)
//
// Order matters: legal patterns must run BEFORE wuming, otherwise wuming
// replaces numbers with [PHONE]/[POSTAL_CODE] and breaks legal regex matching.
type Filter struct {
	wuming    *wuming.Wuming
	ner       *NERDetector
	legal     *LegalDetector
	legalMode bool
	nerMode   bool
}

// NewFilter creates a PII filter with the given configuration.
// A nil config uses defaults (wuming only, no legal or NER).
func NewFilter(cfg *Config) (*Filter, error) {
	w, err := wuming.New()
	if err != nil {
		return nil, err
	}

	f := &Filter{
		wuming: w,
	}

	if cfg != nil {
		f.legalMode = cfg.LegalMode
		f.nerMode = cfg.NERMode

		if cfg.LegalMode {
			f.legal = NewLegalDetector(LegalDetectorConfig{
				Jurisdictions: cfg.Jurisdictions,
			})
		}

		if cfg.NERMode {
			f.ner = NewNERDetector()
		}
	}

	return f, nil
}

// FilterString redacts PII from a single string through the 3-pass pipeline.
func (f *Filter) FilterString(ctx context.Context, input string) (string, error) {
	if input == "" {
		return input, nil
	}

	result := input
	var err error

	// Pass 1: legal regex — legal-specific patterns (case numbers, bar refs, etc.)
	// Must run FIRST so wuming doesn't consume numbers as [PHONE]/[POSTAL_CODE]
	if f.legalMode && f.legal != nil {
		result = f.legal.DetectAndReplace(result)
	}

	// Pass 2: wuming — structured PII (email, phone, IBAN, NIR, credit card, etc.)
	if f.wuming != nil {
		result, err = f.wuming.Redact(ctx, result)
		if err != nil {
			return input, err
		}
	}

	// Pass 3: NER — named entities (PERSON, ORG, GPE, MONEY, LAW, etc.)
	if f.nerMode && f.ner != nil {
		result = f.ner.DetectAndReplace(result)
	}

	return result, nil
}

// FilterStrings redacts PII from a slice of strings.
func (f *Filter) FilterStrings(ctx context.Context, inputs []string) ([]string, error) {
	if len(inputs) == 0 {
		return inputs, nil
	}

	results := make([]string, len(inputs))
	for i, input := range inputs {
		filtered, err := f.FilterString(ctx, input)
		if err != nil {
			return nil, err
		}
		results[i] = filtered
	}
	return results, nil
}
