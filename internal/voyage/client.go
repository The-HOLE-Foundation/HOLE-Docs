package voyage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"
)

// Client handles communication with Voyage AI API
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewClient constructs a Voyage AI client configured from environment and sensible defaults.
// It reads the VOYAGEAI_API_KEY environment variable and logs a warning if it is not set, sets the API base URL to https://api.voyageai.com/v1, and configures the HTTP client with a 30 second timeout.
func NewClient(logger *zap.Logger) *Client {
	apiKey := os.Getenv("VOYAGEAI_API_KEY")
	if apiKey == "" {
		logger.Warn("VOYAGEAI_API_KEY not set, embeddings will be disabled")
	}

	return &Client{
		apiKey:  apiKey,
		baseURL: "https://api.voyageai.com/v1",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// EmbeddingRequest represents a request to Voyage AI
type EmbeddingRequest struct {
	Input     []string `json:"input"`
	Model     string   `json:"model"`
	InputType string   `json:"input_type,omitempty"`
	Truncation bool     `json:"truncation,omitempty"`
}

// EmbeddingResponse represents the response from Voyage AI
type EmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
	} `json:"usage"`
}

// Embed generates embeddings for the given texts using voyage-large-4
func (c *Client) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("VOYAGEAI_API_KEY not configured")
	}

	c.logger.Info("Embedding texts with Voyage AI",
		zap.Int("count", len(texts)),
		zap.String("model", "voyage-4-large"))

	req := EmbeddingRequest{
		Input:      texts,
		Model:      "voyage-4-large",
		InputType:  "document", // Voyage AI optimizes differently for query vs document
		Truncation: true,        // Auto-truncate if texts are too long
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fmt.Sprintf("%s/embeddings", c.baseURL),
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	var embResp EmbeddingResponse
	if err := json.Unmarshal(body, &embResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Validate embeddings response is non-empty
	if len(embResp.Data) == 0 {
		return nil, fmt.Errorf("API returned no embeddings")
	}

	// Extract embeddings in order with bounds checking
	embeddings := make([][]float32, len(embResp.Data))
	for _, item := range embResp.Data {
		if item.Index < 0 || item.Index >= len(embeddings) {
			return nil, fmt.Errorf("API returned invalid embedding index: %d (expected 0-%d)", item.Index, len(embeddings)-1)
		}
		embeddings[item.Index] = item.Embedding
	}

	// Final safety check - verify all embeddings are populated and non-empty
	for i, emb := range embeddings {
		if emb == nil || len(emb) == 0 {
			return nil, fmt.Errorf("API returned nil or empty embedding at index %d", i)
		}
	}

	c.logger.Info("Embeddings generated successfully",
		zap.Int("count", len(embeddings)),
		zap.Int("dimensions", len(embeddings[0])),
		zap.Int("prompt_tokens", embResp.Usage.PromptTokens))

	return embeddings, nil
}

// EmbedQuery generates an embedding for a query (optimized for search)
func (c *Client) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("VOYAGEAI_API_KEY not configured")
	}

	c.logger.Info("Embedding query with Voyage AI",
		zap.String("model", "voyage-4-large"))

	req := EmbeddingRequest{
		Input:      []string{query},
		Model:      "voyage-4-large",
		InputType:  "query", // Query optimization
		Truncation: true,
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fmt.Sprintf("%s/embeddings", c.baseURL),
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	var embResp EmbeddingResponse
	if err := json.Unmarshal(body, &embResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(embResp.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	c.logger.Info("Query embedding generated",
		zap.Int("dimensions", len(embResp.Data[0].Embedding)))

	return embResp.Data[0].Embedding, nil
}

// IsConfigured returns true if the API key is set
func (c *Client) IsConfigured() bool {
	return c.apiKey != ""
}