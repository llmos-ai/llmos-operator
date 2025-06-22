package vectorizer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/sirupsen/logrus"
)

// EmbeddingRequest represents the request structure for the embedding API
type EmbeddingRequest struct {
	Input string `json:"input"`
	Model string `json:"model"`
}

// EmbeddingResponse represents the response structure from the embedding API
type EmbeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

// CustomVectorizer handles embedding generation
type CustomVectorizer struct {
	Host   string
	Scheme string
}

func NewCustomVectorizer(host, scheme string) *CustomVectorizer {
	return &CustomVectorizer{
		Host:   host,
		Scheme: scheme,
	}
}

func (cv *CustomVectorizer) GetVector(text string) ([]float32, error) {
	req := map[string]interface{}{
		"input": text,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	embeddingURL := fmt.Sprintf("%s://%s/v1/embeddings", cv.Scheme, cv.Host)
	resp, err := http.Post(embeddingURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to call embedding API: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logrus.Errorf("failed to close response body: %v", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding API returned status %d: %s", resp.StatusCode, string(body))
	}

	var embeddingResp EmbeddingResponse
	if err := json.Unmarshal(body, &embeddingResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if len(embeddingResp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data returned")
	}

	return embeddingResp.Data[0].Embedding, nil
}
