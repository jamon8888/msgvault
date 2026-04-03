//go:build !windows
// +build !windows

package vector

import (
	"database/sql"
	"fmt"

	_ "github.com/marcboeker/go-duckdb"
)

type VectorStore interface {
	InitSchema() error
	InsertVector(id int64, messageID int64, attachmentID int64, chunkIndex int, embedding []float64) error
	InsertText(id int64, messageID int64, attachmentID int64, chunkIndex int, chunkText string) error
	Search(query []float64, limit int) ([]SearchResult, error)
	Close() error
}

type SearchResult struct {
	ID           int64
	MessageID    int64
	AttachmentID int64
	ChunkIndex   int
	ChunkText    string
	Distance     float64
}

type DuckDBStore struct {
	db *sql.DB
}

func NewDuckDBStore(dsn string) (*DuckDBStore, error) {
	db, err := sql.Open("duckdb", dsn)
	if err != nil {
		return nil, err
	}
	return &DuckDBStore{db: db}, nil
}

func (s *DuckDBStore) InitSchema() error {
	_, err := s.db.Exec("INSTALL vss; LOAD vss;")
	if err != nil {
		return fmt.Errorf("failed to load vss: %w", err)
	}

	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS attachment_vectors (
			id BIGINT,
			message_id BIGINT,
			attachment_id BIGINT,
			chunk_index INTEGER,
			embedding FLOAT[]
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	_, err = s.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_ann ON attachment_vectors USING HNSW (embedding);
	`)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS attachment_text (
			id BIGINT,
			message_id BIGINT,
			attachment_id BIGINT,
			chunk_index INTEGER,
			chunk_text TEXT
		);
	`)
	return err
}

func (s *DuckDBStore) InsertVector(id int64, messageID int64, attachmentID int64, chunkIndex int, embedding []float64) error {
	_, err := s.db.Exec(`
		INSERT INTO attachment_vectors (id, message_id, attachment_id, chunk_index, embedding)
		VALUES (?, ?, ?, ?, ?)
	`, id, messageID, attachmentID, chunkIndex, embedding)
	return err
}

func (s *DuckDBStore) InsertText(id int64, messageID int64, attachmentID int64, chunkIndex int, chunkText string) error {
	_, err := s.db.Exec(`
		INSERT INTO attachment_text (id, message_id, attachment_id, chunk_index, chunk_text)
		VALUES (?, ?, ?, ?, ?)
	`, id, messageID, attachmentID, chunkIndex, chunkText)
	return err
}

func (s *DuckDBStore) Search(query []float64, limit int) ([]SearchResult, error) {
	rows, err := s.db.Query(`
		SELECT v.id, v.message_id, v.attachment_id, v.chunk_index, t.chunk_text,
			   array_distance(v.embedding, ?::float[]) as distance
		FROM attachment_vectors v
		JOIN attachment_text t ON v.id = t.id
		ORDER BY distance
		LIMIT ?
	`, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ID, &r.MessageID, &r.AttachmentID, &r.ChunkIndex, &r.ChunkText, &r.Distance); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, nil
}

func (s *DuckDBStore) Close() error {
	return s.db.Close()
}
