package embedding

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type OllamaClient struct {
	baseURL string
	client  *http.Client
}

type EmbedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type EmbedResponse struct {
	Model      string      `json:"model"`
	Embeddings [][]float64 `json:"embeddings"`
}

func NewOllamaClient(baseURL string) *OllamaClient {
	return &OllamaClient{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

func (c *OllamaClient) Embed(model, text string) ([]float64, error) {
	reqBody, _ := json.Marshal(EmbedRequest{Model: model, Input: text})
	resp, err := c.client.Post(c.baseURL+"/api/embed", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result EmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Embeddings) == 0 {
		return nil, nil
	}
	return result.Embeddings[0], nil
}
