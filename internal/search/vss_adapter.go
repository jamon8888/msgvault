package search

import (
	"context"

	"github.com/wesm/msgvault/internal/embedding"
	"github.com/wesm/msgvault/internal/vector"
)

type VectorSearchAdapter struct {
	vs  vector.VectorStore
	emb embedding.Service
}

func NewVectorSearchAdapter(vs vector.VectorStore, emb embedding.Service) *VectorSearchAdapter {
	return &VectorSearchAdapter{vs: vs, emb: emb}
}

func (a *VectorSearchAdapter) Index(ctx context.Context, id int64, messageID int64, attachmentID int64, chunkIndex int, text string) error {
	emb, err := a.emb.Embed(text)
	if err != nil {
		return err
	}
	if err := a.vs.InsertVector(id, messageID, attachmentID, chunkIndex, emb); err != nil {
		return err
	}
	return a.vs.InsertText(id, messageID, attachmentID, chunkIndex, text)
}

func (a *VectorSearchAdapter) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	emb, err := a.emb.Embed(query)
	if err != nil {
		return nil, err
	}
	vecResults, err := a.vs.Search(emb, limit)
	if err != nil {
		return nil, err
	}
	results := make([]SearchResult, len(vecResults))
	for i, vr := range vecResults {
		results[i] = SearchResult{
			AttachmentID: vr.AttachmentID,
			MessageID:    vr.MessageID,
			ChunkIndex:   vr.ChunkIndex,
			ChunkText:    vr.ChunkText,
			Score:        vr.Distance,
		}
	}
	return results, nil
}

func (a *VectorSearchAdapter) GetChunksByAttachmentID(attachmentID int64) ([]string, error) {
	return a.vs.GetTextByAttachmentID(attachmentID)
}

func (a *VectorSearchAdapter) Delete(ctx context.Context, attachmentID int64) error {
	return nil
}

func (a *VectorSearchAdapter) Close() error {
	return a.vs.Close()
}
