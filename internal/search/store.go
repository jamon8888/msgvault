// Package search provides Gmail-like search query parsing and storage.
package search

import "context"

// SearchStore defines the interface for attachment text search backends.
// Implementations include BM25 (pure Go, default) and DuckDB VSS (optional).
type SearchStore interface {
	// Index stores a text chunk for a given message and attachment.
	Index(ctx context.Context, id int64, messageID int64, attachmentID int64, chunkIndex int, text string) error

	// Search returns chunks matching the query, ranked by relevance.
	Search(ctx context.Context, query string, limit int) ([]SearchResult, error)

	// GetChunksByAttachmentID returns all chunk texts for a given attachment.
	GetChunksByAttachmentID(attachmentID int64) ([]string, error)

	// Delete removes all chunks associated with an attachment.
	Delete(ctx context.Context, attachmentID int64) error

	// Close releases any resources held by the store.
	Close() error
}

// SearchResult represents a single matching text chunk from a search.
type SearchResult struct {
	AttachmentID int64
	MessageID    int64
	ChunkIndex   int
	ChunkText    string
	Score        float64 // BM25: TF-IDF-like; VSS: cosine distance. Relative, not absolute.
}
