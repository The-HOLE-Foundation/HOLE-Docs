package trocr

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"

	"go.uber.org/zap"
)

// Client handles communication with TrOCR Python service
type Client struct {
	pythonScript string // Path to trocr_service.py
	model        string // "handwritten", "printed", or "large"
	logger       *zap.Logger
}

// TrOCRResult represents TrOCR output for a single page
type TrOCRResult struct {
	Success bool   `json:"success"`
	Text    string `json:"text"`
	Image   string `json:"image"`
	Error   string `json:"error,omitempty"`
}

// PageText maps page numbers to extracted text
type PageText map[int]string

// NewClient creates a new TrOCR client
func NewClient(pythonScript string, model string, logger *zap.Logger) *Client {
	return &Client{
		pythonScript: pythonScript,
		model:        model,
		logger:       logger,
	}
}

// ProcessBatch runs TrOCR on all images in a directory
// Returns map of page number -> extracted text
func (c *Client) ProcessBatch(imageDir string) (PageText, error) {
	c.logger.Info("Running TrOCR batch processing",
		zap.String("image_dir", imageDir),
		zap.String("model", c.model))

	// Build command: python3 trocr_service.py <dir> --model <model>
	cmd := exec.Command("python3", c.pythonScript, imageDir, "--model", c.model)

	// Capture stdout (JSON output) and stderr (progress logs)
	output, err := cmd.Output()
	if err != nil {
		// If command failed, try to get stderr
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("TrOCR command failed: %w\nStderr: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("TrOCR command failed: %w", err)
	}

	// Parse JSON output
	var results []TrOCRResult
	if err := json.Unmarshal(output, &results); err != nil {
		return nil, fmt.Errorf("failed to parse TrOCR output: %w\nOutput: %s", err, string(output))
	}

	// Convert to PageText map
	pageText := make(PageText)
	for _, result := range results {
		if !result.Success {
			c.logger.Warn("TrOCR failed for page",
				zap.String("image", result.Image),
				zap.String("error", result.Error))
			continue
		}

		// Extract page number from image filename
		pageNum, err := extractPageNumber(result.Image)
		if err != nil {
			c.logger.Warn("Failed to parse page number",
				zap.String("image", result.Image),
				zap.Error(err))
			continue
		}

		pageText[pageNum] = result.Text
	}

	c.logger.Info("TrOCR batch complete",
		zap.Int("total_pages", len(results)),
		zap.Int("successful_pages", len(pageText)))

	return pageText, nil
}

// extractPageNumber parses page number from image filename
// Patterns: "filename-page-001.png" or "page_001.png"
func extractPageNumber(filename string) (int, error) {
	// Try pattern 1: "filename-page-NNN.ext"
	re1 := regexp.MustCompile(`-page-(\d+)\.`)
	if matches := re1.FindStringSubmatch(filename); len(matches) > 1 {
		return strconv.Atoi(matches[1])
	}

	// Try pattern 2: "page_NNN.ext"
	re2 := regexp.MustCompile(`page_(\d+)\.`)
	if matches := re2.FindStringSubmatch(filename); len(matches) > 1 {
		return strconv.Atoi(matches[1])
	}

	return 0, fmt.Errorf("could not extract page number from filename: %s", filename)
}

// ProcessPDFImages is a convenience method for rendering + TrOCR in one step
// Returns PageText map
func (c *Client) ProcessPDFImages(imageDir string) (PageText, error) {
	// Verify directory exists
	if _, err := filepath.Abs(imageDir); err != nil {
		return nil, fmt.Errorf("invalid image directory: %w", err)
	}

	return c.ProcessBatch(imageDir)
}
