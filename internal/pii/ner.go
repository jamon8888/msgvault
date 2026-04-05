package pii

import (
	"strings"

	"github.com/tsawler/prose/v3"
)

// entityTag maps prose entity labels to redaction tags.
var entityTag = map[string]string{
	"PERSON":      "[PERSON]",
	"ORG":         "[ORG]",
	"GPE":         "[GPE]",
	"MONEY":       "[MONEY]",
	"DATE":        "[DATE]",
	"TIME":        "[TIME]",
	"PERCENT":     "[PERCENT]",
	"FAC":         "[FACILITY]",
	"PRODUCT":     "[PRODUCT]",
	"EVENT":       "[EVENT]",
	"WORK_OF_ART": "[WORK_OF_ART]",
	"LANGUAGE":    "[LANGUAGE]",
	"NORP":        "[NORP]",
	"LAW":         "[LAW]",
	"ORDINAL":     "[ORDINAL]",
	"CARDINAL":    "[CARDINAL]",
}

type entitySpan struct {
	start int
	end   int
	tag   string
}

// NERDetector performs named entity recognition based redaction
// using the prose library (pure Go, no CGO).
type NERDetector struct {
	// enabled entity types (nil = all enabled)
	enabled map[string]bool
}

// NewNERDetector creates a NER-based PII detector.
// If enabledTypes is empty, all entity types are detected.
func NewNERDetector(enabledTypes ...string) *NERDetector {
	d := &NERDetector{}
	if len(enabledTypes) > 0 {
		d.enabled = make(map[string]bool, len(enabledTypes))
		for _, t := range enabledTypes {
			d.enabled[strings.ToUpper(t)] = true
		}
	}
	return d
}

// DetectAndReplace finds named entities in text and replaces them with tags.
func (d *NERDetector) DetectAndReplace(text string) string {
	if text == "" {
		return text
	}

	doc, err := prose.NewDocument(text)
	if err != nil {
		return text
	}

	entities := doc.Entities()
	if len(entities) == 0 {
		return text
	}

	var spans []entitySpan
	for _, ent := range entities {
		tag, ok := entityTag[ent.Label]
		if !ok {
			continue
		}
		if d.enabled != nil && !d.enabled[ent.Label] {
			continue
		}
		idx := strings.Index(text, ent.Text)
		if idx == -1 {
			continue
		}
		spans = append(spans, entitySpan{
			start: idx,
			end:   idx + len(ent.Text),
			tag:   tag,
		})
	}

	if len(spans) == 0 {
		return text
	}

	sortEntitySpans(spans)

	result := text
	for _, s := range spans {
		result = result[:s.start] + s.tag + result[s.end:]
	}

	return result
}

func sortEntitySpans(spans []entitySpan) {
	for i := 0; i < len(spans); i++ {
		for j := i + 1; j < len(spans); j++ {
			if spans[j].start > spans[i].start {
				spans[i], spans[j] = spans[j], spans[i]
			}
		}
	}
}
