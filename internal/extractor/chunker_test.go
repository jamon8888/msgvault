package extractor

import (
	"math"
	"testing"
)

func TestChunker(t *testing.T) {
	text := "This is a long text " + string(make([]byte, 9000))
	t.Logf("Text length: %d", len(text))
	chunks := ChunkText(text, 8192, 512)
	t.Logf("Got %d chunks", len(chunks))
	for i, c := range chunks {
		t.Logf("Chunk %d: %d chars", i, len(c))
	}

	if len(chunks) != 2 {
		t.Errorf("Expected 2 chunks, got %d", len(chunks))
	}

	if len(chunks) > 1 {
		overlapLen := int(math.Min(512, float64(len(chunks[0]))))
		if chunks[0][len(chunks[0])-overlapLen:] != chunks[1][:overlapLen] {
			t.Error("Expected overlap between chunks")
		}
	}
}
