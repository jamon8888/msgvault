package embedding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type OllamaClient struct {
	baseURL string
	client  *http.Client
}

type EmbedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type EmbedResponse struct {
	Model     string    `json:"model"`
	Embedding []float64 `json:"embedding"`
}

func NewOllamaClient(baseURL string) *OllamaClient {
	return &OllamaClient{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

func (c *OllamaClient) Embed(model, text string) ([]float64, error) {
	reqBody, _ := json.Marshal(EmbedRequest{Model: model, Prompt: text})
	resp, err := c.client.Post(c.baseURL+"/api/embeddings", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama error: %s", resp.Status)
	}

	var result EmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Embedding, nil
}
