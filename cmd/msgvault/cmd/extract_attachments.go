package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wesm/msgvault/internal/embedding"
	"github.com/wesm/msgvault/internal/export"
	"github.com/wesm/msgvault/internal/extractor"
	"github.com/wesm/msgvault/internal/query"
	"github.com/wesm/msgvault/internal/search"
	"github.com/wesm/msgvault/internal/store"
	"github.com/wesm/msgvault/internal/vector"
)

var extractLimit int
var extractReprocess bool
var extractFormat string

var extractAttachmentsCmd = &cobra.Command{
	Use:   "extract-attachments",
	Short: "Extract text from unprocessed attachments for semantic search",
	Long: `Extract and index text content from attachments for semantic vector search.

This command processes attachments that haven't been indexed yet, extracts their
text content, generates embeddings, and stores them in the vector database.

The embedding and vector services must be enabled in config.toml:
  [embedding]
  enabled = true
  model = "nomic-embed-text"
  ollama_url = "http://localhost:11434"

  [vector]
  store = "duckdb"

Examples:
  msgvault extract-attachments                    # Extract up to 100 attachments
  msgvault extract-attachments --limit 50         # Extract only 50
  msgvault extract-attachments --reprocess         # Re-extract already indexed
  msgvault extract-attachments --format pdf,docx   # Only PDF and DOCX files`,
	RunE: runExtractAttachments,
}

func init() {
	rootCmd.AddCommand(extractAttachmentsCmd)
	extractAttachmentsCmd.Flags().IntVarP(&extractLimit, "limit", "l", 100, "Max attachments to process")
	extractAttachmentsCmd.Flags().BoolVar(&extractReprocess, "reprocess", false, "Re-extract already indexed attachments")
	extractAttachmentsCmd.Flags().StringVar(&extractFormat, "format", "pdf,docx,txt", "Comma-separated list of formats to process")
}

func runExtractAttachments(cmd *cobra.Command, args []string) error {
	// Check if embedding is enabled (only needed for Ollama provider)
	if cfg.Embedding.Provider == "ollama" && !cfg.Embedding.Enabled {
		return fmt.Errorf("embedding not enabled for ollama provider. Add to config.toml:\n\n[embedding]\nenabled = true\nprovider = \"ollama\"\nmodel = \"nomic-embed-text\"\nollama_url = \"http://localhost:11434\"")
	}
	// BM25 works without any external service

	// Open database
	dbPath := cfg.DatabaseDSN()
	s, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() { _ = s.Close() }()

	engine := query.NewSQLiteEngine(s.DB())

	// Get attachments to process
	filter := query.AttachmentFilter{
		WithTextOnly: !extractReprocess,
		Limit:        extractLimit,
	}

	if extractFormat != "" {
		filter.Formats = parseFormats(extractFormat)
	}

	attachments, err := engine.ListAttachments(cmd.Context(), filter)
	if err != nil {
		return fmt.Errorf("list attachments: %w", err)
	}

	if len(attachments) == 0 {
		fmt.Println("No attachments to process.")
		return nil
	}

	fmt.Printf("Processing %d attachments...\n", len(attachments))

	// Initialize search store based on config
	var searchStore search.SearchStore

	if cfg.Embedding.Provider == "ollama" {
		ollamaClient := embedding.NewOllamaClient(cfg.Embedding.OllamaURL)
		embeddingSvc := embedding.NewEmbeddingService(ollamaClient, cfg.Embedding.Model)
		vectorDir := filepath.Join(cfg.Data.DataDir, "vector.duckdb")
		vectorSvc, err := vector.NewDuckDBStore(vectorDir)
		if err != nil {
			return fmt.Errorf("open vector store: %w", err)
		}
		defer func() { _ = vectorSvc.Close() }()
		if err := vectorSvc.InitSchema(); err != nil {
			return fmt.Errorf("init vector schema: %w", err)
		}
		searchStore = search.NewVectorSearchAdapter(vectorSvc, embeddingSvc)
	} else {
		// BM25 uses the same SQLite DB
		searchStore, err = search.NewBM25Store(s.DB())
		if err != nil {
			return fmt.Errorf("init BM25 store: %w", err)
		}
	}
	defer searchStore.Close()

	// Initialize extractor
	extractorSvc := &extractor.ExtractorService{}

	// Process attachments
	processed := 0
	failed := 0

	for _, att := range attachments {
		if err := processAttachment(cmd.Context(), extractorSvc, searchStore, att, cfg.AttachmentsDir()); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process attachment %d: %v\n", att.ID, err)
			failed++
			continue
		}
		processed++
		if processed%10 == 0 {
			fmt.Printf("Processed %d/%d\n", processed, len(attachments))
		}
	}

	fmt.Printf("\nDone: %d processed, %d failed\n", processed, failed)
	return nil
}

func parseFormats(f string) []string {
	if f == "" {
		return nil
	}
	parts := strings.Split(f, ",")
	var formats []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			formats = append(formats, p)
		}
	}
	return formats
}

func processAttachment(ctx context.Context, extractorSvc extractor.Service, searchStore search.SearchStore, att query.AttachmentInfo, attachmentsDir string) error {
	// Get format from mime type
	format := mimeTypeToFormat(att.MimeType)
	if format == "" {
		return fmt.Errorf("unsupported format: %s", att.MimeType)
	}

	// Read attachment file
	data, err := readAttachmentForExtraction(attachmentsDir, att.ContentHash)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	// Write to temp file
	tmpFile, err := os.CreateTemp("", "att-*."+format)
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return fmt.Errorf("write temp: %w", err)
	}
	tmpFile.Close()

	// Extract text
	text, err := extractorSvc.Extract(format, tmpPath)
	if err != nil {
		return fmt.Errorf("extract: %w", err)
	}

	if text == "" {
		return nil
	}

	// Chunk text
	chunks := extractor.ChunkText(text, 1000, 200)
	if len(chunks) == 0 {
		return nil
	}

	// Index chunks
	for i, chunk := range chunks {
		if err := searchStore.Index(ctx, int64(i), 0, att.ID, i, chunk); err != nil {
			return fmt.Errorf("index chunk %d: %w", i, err)
		}
	}

	// Flush after all chunks indexed for this attachment
	if flusher, ok := searchStore.(interface{ Flush() error }); ok {
		if err := flusher.Flush(); err != nil {
			return fmt.Errorf("flush: %w", err)
		}
	}

	return nil
}

func readAttachmentForExtraction(attachmentsDir, contentHash string) ([]byte, error) {
	filePath, err := export.StoragePath(attachmentsDir, contentHash)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	const maxSize = 50 * 1024 * 1024 // 50MB
	if info.Size() > maxSize {
		return nil, fmt.Errorf("file too large: %d bytes", info.Size())
	}

	data := make([]byte, info.Size())
	if _, err := f.Read(data); err != nil {
		return nil, err
	}

	return data, nil
}

func mimeTypeToFormat(mimeType string) string {
	switch {
	case strings.Contains(mimeType, "pdf"):
		return "pdf"
	case strings.Contains(mimeType, "wordprocessingml.document"):
		return "docx"
	case strings.Contains(mimeType, "text/"):
		return "txt"
	default:
		return ""
	}
}
