package main

import (
	"context"
	"fmt"

	"github.com/wesm/msgvault/internal/pii"
)

func main() {
	cfg := &pii.Config{
		LegalMode:     true,
		NERMode:       true,
		Jurisdictions: []string{"fr", "uk", "us", "de"},
	}

	filter, err := pii.NewFilter(cfg)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	ctx := context.Background()

	tests := []string{
		// PII structurées
		"Contactez-moi à jean.dupont@gmail.com ou au 06 12 34 56 78",
		"Mon IBAN: FR76 3000 6000 0112 3456 7890 189",
		"Ma carte bleue: 4111 1111 1111 1111",
		"Mon NIR: 1 85 12 75 112 089 32",

		// Juridique français
		"Dans l'affaire RG 23/12345, Maître Dupont, avocat au barreau de Paris, représente la société ABC.",
		"Le jugement n° 2023/456 condamne le défendeur à verser € 50 000 de dommages-intérêts.",
		"CARPA n° 1234567890, RCS Paris B 123 456 789, SIRET 12345678901234",
		"Section AB n° 1234, hypothèque n° 567890",

		// Juridique UK
		"In the matter of [2023] EWHC 1234, SRA ID: 123456",

		// Juridique US
		"Case 1:23-cv-01234, Bar No: 123456, EIN: 12-3456789",

		// Juridique Allemagne
		"Aktenzeichen 12 O 123/23, HRB 12345 (AG Berlin)",

		// Texte réel de bordereau Vinted
		"Dimensions max de mon colis Poids = 30 kg max VINTED USER 75011 PARIS",

		// Test complet mixte
		"Maître Martin (avocat au barreau de Lyon) traite l'affaire RG 24/56789. " +
			"Son client, M. Bernard (NIR: 1 90 01 75 112 089 32), " +
			"réclame € 15 000 de dommages. Contact: martin@cabinet-legal.fr, 04 78 12 34 56.",
	}

	fmt.Println("=== PIPELINE PII 3 PASSES ===")
	fmt.Println("Pass 1: wuming (PII structurées)")
	fmt.Println("Pass 2: prose NER (entités nommées)")
	fmt.Println("Pass 3: legal regex (juridique FR/UK/US/DE)")
	fmt.Println()

	for i, input := range tests {
		output, err := filter.FilterString(ctx, input)
		if err != nil {
			fmt.Printf("[%d] ERROR: %v\n", i+1, err)
			continue
		}

		fmt.Printf("--- Test %d ---\n", i+1)
		fmt.Printf("IN:  %s\n", input)
		fmt.Printf("OUT: %s\n", output)

		if input == output {
			fmt.Printf("⚠️  AUCUN CHANGEMENT (PII non détectée)\n")
		} else {
			fmt.Printf("✅ PII détectée et masquée\n")
		}
		fmt.Println()
	}
}
