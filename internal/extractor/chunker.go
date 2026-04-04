package extractor

func ChunkText(text string, chunkSize, overlap int) []string {
	if len(text) <= chunkSize {
		return []string{text}
	}

	var chunks []string
	start := 0

	for start < len(text) {
		end := start + chunkSize
		if end > len(text) {
			end = len(text)
		}
		chunks = append(chunks, text[start:end])
		if end >= len(text) {
			break
		}
		start = end - overlap
		if start < 0 {
			start = 0
		}
	}

	return chunks
}
