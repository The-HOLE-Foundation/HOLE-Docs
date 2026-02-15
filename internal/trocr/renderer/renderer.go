package renderer

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/The-HOLE-Foundation/hole-docs/internal/pdf"
	"github.com/The-HOLE-Foundation/hole-docs/internal/storage"
	"go.uber.org/zap"
)

// Renderer handles PDF to image conversion for TrOCR processing
type Renderer struct {
	pdfProcessor *pdf.Processor
	logger       *zap.Logger
}

// Config holds rendering configuration
type Config struct {
	Format string // "png" or "jpeg"
	DPI    int    // Resolution (150-600, recommended: 300 for TrOCR)
}

// DefaultConfig returns recommended settings for TrOCR
func DefaultConfig() Config {
	return Config{
		Format: "png", // Lossless format preferred
		DPI:    200,   // Good balance for TrOCR (faster than 300, accurate enough)
	}
}

// NewRenderer creates a new PDF renderer
func NewRenderer(cache *storage.Cache, logger *zap.Logger) *Renderer {
	return &Renderer{
		pdfProcessor: pdf.NewProcessor(cache, logger),
		logger:       logger,
	}
}

// RenderPDFToImages converts all pages of a PDF to individual images
// Returns slice of image paths in page order
func (r *Renderer) RenderPDFToImages(pdfPath string, outputDir string, config Config) ([]string, error) {
	r.logger.Info("Rendering PDF to images for TrOCR",
		zap.String("pdf", pdfPath),
		zap.String("output_dir", outputDir),
		zap.String("format", config.Format),
		zap.Int("dpi", config.DPI))

	// Verify PDF exists
	if _, err := os.Stat(pdfPath); err != nil {
		return nil, fmt.Errorf("PDF file not found: %s", pdfPath)
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Render using existing PDF processor
	err := r.pdfProcessor.RenderPagesToImages(pdfPath, outputDir, config.Format, config.DPI)
	if err != nil {
		return nil, fmt.Errorf("failed to render PDF pages: %w", err)
	}

	// Collect all generated image paths
	imagePaths, err := r.collectImagePaths(outputDir, config.Format)
	if err != nil {
		return nil, fmt.Errorf("failed to collect image paths: %w", err)
	}

	r.logger.Info("PDF rendered successfully",
		zap.String("pdf", pdfPath),
		zap.Int("page_count", len(imagePaths)))

	return imagePaths, nil
}

// collectImagePaths finds all rendered images in output directory
// Returns paths in page order
func (r *Renderer) collectImagePaths(outputDir string, format string) ([]string, error) {
	// Pattern: filename-page-001.png, filename-page-002.png, etc.
	pattern := filepath.Join(outputDir, fmt.Sprintf("*-page-*.%s", format))

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob images: %w", err)
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no images found in %s (pattern: %s)", outputDir, pattern)
	}

	// Note: Ghostscript numbers pages sequentially, so alphabetical sort works
	// for up to 999 pages (page-001, page-002, ..., page-999)
	return matches, nil
}

// RenderPDFPair renders both original and marked PDFs for comparison
// Returns (originalImages, markedImages, error)
func (r *Renderer) RenderPDFPair(originalPDF, markedPDF string, workDir string, config Config) ([]string, []string, error) {
	r.logger.Info("Rendering PDF pair for visual diff",
		zap.String("original", originalPDF),
		zap.String("marked", markedPDF))

	// Create subdirectories for each PDF
	originalDir := filepath.Join(workDir, "original_pages")
	markedDir := filepath.Join(workDir, "marked_pages")

	// Render original PDF
	r.logger.Info("Rendering original PDF")
	originalImages, err := r.RenderPDFToImages(originalPDF, originalDir, config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to render original PDF: %w", err)
	}

	// Render marked PDF
	r.logger.Info("Rendering marked PDF")
	markedImages, err := r.RenderPDFToImages(markedPDF, markedDir, config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to render marked PDF: %w", err)
	}

	// Verify page counts match
	if len(originalImages) != len(markedImages) {
		return nil, nil, fmt.Errorf("page count mismatch: original=%d, marked=%d",
			len(originalImages), len(markedImages))
	}

	r.logger.Info("PDF pair rendered successfully",
		zap.Int("page_count", len(originalImages)))

	return originalImages, markedImages, nil
}
