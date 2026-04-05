package search

import (
	"context"
	"database/sql"
	"math"
	"sort"
	"strings"
	"sync"
)

type BM25Store struct {
	mu       sync.RWMutex
	db       *sql.DB
	chunks   map[int64]chunk
	docOrder []int64
	index    *bm25Index
	nextID   int64
}

type scoredDoc struct {
	bm25Idx int
	score   float64
}

type chunk struct {
	messageID    int64
	attachmentID int64
	chunkIndex   int
	text         string
}

// bm25Index is a lightweight inline BM25 Okapi implementation with smoothed IDF.
// Unlike the crawlab-team/bm25 library, this uses log(1 + (N-n+0.5)/(n+0.5))
// instead of log((N-n+0.5)/(n+0.5)), guaranteeing non-zero scores even when
// terms appear in all documents.
type bm25Index struct {
	corpus     [][]string
	docLens    []int
	avgDocLen  float64
	docFreq    map[string]int
	tokenCount int
	k1         float64
	b          float64
}

func newBM25Index(docs []string, tokenize func(string) []string, k1, b float64) *bm25Index {
	idx := &bm25Index{
		corpus:  make([][]string, len(docs)),
		docLens: make([]int, len(docs)),
		docFreq: make(map[string]int),
		k1:      k1,
		b:       b,
	}

	var totalLen int
	for i, doc := range docs {
		tokens := tokenize(doc)
		idx.corpus[i] = tokens
		idx.docLens[i] = len(tokens)
		totalLen += len(tokens)
	}

	if len(docs) > 0 {
		idx.avgDocLen = float64(totalLen) / float64(len(docs))
	}
	idx.tokenCount = len(docs)

	// Compute document frequency (number of docs containing each term, not total occurrences)
	for _, tokens := range idx.corpus {
		seen := make(map[string]bool)
		for _, t := range tokens {
			if !seen[t] {
				idx.docFreq[t]++
				seen[t] = true
			}
		}
	}

	return idx
}

// idf computes the smoothed IDF for a term.
// Standard: log((N - n + 0.5) / (n + 0.5)) → 0 when n == N
// Smoothed: log(1 + (N - n + 0.5) / (n + 0.5)) → always > 0
func (idx *bm25Index) idf(term string) float64 {
	n := idx.docFreq[term]
	N := idx.tokenCount
	return math.Log(1.0 + (float64(N)-float64(n)+0.5)/(float64(n)+0.5))
}

// scores computes BM25 Okapi scores for a tokenized query against all documents.
func (idx *bm25Index) scores(query []string) []float64 {
	scores := make([]float64, len(idx.corpus))
	for _, q := range query {
		idf := idx.idf(q)
		if idf == 0 {
			continue
		}
		for i, doc := range idx.corpus {
			tf := 0
			for _, t := range doc {
				if t == q {
					tf++
				}
			}
			if tf == 0 {
				continue
			}
			docLen := float64(idx.docLens[i])
			k := idx.k1 * (1.0 - idx.b + idx.b*docLen/idx.avgDocLen)
			scores[i] += idf * (float64(tf) / (float64(tf) + k))
		}
	}
	return scores
}

func NewBM25Store(db *sql.DB) (*BM25Store, error) {
	s := &BM25Store{
		db:       db,
		chunks:   make(map[int64]chunk),
		docOrder: make([]int64, 0),
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS attachment_chunks (
			id INTEGER PRIMARY KEY,
			message_id INTEGER NOT NULL,
			attachment_id INTEGER NOT NULL,
			chunk_index INTEGER NOT NULL,
			chunk_text TEXT NOT NULL
		)
	`); err != nil {
		return nil, err
	}

	if err := s.loadFromDB(); err != nil {
		return nil, err
	}

	s.rebuildIndex()
	return s, nil
}

func (s *BM25Store) loadFromDB() error {
	rows, err := s.db.Query(`SELECT id, message_id, attachment_id, chunk_index, chunk_text FROM attachment_chunks ORDER BY id`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var c chunk
		if err := rows.Scan(&id, &c.messageID, &c.attachmentID, &c.chunkIndex, &c.text); err != nil {
			return err
		}
		s.chunks[id] = c
		s.docOrder = append(s.docOrder, id)
		if id >= s.nextID {
			s.nextID = id + 1
		}
	}
	return rows.Err()
}

func (s *BM25Store) Index(_ context.Context, id int64, messageID int64, attachmentID int64, chunkIndex int, text string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	docID := s.nextID
	s.nextID++

	if _, err := s.db.Exec(
		`INSERT INTO attachment_chunks (id, message_id, attachment_id, chunk_index, chunk_text) VALUES (?, ?, ?, ?, ?)`,
		docID, messageID, attachmentID, chunkIndex, text,
	); err != nil {
		return err
	}

	s.chunks[docID] = chunk{
		messageID:    messageID,
		attachmentID: attachmentID,
		chunkIndex:   chunkIndex,
		text:         text,
	}
	s.docOrder = append(s.docOrder, docID)
	return nil
}

func (s *BM25Store) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rebuildIndex()
	return nil
}

func (s *BM25Store) Search(_ context.Context, query string, limit int) ([]SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.index == nil {
		return nil, nil
	}

	terms := bm25Tokenize(query)
	if len(terms) == 0 {
		return nil, nil
	}

	scores := s.index.scores(terms)

	var scored []scoredDoc
	for i, score := range scores {
		if score > 0 {
			scored = append(scored, scoredDoc{i, score})
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	if limit > 0 && len(scored) > limit {
		scored = scored[:limit]
	}

	results := make([]SearchResult, len(scored))
	for i, sd := range scored {
		docID := s.docOrder[sd.bm25Idx]
		c := s.chunks[docID]
		results[i] = SearchResult{
			AttachmentID: c.attachmentID,
			MessageID:    c.messageID,
			ChunkIndex:   c.chunkIndex,
			ChunkText:    c.text,
			Score:        sd.score,
		}
	}

	return results, nil
}

func (s *BM25Store) GetChunksByAttachmentID(attachmentID int64) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(
		`SELECT chunk_text FROM attachment_chunks WHERE attachment_id = ? ORDER BY chunk_index`,
		attachmentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var texts []string
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err != nil {
			return nil, err
		}
		texts = append(texts, text)
	}
	return texts, rows.Err()
}

func (s *BM25Store) Delete(_ context.Context, attachmentID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.db.Exec(`DELETE FROM attachment_chunks WHERE attachment_id = ?`, attachmentID); err != nil {
		return err
	}

	for docID, c := range s.chunks {
		if c.attachmentID == attachmentID {
			delete(s.chunks, docID)
		}
	}

	s.rebuildIndex()
	return nil
}

func (s *BM25Store) Close() error {
	return nil
}

func (s *BM25Store) rebuildIndex() {
	var texts []string
	for _, docID := range s.docOrder {
		if c, ok := s.chunks[docID]; ok {
			texts = append(texts, c.text)
		}
	}

	var newOrder []int64
	for _, docID := range s.docOrder {
		if _, ok := s.chunks[docID]; ok {
			newOrder = append(newOrder, docID)
		}
	}
	s.docOrder = newOrder

	if len(texts) == 0 {
		s.index = nil
		return
	}

	// k1=1.5 (TF saturation), b=0.6 (reduced length normalization for short docs)
	s.index = newBM25Index(texts, bm25Tokenize, 1.5, 0.6)
}

func bm25Tokenize(text string) []string {
	text = strings.ToLower(text)
	words := strings.FieldsFunc(text, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == ',' || r == '.' || r == '!' || r == '?' || r == ';' || r == ':' || r == '(' || r == ')' || r == '[' || r == ']' || r == '{' || r == '}' || r == '"' || r == '\''
	})
	var result []string
	for _, w := range words {
		if len(w) > 0 {
			result = append(result, w)
		}
	}
	return result
}
