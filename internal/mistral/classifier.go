package mistral

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

// Classification result
type DocumentClassification struct {
	DocumentType  string  `json:"document_type"`
	Category      string  `json:"category"`
	Confidence    float64 `json:"confidence"`
	Reasoning     string  `json:"reasoning"`
}

// ChatRequest for Mistral chat completion
type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
}

// ChatMessage represents a chat message
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse from Mistral
type ChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// ClassifyDocument uses Mistral chat to classify document type and category
func (c *Client) ClassifyDocument(ctx context.Context, ocrText string) (*DocumentClassification, error) {
	// Take first 2000 characters for classification
	preview := ocrText
	if len(preview) > 2000 {
		preview = preview[:2000]
	}

	c.logger.Info("Classifying document with Mistral",
		zap.Int("text_length", len(preview)))

	// Build classification prompt
	prompt := fmt.Sprintf(`Analyze this document excerpt and classify it.

Document excerpt:
%s

Respond with ONLY a JSON object (no markdown, no explanation) with this exact structure:
{
  "document_type": "<one of: police_report, arrest_record, incident_report, legal_brief, court_filing, motion, court_order, judgment, ruling, investigative_report, administrative_record, other>",
  "category": "<one of: investigative, judicial, administrative, legislative>",
  "confidence": <0.0 to 1.0>,
  "reasoning": "<brief explanation>"
}`, preview)

	// Create chat request
	chatReq := ChatRequest{
		Model: "mistral-large-latest",
		Messages: []ChatMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	// Marshal request
	reqBody, err := json.Marshal(chatReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		c.baseURL+"/chat/completions",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("mistral chat request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("Mistral chat API error",
			zap.Int("status_code", resp.StatusCode),
			zap.String("response", string(body)))
		return nil, fmt.Errorf("mistral chat error %d: %s", resp.StatusCode, string(body))
	}

	// Parse chat response
	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode chat response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from mistral")
	}

	// Extract JSON from response
	content := chatResp.Choices[0].Message.Content

	// Remove markdown code blocks if present
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	c.logger.Debug("Classification response",
		zap.Int("content_length", len(content)),
		zap.String("content_preview", content[:min(50, len(content))]))

	// Parse classification result
	var classification DocumentClassification
	if err := json.Unmarshal([]byte(content), &classification); err != nil {
		c.logger.Error("Failed to parse classification JSON",
			zap.String("content", content),
			zap.Error(err))
		return nil, fmt.Errorf("failed to parse classification: %w", err)
	}

	c.logger.Info("Document classified",
		zap.String("type", classification.DocumentType),
		zap.String("category", classification.Category),
		zap.Float64("confidence", classification.Confidence))

	return &classification, nil
}

// ExtractedEntities represents structured data extracted from document
type ExtractedEntities struct {
	People      []string               `json:"people"`
	Dates       map[string]string      `json:"dates"`       // e.g., {"arrest_date": "2024-01-15", "trial_date": "2024-06-20"}
	Locations   []string               `json:"locations"`
	CaseNumbers []string               `json:"case_numbers"`
	Charges     []map[string]string    `json:"charges"`     // e.g., [{"charge": "Assault", "statute": "PC 22.01"}]
	Court       string                 `json:"court"`
	Judge       string                 `json:"judge"`
	Parties     map[string]interface{} `json:"parties"`     // {plaintiff: "", defendant: "", attorney: ""}
}

// ExtractEntities uses Mistral to extract structured data from OCR text
func (c *Client) ExtractEntities(ctx context.Context, ocrText string) (*ExtractedEntities, error) {
	// Take first 4000 characters for entity extraction
	preview := ocrText
	if len(preview) > 4000 {
		preview = preview[:4000]
	}

	c.logger.Info("Extracting entities with Mistral",
		zap.Int("text_length", len(preview)))

	// Build extraction prompt
	prompt := fmt.Sprintf(`Extract structured information from this legal/investigative document.

Document text:
%s

Respond with ONLY a JSON object (no markdown, no explanation) with this structure:
{
  "people": ["names of people mentioned"],
  "dates": {"arrest_date": "YYYY-MM-DD", "trial_date": "YYYY-MM-DD", ...},
  "locations": ["cities, addresses, jurisdictions"],
  "case_numbers": ["case/docket numbers"],
  "charges": [{"charge": "name", "statute": "code"}],
  "court": "court name if mentioned",
  "judge": "judge name if mentioned",
  "parties": {"plaintiff": "name", "defendant": "name", "attorney": "name"}
}

If a field is not found, use empty array/object/string. Extract only what's clearly stated in the text.`, preview)

	// Create chat request
	chatReq := ChatRequest{
		Model: "mistral-large-latest",
		Messages: []ChatMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	// Marshal and send request
	reqBody, err := json.Marshal(chatReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		c.baseURL+"/chat/completions",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	// Execute
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("mistral chat request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mistral chat error %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from mistral")
	}

	// Extract JSON
	content := chatResp.Choices[0].Message.Content
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	// Parse entities
	var entities ExtractedEntities
	if err := json.Unmarshal([]byte(content), &entities); err != nil {
		c.logger.Error("Failed to parse entities JSON",
			zap.String("content", content),
			zap.Error(err))
		return nil, fmt.Errorf("failed to parse entities: %w", err)
	}

	c.logger.Info("Entities extracted",
		zap.Int("people_count", len(entities.People)),
		zap.Int("case_numbers_count", len(entities.CaseNumbers)),
		zap.Int("locations_count", len(entities.Locations)))

	return &entities, nil
}
