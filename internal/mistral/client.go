package mistral

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Client handles communication with Mistral Document API
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// OCRRequest represents the request to Mistral OCR API
// Based on Python SDK example from official docs
type OCRRequest struct {
	Model    string `json:"model"`
	Document struct {
		Type        string `json:"type"`
		DocumentURL string `json:"document_url"`
	} `json:"document"`
	IncludeImageBase64 bool `json:"include_image_base64"`
}

// OCRPage represents a single page from Mistral OCR
type OCRPage struct {
	Index    int    `json:"index"`
	Markdown string `json:"markdown"`
}

// OCRResponse represents the actual response from Mistral OCR API
type OCRResponse struct {
	Pages        []OCRPage              `json:"pages"`
	TextContent  string                 `json:"text_content"`  // Computed field
	ImagesBBoxes []ImageBBox            `json:"images_bboxes"`
	Header       string                 `json:"header"`
	Footer       string                 `json:"footer"`
	Structure    map[string]interface{} `json:"structure"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// ImageBBox represents an image bounding box
type ImageBBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// NewClient creates a Client configured with the given API key, the default Mistral base URL (https://api.mistral.ai/v1), an HTTP client with a 120s timeout, and the provided logger.
func NewClient(apiKey string, logger *zap.Logger) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: "https://api.mistral.ai/v1",
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		logger: logger,
	}
}

// ProcessPDF processes a PDF file with Mistral OCR
// Uses Python SDK pattern: client.ocr.process()
func (c *Client) ProcessPDF(ctx context.Context, pdfPath string) (*OCRResponse, error) {
	c.logger.Info("Processing PDF with Mistral OCR",
		zap.String("pdf_path", pdfPath))

	// Read PDF file
	pdfData, err := os.ReadFile(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF: %w", err)
	}

	fileSize := len(pdfData)
	c.logger.Info("PDF file read",
		zap.Int("size_bytes", fileSize),
		zap.Float64("size_mb", float64(fileSize)/(1024*1024)))

	// Encode to base64
	base64PDF := base64.StdEncoding.EncodeToString(pdfData)

	// Build request matching Python SDK pattern
	req := OCRRequest{
		Model: "mistral-ocr-latest",
	}
	req.Document.Type = "document_url"
	req.Document.DocumentURL = fmt.Sprintf("data:application/pdf;base64,%s", base64PDF)
	req.IncludeImageBase64 = false

	// Marshal request
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	c.logger.Info("Sending request to Mistral API",
		zap.Int("request_size_bytes", len(reqBody)))

	// Try the ocr.process endpoint
	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		c.baseURL+"/ocr",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	// Execute request
	startTime := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("mistral api request failed: %w", err)
	}
	defer resp.Body.Close()

	duration := time.Since(startTime)
	c.logger.Info("Mistral API response received",
		zap.Int("status_code", resp.StatusCode),
		zap.Duration("duration", duration))

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Truncate body for logging to avoid PII leakage
		bodyPreview := string(body)
		if len(bodyPreview) > 200 {
			bodyPreview = bodyPreview[:200] + "... (truncated)"
		}
		c.logger.Error("Mistral API error",
			zap.Int("status_code", resp.StatusCode),
			zap.String("response_preview", bodyPreview))
		return nil, fmt.Errorf("mistral api error %d", resp.StatusCode)
	}

	// Log raw response for debugging (truncated to avoid PII)
	bodyPreview := string(body)
	if len(bodyPreview) > 200 {
		bodyPreview = bodyPreview[:200] + "... (truncated)"
	}
	c.logger.Debug("Raw Mistral API response",
		zap.Int("body_length", len(body)),
		zap.String("body_preview", bodyPreview))

	// Parse response
	var ocrResp OCRResponse
	if err := json.Unmarshal(body, &ocrResp); err != nil {
		// Use the already-truncated bodyPreview to avoid PII leakage
		c.logger.Error("Failed to parse response",
			zap.String("response_preview", bodyPreview))
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Combine all pages' markdown into TextContent
	var fullText strings.Builder
	for i, page := range ocrResp.Pages {
		if i > 0 {
			fullText.WriteString("\n\n---\n\n") // Page separator
		}
		fullText.WriteString(page.Markdown)
	}
	ocrResp.TextContent = fullText.String()

	c.logger.Info("OCR processing complete",
		zap.Int("pages_count", len(ocrResp.Pages)),
		zap.Int("text_length", len(ocrResp.TextContent)),
		zap.String("header", ocrResp.Header),
		zap.String("footer", ocrResp.Footer),
		zap.Int("images_count", len(ocrResp.ImagesBBoxes)))

	return &ocrResp, nil
}

// min returns the smaller of a and b.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetPageCount returns the number of pages in the document
func (r *OCRResponse) GetPageCount() int {
	if len(r.Pages) > 0 {
		return len(r.Pages)
	}

	// Fallback to structure metadata
	if r.Structure == nil {
		return 0
	}

	if pages, ok := r.Structure["pages"].(float64); ok {
		return int(pages)
	}

	if pageCount, ok := r.Structure["page_count"].(float64); ok {
		return int(pageCount)
	}

	return 0
}

// GetTableCount extracts table count from OCR structure
func (r *OCRResponse) GetTableCount() int {
	if r.Structure == nil {
		return 0
	}

	if tables, ok := r.Structure["tables_count"].(float64); ok {
		return int(tables)
	}

	return 0
}

// HasImages checks if the document contains images
func (r *OCRResponse) HasImages() bool {
	return len(r.ImagesBBoxes) > 0
}