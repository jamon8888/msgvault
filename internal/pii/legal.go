package pii

import (
	"regexp"
	"sort"
	"strings"
)

// legalPattern defines a regex-based legal PII detector.
type legalPattern struct {
	name   string
	re     *regexp.Regexp
	tag    string
	locale string
}

type legalMatch struct {
	start int
	end   int
	tag   string
}

// LegalDetector detects legal-specific PII patterns via regex.
type LegalDetector struct {
	patterns []legalPattern
}

// LegalDetectorConfig controls which jurisdictions are active.
type LegalDetectorConfig struct {
	Jurisdictions []string // "fr", "uk", "us", "de", "all". Empty = all.
}

// NewLegalDetector creates a legal PII detector with the given config.
func NewLegalDetector(cfg LegalDetectorConfig) *LegalDetector {
	d := &LegalDetector{}
	d.patterns = buildPatterns(cfg)
	return d
}

// DetectAndReplace finds legal PII patterns and replaces them with tags.
func (d *LegalDetector) DetectAndReplace(text string) string {
	if text == "" || len(d.patterns) == 0 {
		return text
	}

	var matches []legalMatch
	for _, p := range d.patterns {
		for _, m := range p.re.FindAllStringSubmatchIndex(text, -1) {
			if len(m) < 2 {
				continue
			}
			start, end := m[0], m[1]
			matches = append(matches, legalMatch{start, end, p.tag})
		}
	}

	if len(matches) == 0 {
		return text
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].start > matches[j].start
	})

	matches = dedupOverlaps(matches)

	result := text
	for _, m := range matches {
		result = result[:m.start] + m.tag + result[m.end:]
	}

	return result
}

func dedupOverlaps(matches []legalMatch) []legalMatch {
	if len(matches) <= 1 {
		return matches
	}
	result := []legalMatch{matches[0]}
	for i := 1; i < len(matches); i++ {
		last := result[len(result)-1]
		if matches[i].end <= last.start {
			result = append(result, matches[i])
		}
	}
	return result
}

func buildPatterns(cfg LegalDetectorConfig) []legalPattern {
	var all []legalPattern

	jurisdictions := cfg.Jurisdictions
	if len(jurisdictions) == 0 {
		jurisdictions = []string{"fr", "uk", "us", "de"}
	}

	active := make(map[string]bool, len(jurisdictions))
	for _, j := range jurisdictions {
		active[strings.ToLower(j)] = true
	}
	if active["all"] {
		active = map[string]bool{"fr": true, "uk": true, "us": true, "de": true}
	}

	if active["fr"] {
		all = append(all, frenchPatterns...)
	}
	if active["uk"] {
		all = append(all, ukPatterns...)
	}
	if active["us"] {
		all = append(all, usPatterns...)
	}
	if active["de"] {
		all = append(all, dePatterns...)
	}

	return all
}

// ─── French Legal Patterns ─────────────────────────────────────────────

var frenchPatterns = []legalPattern{
	// Numéro de dossier / affaire: "RG 23/12345", "n° 23/00123", "affaire n° 23/12345"
	{
		name: "fr_case_number",
		re:   regexp.MustCompile(`(?i)(?:n°\s*|numéro\s*|affaire\s*(?:n°\s*)?|RG\s*)(\d{2}[/\-]\d{3,6})`),
		tag:  "[CASE_NO]",
	},
	// Numéro de jugement: "jugement n° 2023/456", "arrêt n° 1234"
	{
		name: "fr_judgment_number",
		re:   regexp.MustCompile(`(?i)(?:jugement|arrêt|ordonnance|décision)\s*(?:n°\s*)?(\d{4}[/\-]\d{2,5})`),
		tag:  "[JUDGMENT_NO]",
	},
	// Avocat au barreau: "avocat au barreau de Paris", "CAPA Paris"
	{
		name: "fr_lawyer_bar",
		re:   regexp.MustCompile(`(?i)avocat(?:\s+au)?\s+barreau\s+de\s+\w+`),
		tag:  "[LAWYER_BAR]",
	},
	{
		name: "fr_capa",
		re:   regexp.MustCompile(`(?i)CAPA\s+\w+`),
		tag:  "[LAWYER_BAR]",
	},
	{
		name: "fr_rin",
		re:   regexp.MustCompile(`(?i)RIN[:\s]*(\d{8,12})`),
		tag:  "[LAWYER_ID]",
	},
	// CARPA: "CARPA n° 1234567890"
	{
		name: "fr_carpa",
		re:   regexp.MustCompile(`(?i)CARPA\s*(?:n°\s*)?(\d{8,14})`),
		tag:  "[CARPA_ACCOUNT]",
	},
	// Numéro de parquet
	{
		name: "fr_parquet",
		re:   regexp.MustCompile(`(?i)parquet\s*(?:n°\s*)?(\d{2,4}[/\-]\d{3,6})`),
		tag:  "[PARQUET_NO]",
	},
	// Montant condamnation: "condamné à payer la somme de € 50 000"
	{
		name: "fr_damages_eur",
		re:   regexp.MustCompile(`(?i)(?:condamné|payer|somme\s+de|dommages[\s-]*intérêts)\s*(?:de\s*)?([€£]\s*[\d\s,.]+)`),
		tag:  "[SETTLEMENT_AMOUNT]",
	},
	// RCS: "RCS Paris B 123 456 789"
	{
		name: "fr_rcs",
		re:   regexp.MustCompile(`(?i)RCS\s+\w+\s+[A-Z]\s*[\d\s]{6,12}`),
		tag:  "[COMPANY_REG]",
	},
	// SIRET
	{
		name: "fr_siret",
		re:   regexp.MustCompile(`\b(\d{3}\s?\d{3}\s?\d{3}\s?\d{5})\b`),
		tag:  "[SIRET]",
	},
	// Numéro de minute notaire
	{
		name: "fr_notaire_minute",
		re:   regexp.MustCompile(`(?i)(?:notaire|acte\s+notarié)\s*(?:n°\s*)?(\d{4,8})`),
		tag:  "[NOTARY_REF]",
	},
	// Numéro d'écrou
	{
		name: "fr_ecrou",
		re:   regexp.MustCompile(`(?i)écrou\s*(?:n°\s*)?(\d{4,8})`),
		tag:  "[PRISONER_NO]",
	},
	// Numéro de plainte
	{
		name: "fr_plainte",
		re:   regexp.MustCompile(`(?i)plainte\s*(?:n°\s*)?(\d{4}[/\-]\d{3,7})`),
		tag:  "[COMPLAINT_NO]",
	},
	// Numéro de mandat
	{
		name: "fr_mandat",
		re:   regexp.MustCompile(`(?i)mandat\s*(?:n°\s*)?(\d{4,8})`),
		tag:  "[WARRANT_NO]",
	},
	// Parcelle cadastrale: "section AB n° 1234"
	{
		name: "fr_parcelle",
		re:   regexp.MustCompile(`(?i)section\s+[A-Z]{1,4}\s*(?:n°\s*)?\d{1,6}`),
		tag:  "[LAND_PARCEL]",
	},
	// Référence hypothécaire
	{
		name: "fr_hypotheque",
		re:   regexp.MustCompile(`(?i)hypothèque\s*(?:n°\s*)?(\d{4,10})`),
		tag:  "[MORTGAGE_REF]",
	},
	// Numéro de police d'assurance
	{
		name: "fr_police_assurance",
		re:   regexp.MustCompile(`(?i)police\s+(?:n°\s*)?(\d{6,12})`),
		tag:  "[INSURANCE_POLICY]",
	},
	// Numéro de brevet EP
	{
		name: "fr_patent_ep",
		re:   regexp.MustCompile(`(?i)EP\s+\d[\s\d]{6,10}\s*[A-Z]\d*`),
		tag:  "[PATENT_NO]",
	},
	// Numéro de marque INPI
	{
		name: "fr_inpi",
		re:   regexp.MustCompile(`(?i)INPI\s*(?:n°\s*)?(\d{6,8})`),
		tag:  "[TRADEMARK_NO]",
	},
	// Numéro de casier judiciaire
	{
		name: "fr_casier",
		re:   regexp.MustCompile(`(?i)casier\s+(?:judiciaire\s+)?(?:n°\s*)?(\d{4,8})`),
		tag:  "[CRIMINAL_RECORD]",
	},
	// Numéro de séjour hospitalier
	{
		name: "fr_sejour",
		re:   regexp.MustCompile(`(?i)séjour\s*(?:n°\s*)?(\d{6,12})`),
		tag:  "[HOSPITAL_STAY]",
	},
	// Numéro de dossier médical
	{
		name: "fr_dossier_medical",
		re:   regexp.MustCompile(`(?i)dossier\s+médical\s*(?:n°\s*)?(\d{4,8})`),
		tag:  "[MEDICAL_RECORD]",
	},
	// Numéro de certificat médical
	{
		name: "fr_certificat_medical",
		re:   regexp.MustCompile(`(?i)certificat\s+médical\s+(?:du|n°\s*)(\d{2}[/\-.]\d{2}[/\-.]\d{4})`),
		tag:  "[MEDICAL_CERT]",
	},
}

// ─── UK Legal Patterns ─────────────────────────────────────────────────

var ukPatterns = []legalPattern{
	// Claim number: "HQ-2023-001234", "CR-2023-000123"
	{
		name: "uk_claim_number",
		re:   regexp.MustCompile(`(?i)(?:Claim\s+No[:\s]*)?([A-Z]{2,3}[-\s]\d{4}[-\s]\d{4,8})`),
		tag:  "[CASE_NO]",
	},
	// Judgment citation: "[2023] EWHC 1234", "[2023] EWCA Civ 567"
	{
		name: "uk_judgment_citation",
		re:   regexp.MustCompile(`\[\d{4}\]\s+(?:EWCA\s+(?:Civ|Crim)|EWHC|UKSC|EWFC)\s+\d+`),
		tag:  "[JUDGMENT_NO]",
	},
	// SRA ID: "SRA ID: 123456"
	{
		name: "uk_sra_id",
		re:   regexp.MustCompile(`(?i)SRA\s+ID[:\s]*(\d{4,8})`),
		tag:  "[LAWYER_ID]",
	},
	// Company number (exactly 8 digits, not preceded by other digits)
	{
		name: "uk_company_no",
		re:   regexp.MustCompile(`(?i)(?:Company\s+No[:\s]*)(\d{8})\b`),
		tag:  "[COMPANY_REG]",
	},
	// Title number (Land Registry)
	{
		name: "uk_title_number",
		re:   regexp.MustCompile(`(?i)Title\s+No[:\s]*([A-Z]{2}\d{4,8})`),
		tag:  "[LAND_TITLE]",
	},
	// Crime reference
	{
		name: "uk_crime_ref",
		re:   regexp.MustCompile(`(?i)Crime\s+Ref[:\s]*(\d{9,12})`),
		tag:  "[CRIME_REF]",
	},
	// NHS Number
	{
		name: "uk_nhs",
		re:   regexp.MustCompile(`(?i)NHS\s*(?:Number|No)?[:\s]*(\d{3}\s?\d{3}\s?\d{4})`),
		tag:  "[NHS_NUMBER]",
	},
	// National Insurance Number
	{
		name: "uk_nin",
		re:   regexp.MustCompile(`\b[A-CEDGHJ-PR-TW-Z]{2}\s?\d{2}\s?\d{2}\s?\d{2}\s?[A-D]\b`),
		tag:  "[NATIONAL_ID]",
	},
}

// ─── US Legal Patterns ─────────────────────────────────────────────────

var usPatterns = []legalPattern{
	// Federal case: "1:23-cv-01234", "23-cr-00456"
	{
		name: "us_federal_case",
		re:   regexp.MustCompile(`\b\d{1,2}[:\-]\d{2}[:\-](?:cv|cr|mj|bk|ap)\s*[:\-]\s*\d{4,6}\b`),
		tag:  "[CASE_NO]",
	},
	// Supreme Court: "No. 22-1234"
	{
		name: "us_scotus",
		re:   regexp.MustCompile(`(?i)No\.\s*\d{2}[-–]\d+`),
		tag:  "[CASE_NO]",
	},
	// Docket number
	{
		name: "us_docket",
		re:   regexp.MustCompile(`(?i)Docket\s+No[:\s]*(\d{2}[-–]\d{3,6})`),
		tag:  "[DOCKET_NO]",
	},
	// Slip opinion
	{
		name: "us_slip_opinion",
		re:   regexp.MustCompile(`(?i)Slip\s+Opinion\s+No[:\s]*(\d{2}[-–]\d{3,5})`),
		tag:  "[JUDGMENT_NO]",
	},
	// Bar number
	{
		name: "us_bar_number",
		re:   regexp.MustCompile(`(?i)Bar\s+No[:\s]*(\d{5,8})`),
		tag:  "[LAWYER_ID]",
	},
	// EIN
	{
		name: "us_ein",
		re:   regexp.MustCompile(`\b\d{2}-\d{7}\b`),
		tag:  "[TAX_ID]",
	},
	// Inmate number
	{
		name: "us_inmate",
		re:   regexp.MustCompile(`(?i)Inmate\s+No[:\s]*(\d{5,9})`),
		tag:  "[PRISONER_NO]",
	},
	// FBI number
	{
		name: "us_fbi",
		re:   regexp.MustCompile(`(?i)FBI\s+No[:\s]*(\d{9,10})`),
		tag:  "[CRIMINAL_RECORD]",
	},
	// Medicare (already covered by wuming, but explicit for legal context)
	{
		name: "us_medicare_legal",
		re:   regexp.MustCompile(`(?i)Medicare\s*(?:No|Number|ID)?[:\s]*([A-Z0-9]{10,12})`),
		tag:  "[MEDICARE_NO]",
	},
	// MRN (Medical Record Number)
	{
		name: "us_mrn",
		re:   regexp.MustCompile(`(?i)MRN[:\s]*(\d{7,12})`),
		tag:  "[MEDICAL_RECORD]",
	},
	// Patent: "US 12,345,678 B2"
	{
		name: "us_patent",
		re:   regexp.MustCompile(`(?i)US\s+[\d,]{7,12}\s*[A-Z]\d*`),
		tag:  "[PATENT_NO]",
	},
}

// ─── German Legal Patterns ─────────────────────────────────────────────

var dePatterns = []legalPattern{
	// Aktenzeichen: "12 O 123/23", "3 W 45/23"
	{
		name: "de_aktenzeichen",
		re:   regexp.MustCompile(`\b\d{1,4}\s+[A-Z]{1,4}\s+\d{1,6}/\d{2}\b`),
		tag:  "[CASE_NO]",
	},
	// Staatsanwaltschaft
	{
		name: "de_staatsanwaltschaft",
		re:   regexp.MustCompile(`(?i)Staatsanwaltschaft\s+(?:Az[:\s]*)?(\d{2,4}\s+Js\s+\d{2,6}/\d{2})`),
		tag:  "[PARQUET_NO]",
	},
	// Urteil
	{
		name: "de_urteil",
		re:   regexp.MustCompile(`(?i)Urteil\s+vom\s+(\d{2}\.\d{2}\.\d{4})`),
		tag:  "[JUDGMENT_NO]",
	},
	// Beschluss
	{
		name: "de_beschluss",
		re:   regexp.MustCompile(`(?i)Beschluss\s+(?:Az[:\s]*)?(\d{1,4}\s+[A-Z]\s+\d{1,6}/\d{2})`),
		tag:  "[JUDGMENT_NO]",
	},
	// Rechtsanwaltkammer
	{
		name: "de_rak",
		re:   regexp.MustCompile(`(?i)RAK\s*(?:Nr[:\s]*)?(\d{4,8})`),
		tag:  "[LAWYER_ID]",
	},
	// Handelsregister
	{
		name: "de_hrb",
		re:   regexp.MustCompile(`(?i)HRB\s+\d+\s+\(AG\s+\w+\)`),
		tag:  "[COMPANY_REG]",
	},
	// Grundbuch
	{
		name: "de_grundbuch",
		re:   regexp.MustCompile(`(?i)Grundbuch\s+(?:Blatt\s+)?(\d{4,8})`),
		tag:  "[LAND_TITLE]",
	},
	// Flurstück
	{
		name: "de_flurstueck",
		re:   regexp.MustCompile(`(?i)Flurstück\s+(\d{1,6}/\d{1,4})`),
		tag:  "[LAND_PARCEL]",
	},
	// Notar
	{
		name: "de_notar",
		re:   regexp.MustCompile(`(?i)Notar(?:vertrag)?\s*(?:Nr[:\s]*)?(\d{4,8})`),
		tag:  "[NOTARY_REF]",
	},
	// Gerichtsvollzieher
	{
		name: "de_gerichtsvollzieher",
		re:   regexp.MustCompile(`(?i)Gerichtsvollzieher\s*(?:Nr[:\s]*)?(\d{4,8})`),
		tag:  "[BAILIFF_ID]",
	},
}
