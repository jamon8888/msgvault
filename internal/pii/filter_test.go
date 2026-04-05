package pii

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilterString(t *testing.T) {
	filter, err := NewFilter(nil)
	require.NoError(t, err)

	tests := []struct {
		input       string
		containsPII bool
	}{
		{
			input:       "Hello world",
			containsPII: false,
		},
		{
			input:       "Contact me at john.doe@example.com",
			containsPII: true,
		},
		{
			input:       "Call me at 555-123-4567",
			containsPII: true,
		},
		{
			input:       "My SSN is 123-45-6789",
			containsPII: true,
		},
		{
			input:       "Email: jane@test.com, Phone: (555) 123-4567",
			containsPII: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			output, err := filter.FilterString(context.Background(), tt.input)
			require.NoError(t, err)

			if tt.containsPII {
				// Output should be different from input (PII was redacted)
				require.NotEqual(t, tt.input, output, "Expected PII to be redacted")
			} else {
				// Output should be the same as input (no PII to redact)
				require.Equal(t, tt.input, output, "Expected no change when no PII present")
			}
		})
	}
}

func TestFilterStrings(t *testing.T) {
	filter, err := NewFilter(nil)
	require.NoError(t, err)

	input := []string{
		"Hello world",
		"Contact me at john.doe@example.com",
		"Call me at 555-123-4567",
		"",
	}

	outputs, err := filter.FilterStrings(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, len(input), len(outputs))

	// First and last should be unchanged (no PII)
	require.Equal(t, "Hello world", outputs[0])
	require.Equal(t, "", outputs[3])

	// Middle ones should be changed (contain PII)
	require.NotEqual(t, "Contact me at john.doe@example.com", outputs[1])
	require.NotEqual(t, "Call me at 555-123-4567", outputs[2])
}
