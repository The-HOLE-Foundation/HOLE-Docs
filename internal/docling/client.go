package docling

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Client handles communication with Docling service
type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewClient creates a Client configured to communicate with a Docling service.
// If baseURL is empty the client defaults to "http://localhost:5000". The
// returned client uses an http.Client with a 5-minute timeout to accommodate
// processing of large documents and preserves the provided logger.
func NewClient(baseURL string, logger *zap.Logger) *Client {
	if baseURL == "" {
		baseURL = "http://localhost:5000" // Default local development
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute, // Docling can take time for large documents
		},
		logger: logger,
	}
}

// IsConfigured checks if Docling service is available
func (c *Client) IsConfigured() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.HealthCheck(ctx)
	return err == nil
}

// HealthCheck verifies Docling service is running
func (c *Client) HealthCheck(ctx context.Context) (*HealthCheckResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Docling health check failed", zap.Error(err))
		return nil, fmt.Errorf("docling service unavailable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("docling health check returned %d", resp.StatusCode)
	}

	var healthResp HealthCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		return nil, fmt.Errorf("failed to parse health check response: %w", err)
	}

	return &healthResp, nil
}

// ProcessPDF processes a PDF file with Docling
// Returns structured document with text, metadata, tables, and hierarchy
func (c *Client) ProcessPDF(ctx context.Context, filePath string) (*ProcessingResult, error) {
	return c.process(ctx, filePath, "pdf")
}

// ProcessDocument processes any document format that Docling supports
// Supports: PDF, DOCX, PPTX, images, etc.
func (c *Client) ProcessDocument(ctx context.Context, filePath string) (*ProcessingResult, error) {
	return c.process(ctx, filePath, "auto")
}

// process handles the actual document processing request
func (c *Client) process(ctx context.Context, filePath string, format string) (*ProcessingResult, error) {
	c.logger.Info("Processing document with Docling",
		zap.String("file_path", filePath),
		zap.String("format", format))

	// Prepare request
	reqBody := ProcessRequest{
		FilePath: filePath,
		FileName: filePath, // Can be extracted separately if needed
		Options: map[string]interface{}{
			"format":        format,
			"extract_text":  true,
			"extract_tables": true,
			"extract_structure": true,
			"extract_metadata": true,
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/process", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create process request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Execute request
	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to call Docling service", zap.Error(err))
		return nil, fmt.Errorf("docling service call failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle error responses
	if resp.StatusCode != http.StatusOK {
		c.logger.Error("Docling returned error",
			zap.Int("status_code", resp.StatusCode),
			zap.String("response", string(body)))
		return nil, fmt.Errorf("docling error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var processResp ProcessResponse
	if err := json.Unmarshal(body, &processResp); err != nil {
		return nil, fmt.Errorf("failed to parse docling response: %w", err)
	}

	if !processResp.Success {
		return nil, fmt.Errorf("docling processing failed: %s", processResp.Error)
	}

	// Add processing timing
	processingResult := processResp.Data
	processingResult.ProcessedAt = time.Now()
	processingResult.ProcessingTime = time.Since(startTime).Milliseconds()
	processingResult.RawJSON = json.RawMessage(body)

	c.logger.Info("Document processed successfully with Docling",
		zap.String("file_path", filePath),
		zap.Int("pages", processingResult.PageCount),
		zap.String("language", processingResult.Language),
		zap.Int64("processing_time_ms", processingResult.ProcessingTime))

	return &processingResult, nil
}

// BatchProcess processes multiple documents in sequence
// Returns results in order, continues on individual failures
func (c *Client) BatchProcess(ctx context.Context, filePaths []string) ([]*ProcessingResult, []error) {
	results := make([]*ProcessingResult, len(filePaths))
	errors := make([]error, len(filePaths))

	for i, filePath := range filePaths {
		result, err := c.ProcessDocument(ctx, filePath)
		results[i] = result
		errors[i] = err

		if err != nil {
			c.logger.Warn("Failed to process document in batch",
				zap.String("file_path", filePath),
				zap.Error(err))
		}
	}

	return results, errors
}

// GetTextContent extracts plain text from processing result
// Useful for search indexing
func (c *Client) GetTextContent(result *ProcessingResult) string {
	if result == nil {
		return ""
	}
	return result.RawText
}

// GetMarkdown extracts markdown representation
// Useful for readable output and further processing
func (c *Client) GetMarkdown(result *ProcessingResult) string {
	if result == nil {
		return ""
	}
	return result.Markdown
}

// GetStructuredData returns document structure and hierarchy
// Useful for legal document analysis where structure matters
func (c *Client) GetStructuredData(result *ProcessingResult) DocumentStructure {
	if result == nil {
		return DocumentStructure{}
	}
	return result.Structure
}

// GetTables returns extracted tables in structured format
// Each table is separately parsed with cells and rows
func (c *Client) GetTables(result *ProcessingResult) []Table {
	if result == nil {
		return []Table{}
	}
	return result.Tables
}

// GetMetadata returns extracted document metadata
// Includes title, author, dates, language, etc.
func (c *Client) GetMetadata(result *ProcessingResult) DocumentMetadata {
	if result == nil {
		return DocumentMetadata{}
	}
	return result.Metadata
}

// ExportAsJSON returns complete result as JSON for storage
// Provides complete lossless representation for legal document archival
func (c *Client) ExportAsJSON(result *ProcessingResult) (json.RawMessage, error) {
	if result == nil {
		return nil, fmt.Errorf("result is nil")
	}

	// Use RawJSON if available (preserves original API response exactly)
	if result.RawJSON != nil {
		return result.RawJSON, nil
	}

	// Otherwise marshal the result
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return data, nil
}

// ExportAsMarkdown returns markdown representation
func (c *Client) ExportAsMarkdown(result *ProcessingResult) string {
	if result == nil {
		return ""
	}
	return result.Markdown
}

// ExtractKeywords extracts keywords from the document using simple heuristics
// ExtractKeywords extracts keywords from a ProcessingResult.
// It returns a slice composed of the document Title and Subject (if present)
// and the content of first-level headings found in the document structure.
// If result is nil or RawText is empty, it returns an empty slice.
// This is a heuristic extraction and may be replaced with NLP-based analysis later.
func ExtractKeywords(result *ProcessingResult) []string {
	if result == nil || result.RawText == "" {
		return []string{}
	}

	// This is a simplified version - in production, you'd use NLP
	// For now, extract from metadata and first-level headings
	keywords := []string{}

	if result.Metadata.Title != "" {
		keywords = append(keywords, result.Metadata.Title)
	}

	if result.Metadata.Subject != "" {
		keywords = append(keywords, result.Metadata.Subject)
	}

	// Extract first-level headings as keywords
	for _, elem := range result.Structure.Elements {
		if elem.Type == "heading" && elem.Level == 1 && elem.Content != "" {
			keywords = append(keywords, elem.Content)
		}
	}

	return keywords
}