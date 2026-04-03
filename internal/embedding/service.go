package embedding

type Service interface {
	Embed(text string) ([]float64, error)
}

type EmbeddingService struct {
	client *OllamaClient
	model  string
}

func NewEmbeddingService(client *OllamaClient, model string) *EmbeddingService {
	return &EmbeddingService{
		client: client,
		model:  model,
	}
}

func (s *EmbeddingService) Embed(text string) ([]float64, error) {
	return s.client.Embed(s.model, text)
}
