//go:build windows
// +build windows

package vector

import "errors"

var errVectorStoreNotAvailable = errors.New("vector store not available on Windows (requires native DuckDB)")

type VectorStore interface {
	InitSchema() error
	InsertVector(id int64, messageID int64, attachmentID int64, chunkIndex int, embedding []float64) error
	InsertText(id int64, messageID int64, attachmentID int64, chunkIndex int, chunkText string) error
	Search(query []float64, limit int) ([]SearchResult, error)
	GetTextByAttachmentID(attachmentID int64) ([]string, error)
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

type WindowsDuckDBStore struct{}

func NewDuckDBStore(dsn string) (*WindowsDuckDBStore, error) {
	return nil, errVectorStoreNotAvailable
}

func (s *WindowsDuckDBStore) InitSchema() error {
	return errVectorStoreNotAvailable
}

func (s *WindowsDuckDBStore) InsertVector(id int64, messageID int64, attachmentID int64, chunkIndex int, embedding []float64) error {
	return errVectorStoreNotAvailable
}

func (s *WindowsDuckDBStore) InsertText(id int64, messageID int64, attachmentID int64, chunkIndex int, chunkText string) error {
	return errVectorStoreNotAvailable
}

func (s *WindowsDuckDBStore) Search(query []float64, limit int) ([]SearchResult, error) {
	return nil, errVectorStoreNotAvailable
}

func (s *WindowsDuckDBStore) GetTextByAttachmentID(attachmentID int64) ([]string, error) {
	return nil, errVectorStoreNotAvailable
}

func (s *WindowsDuckDBStore) Close() error {
	return nil
}
