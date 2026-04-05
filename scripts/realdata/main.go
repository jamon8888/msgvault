package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/wesm/msgvault/internal/pii"
)

func main() {
	dbPath := os.Getenv("MSGVAULT_DB")
	if dbPath == "" {
		dbPath = "C:\\Users\\NMarchitecte\\.msgvault\\msgvault.db"
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		fmt.Printf("Error opening DB: %v\n", err)
		return
	}
	defer db.Close()

	rows, err := db.Query(`SELECT attachment_id, chunk_text FROM attachment_chunks ORDER BY attachment_id, chunk_index`)
	if err != nil {
		fmt.Printf("Error querying chunks: %v\n", err)
		return
	}
	defer rows.Close()

	type chunk struct {
		attID int64
		text  string
	}
	var chunks []chunk
	for rows.Next() {
		var c chunk
		if err := rows.Scan(&c.attID, &c.text); err != nil {
			continue
		}
		chunks = append(chunks, c)
	}

	fmt.Printf("Loaded %d chunks from %d attachments\n\n", len(chunks), func() int {
		m := make(map[int64]bool)
		for _, c := range chunks {
			m[c.attID] = true
		}
		return len(m)
	}())

	filter, err := pii.NewFilter(&pii.Config{
		LegalMode:     true,
		NERMode:       true,
		Jurisdictions: []string{"fr", "uk", "us", "de"},
	})
	if err != nil {
		fmt.Printf("Error creating filter: %v\n", err)
		return
	}

	ctx := context.Background()

	totalPII := 0
	totalClean := 0

	for _, c := range chunks {
		output, err := filter.FilterString(ctx, c.text)
		if err != nil {
			fmt.Printf("att=%d ERROR: %v\n", c.attID, err)
			continue
		}

		if output != c.text {
			totalPII++
			fmt.Printf("=== Attachment %d (PII DETECTED) ===\n", c.attID)
			fmt.Printf("IN:  %s\n", truncate(c.text, 120))
			fmt.Printf("OUT: %s\n", truncate(output, 120))
			fmt.Println()
		} else {
			totalClean++
			fmt.Printf("=== Attachment %d (clean) ===\n", c.attID)
			fmt.Printf("    %s\n", truncate(c.text, 100))
			fmt.Println()
		}
	}

	fmt.Printf("\n=== SUMMARY ===\n")
	fmt.Printf("Total chunks: %d\n", len(chunks))
	fmt.Printf("With PII:     %d\n", totalPII)
	fmt.Printf("Clean:        %d\n", totalClean)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
