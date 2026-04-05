package pii

import (
	"strings"
	"testing"
)

func TestLegalDetector_FrenchCaseNumber(t *testing.T) {
	detector := NewLegalDetector(LegalDetectorConfig{Jurisdictions: []string{"fr"}})

	tests := []struct {
		input   string
		wantTag string
	}{
		{"Affaire RG 23/12345", "[CASE_NO]"},
		{"jugement n° 2023/456", "[JUDGMENT_NO]"},
		{"avocat au barreau de Paris", "[LAWYER_BAR]"},
		{"CARPA n° 1234567890", "[CARPA_ACCOUNT]"},
		{"RCS Paris B 123 456 789", "[COMPANY_REG]"},
		{"section AB n° 1234", "[LAND_PARCEL]"},
		{"police n° 12345678", "[INSURANCE_POLICY]"},
		{"plainte n° 2023/12345", "[COMPLAINT_NO]"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := detector.DetectAndReplace(tt.input)
			if !strings.Contains(result, tt.wantTag) {
				t.Errorf("expected %s in result, got: %s", tt.wantTag, result)
			}
		})
	}
}

func TestLegalDetector_UKPatterns(t *testing.T) {
	detector := NewLegalDetector(LegalDetectorConfig{Jurisdictions: []string{"uk"}})

	tests := []struct {
		input   string
		wantTag string
	}{
		{"[2023] EWHC 1234", "[JUDGMENT_NO]"},
		{"[2023] EWCA Civ 567", "[JUDGMENT_NO]"},
		{"SRA ID: 123456", "[LAWYER_ID]"},
		{"Crime Ref: 123456789", "[CRIME_REF]"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := detector.DetectAndReplace(tt.input)
			if !strings.Contains(result, tt.wantTag) {
				t.Errorf("expected %s in result, got: %s", tt.wantTag, result)
			}
		})
	}
}

func TestLegalDetector_USPatterns(t *testing.T) {
	detector := NewLegalDetector(LegalDetectorConfig{Jurisdictions: []string{"us"}})

	tests := []struct {
		input   string
		wantTag string
	}{
		{"1:23-cv-01234", "[CASE_NO]"},
		{"No. 22-1234", "[CASE_NO]"},
		{"Bar No: 123456", "[LAWYER_ID]"},
		{"Docket No: 23-456", "[DOCKET_NO]"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := detector.DetectAndReplace(tt.input)
			if !strings.Contains(result, tt.wantTag) {
				t.Errorf("expected %s in result, got: %s", tt.wantTag, result)
			}
		})
	}
}

func TestLegalDetector_GermanPatterns(t *testing.T) {
	detector := NewLegalDetector(LegalDetectorConfig{Jurisdictions: []string{"de"}})

	tests := []struct {
		input   string
		wantTag string
	}{
		{"12 O 123/23", "[CASE_NO]"},
		{"Urteil vom 12.03.2023", "[JUDGMENT_NO]"},
		{"HRB 12345 (AG Berlin)", "[COMPANY_REG]"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := detector.DetectAndReplace(tt.input)
			if !strings.Contains(result, tt.wantTag) {
				t.Errorf("expected %s in result, got: %s", tt.wantTag, result)
			}
		})
	}
}

func TestLegalDetector_MultipleJurisdictions(t *testing.T) {
	detector := NewLegalDetector(LegalDetectorConfig{Jurisdictions: []string{"fr", "uk"}})

	// French
	result := detector.DetectAndReplace("RG 23/12345")
	if !strings.Contains(result, "[CASE_NO]") {
		t.Errorf("expected [CASE_NO] for French pattern, got: %s", result)
	}

	// UK
	result = detector.DetectAndReplace("[2023] EWHC 1234")
	if !strings.Contains(result, "[JUDGMENT_NO]") {
		t.Errorf("expected [JUDGMENT_NO] for UK pattern, got: %s", result)
	}
}

func TestLegalDetector_EmptyText(t *testing.T) {
	detector := NewLegalDetector(LegalDetectorConfig{})
	result := detector.DetectAndReplace("")
	if result != "" {
		t.Errorf("expected empty string, got: %s", result)
	}
}

func TestLegalDetector_NoMatch(t *testing.T) {
	detector := NewLegalDetector(LegalDetectorConfig{Jurisdictions: []string{"fr"}})
	result := detector.DetectAndReplace("Hello world, no legal patterns here")
	if result != "Hello world, no legal patterns here" {
		t.Errorf("expected unchanged text, got: %s", result)
	}
}

func TestLegalDetector_ComplexLegalText(t *testing.T) {
	detector := NewLegalDetector(LegalDetectorConfig{Jurisdictions: []string{"fr"}})

	input := "Dans l'affaire RG 23/12345, Maître Dupont, avocat au barreau de Paris, " +
		"représente la société SARL ABC (RCS Paris B 123 456 789). " +
		"Le tribunal a condamné le défendeur à payer la somme de € 50 000."

	result := detector.DetectAndReplace(input)

	// Check that at least some patterns were detected
	expectedTags := []string{"[CASE_NO]", "[LAWYER_BAR]", "[COMPANY_REG]"}
	found := 0
	for _, tag := range expectedTags {
		if strings.Contains(result, tag) {
			found++
		}
	}

	if found < 2 {
		t.Errorf("expected at least 2 legal tags, found %d. Result: %s", found, result)
	}
}
