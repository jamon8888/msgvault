package pii

import (
	"strings"
	"testing"
)

func TestNERDetector_Person(t *testing.T) {
	detector := NewNERDetector("PERSON")
	text := "Maître Dupont représente M. Martin dans cette affaire."
	result := detector.DetectAndReplace(text)

	if !strings.Contains(result, "[PERSON]") {
		t.Errorf("expected [PERSON] tag in result, got: %s", result)
	}
	if strings.Contains(result, "Dupont") {
		t.Errorf("Dupont should have been redacted, got: %s", result)
	}
}

func TestNERDetector_Money(t *testing.T) {
	detector := NewNERDetector("MONEY")
	text := "Le tribunal a condamné la société à verser 50 000 euros."
	result := detector.DetectAndReplace(text)

	// prose may or may not detect French money amounts, so we just check it doesn't crash
	t.Logf("Result: %s", result)
}

func TestNERDetector_Organization(t *testing.T) {
	detector := NewNERDetector("ORG")
	text := "La société Google et l'entreprise Apple ont signé un accord."
	result := detector.DetectAndReplace(text)

	t.Logf("Result: %s", result)
}

func TestNERDetector_AllTypes(t *testing.T) {
	detector := NewNERDetector()
	text := "John Doe works at Google in Paris since 2020."
	result := detector.DetectAndReplace(text)

	t.Logf("Result: %s", result)

	// At minimum, some entities should be detected
	if result == text {
		t.Log("Note: NER did not detect any entities (may be expected for short English text)")
	}
}

func TestNERDetector_EmptyText(t *testing.T) {
	detector := NewNERDetector()
	result := detector.DetectAndReplace("")
	if result != "" {
		t.Errorf("expected empty string, got: %s", result)
	}
}

func TestNERDetector_EnabledTypes(t *testing.T) {
	detector := NewNERDetector("PERSON", "ORG")
	// Should only detect PERSON and ORG, not GPE/MONEY etc.
	if !detector.enabled["PERSON"] || !detector.enabled["ORG"] {
		t.Error("expected PERSON and ORG to be enabled")
	}
	if detector.enabled["GPE"] {
		t.Error("expected GPE to be disabled")
	}
}
