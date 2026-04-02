// Package pii provides personally identifiable information (PII) filtering
// for msgvault's MCP interface to prevent leakage of sensitive data.
package pii

import (
	"context"

	"github.com/taoq-ai/wuming"
)

// Filter provides PII detection and redaction capabilities.
type Filter struct {
	wuming *wuming.Wuming
}

// NewFilter creates a new PII filter with default configuration.
// It detects and redacts common PII types like emails, phone numbers, names, etc.
func NewFilter() (*Filter, error) {
	// Create a wuming instance with default settings
	// This will detect common PII types and replace them with tags like [EMAIL], [PHONE], etc.
	w, err := wuming.New()
	if err != nil {
		return nil, err
	}

	return &Filter{
		wuming: w,
	}, nil
}

// FilterString redacts PII from a single string.
func (f *Filter) FilterString(ctx context.Context, input string) (string, error) {
	if f.wuming == nil {
		return input, nil
	}
	return f.wuming.Redact(ctx, input)
}

// FilterStrings redacts PII from a slice of strings.
func (f *Filter) FilterStrings(ctx context.Context, inputs []string) ([]string, error) {
	if f.wuming == nil || len(inputs) == 0 {
		return inputs, nil
	}

	results := make([]string, len(inputs))
	for i, input := range inputs {
		filtered, err := f.wuming.Redact(ctx, input)
		if err != nil {
			return nil, err
		}
		results[i] = filtered
	}
	return results, nil
}
