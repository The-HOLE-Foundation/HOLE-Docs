package trocr

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/The-HOLE-Foundation/hole-docs/internal/storage"
	"github.com/The-HOLE-Foundation/hole-docs/internal/trocr/differ"
	"github.com/The-HOLE-Foundation/hole-docs/internal/trocr/renderer"
	"go.uber.org/zap"
)

// Processor orchestrates the complete TrOCR visual diff workflow
type Processor struct {
	renderer     *renderer.Renderer
	trocrClient  *Client
	differ       *differ.Differ
	workDir      string
	pythonScript string
	logger       *zap.Logger
}

// ProcessorConfig holds configuration for the visual diff processor
type ProcessorConfig struct {
	WorkDir      string // Working directory for temp files
	PythonScript string // Path to trocr_service.py
	Model        string // TrOCR model: "handwritten", "printed", or "large"
	RenderDPI    int    // DPI for PDF rendering (default: 200)
}

// Result holds the complete visual diff result
type Result struct {
	OriginalPDF  string          `json:"original_pdf"`
	MarkedPDF    string          `json:"marked_pdf"`
	PageCount    int             `json:"page_count"`
	Changes      []differ.Change `json:"changes"`
	ChangedPages int             `json:"changed_pages"`
	Summary      string          `json:"summary"`
	Success      bool            `json:"success"`
	Error        string          `json:"error,omitempty"`
}

// NewProcessor creates a new visual diff processor
func NewProcessor(cache *storage.Cache, config ProcessorConfig, logger *zap.Logger) (*Processor, error) {
	// Set defaults
	if config.WorkDir == "" {
		config.WorkDir = os.TempDir()
	}
	if config.Model == "" {
		config.Model = "printed"
	}
	if config.RenderDPI <= 0 {
		config.RenderDPI = 200
	}

	// Verify Python script exists
	if config.PythonScript == "" {
		return nil, fmt.Errorf("python script path is required")
	}
	if _, err := os.Stat(config.PythonScript); err != nil {
		return nil, fmt.Errorf("python script not found: %s", config.PythonScript)
	}

	// Create working directory
	if err := os.MkdirAll(config.WorkDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create working directory: %w", err)
	}

	return &Processor{
		renderer:     renderer.NewRenderer(cache, logger),
		trocrClient:  NewClient(config.PythonScript, config.Model, logger),
		differ:       differ.NewDiffer(logger),
		workDir:      config.WorkDir,
		pythonScript: config.PythonScript,
		logger:       logger,
	}, nil
}

// ProcessVisualDiff runs the complete visual diff workflow
func (p *Processor) ProcessVisualDiff(originalPDF, markedPDF string) (*Result, error) {
	p.logger.Info("Starting visual diff processing",
		zap.String("original", originalPDF),
		zap.String("marked", markedPDF))

	result := &Result{
		OriginalPDF: originalPDF,
		MarkedPDF:   markedPDF,
	}

	// Step 1: Render both PDFs to images
	p.logger.Info("Step 1: Rendering PDFs to images")
	config := renderer.DefaultConfig()
	originalImages, _, err := p.renderer.RenderPDFPair(
		originalPDF,
		markedPDF,
		p.workDir,
		config,
	)
	if err != nil {
		result.Error = fmt.Sprintf("Rendering failed: %v", err)
		return result, err
	}

	result.PageCount = len(originalImages)
	p.logger.Info("PDFs rendered successfully", zap.Int("pages", result.PageCount))

	// Step 2: Run TrOCR on original images
	p.logger.Info("Step 2: Running TrOCR on original PDF")
	originalDir := filepath.Join(p.workDir, "original_pages")
	originalTexts, err := p.trocrClient.ProcessBatch(originalDir)
	if err != nil {
		result.Error = fmt.Sprintf("TrOCR failed on original: %v", err)
		return result, err
	}

	// Step 3: Run TrOCR on marked images
	p.logger.Info("Step 3: Running TrOCR on marked PDF")
	markedDir := filepath.Join(p.workDir, "marked_pages")
	markedTexts, err := p.trocrClient.ProcessBatch(markedDir)
	if err != nil {
		result.Error = fmt.Sprintf("TrOCR failed on marked: %v", err)
		return result, err
	}

	// Step 4: Compute diffs
	p.logger.Info("Step 4: Computing text differences")
	changes, err := p.differ.ComputeChanges(originalTexts, markedTexts)
	if err != nil {
		result.Error = fmt.Sprintf("Diff computation failed: %v", err)
		return result, err
	}

	result.Changes = changes
	result.ChangedPages = len(changes)
	result.Summary = p.differ.GenerateSummary(changes)
	result.Success = true

	p.logger.Info("Visual diff processing complete",
		zap.Int("total_pages", result.PageCount),
		zap.Int("changed_pages", result.ChangedPages))

	return result, nil
}

// ExportResult saves the diff result to a JSON file
func (p *Processor) ExportResult(result *Result, outputPath string) error {
	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write result file: %w", err)
	}

	p.logger.Info("Result exported",
		zap.String("output", outputPath),
		zap.Int64("size", int64(len(data))))

	return nil
}

// Cleanup removes temporary files generated during processing
func (p *Processor) Cleanup() error {
	// Remove original_pages and marked_pages directories
	originalDir := filepath.Join(p.workDir, "original_pages")
	markedDir := filepath.Join(p.workDir, "marked_pages")

	var errs []error

	if err := os.RemoveAll(originalDir); err != nil {
		errs = append(errs, fmt.Errorf("failed to remove original_pages: %w", err))
	}

	if err := os.RemoveAll(markedDir); err != nil {
		errs = append(errs, fmt.Errorf("failed to remove marked_pages: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("cleanup failed with %d errors: %v", len(errs), errs)
	}

	p.logger.Info("Cleanup complete", zap.String("work_dir", p.workDir))
	return nil
}
