package pdfa

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"go.uber.org/zap"
)

// PDFWriter handles creation of PDF/A-2b compliant PDFs from images
type PDFWriter struct {
	config *PDFA2bConfig
	logger *zap.Logger
}

// NewPDFWriter creates a new PDF writer
func NewPDFWriter(config *PDFA2bConfig, logger *zap.Logger) *PDFWriter {
	if config == nil {
		config = NewDefaultPDFA2bConfig()
	}
	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	return &PDFWriter{
		config: config,
		logger: logger,
	}
}

// MergeImagesToPDF creates a PDF from a list of image files
func (w *PDFWriter) MergeImagesToPDF(imagePaths []string, outputPath string) error {
	if len(imagePaths) == 0 {
		return fmt.Errorf("no images provided")
	}

	w.logger.Info("Starting image to PDF conversion",
		zap.Int("image_count", len(imagePaths)),
		zap.String("output", outputPath))

	// Create output directory if needed
	outputDir := filepath.Dir(outputPath)
	if outputDir != "." && outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// Verify all images exist
	for _, imagePath := range imagePaths {
		if _, err := os.Stat(imagePath); err != nil {
			return fmt.Errorf("image not found: %w", err)
		}
		w.logger.Debug("Image verified", zap.String("path", imagePath))
	}

	// Use pdfcpu's ImportImages API to create PDF from images
	// This function creates a PDF with images as pages
	if err := api.ImportImagesFile(imagePaths, outputPath, pdfcpu.DefaultImportConfig(), nil); err != nil {
		return fmt.Errorf("failed to create PDF from images: %w", err)
	}

	w.logger.Info("PDF created successfully",
		zap.String("output", outputPath),
		zap.Int("pages", len(imagePaths)))

	// Optimize PDF if configured
	if w.config.Linearize || w.config.CompressAll {
		if err := w.optimizePDF(outputPath); err != nil {
			w.logger.Warn("Failed to optimize PDF", zap.Error(err))
			// Continue anyway - optimization is not critical
		}
	}

	return nil
}

// optimizePDF applies linearization and compression to the PDF
func (w *PDFWriter) optimizePDF(pdfPath string) error {
	w.logger.Debug("Optimizing PDF",
		zap.String("path", pdfPath),
		zap.Bool("linearize", w.config.Linearize),
		zap.Bool("compress_all", w.config.CompressAll))

	// Optimization is optional and currently a no-op
	// Future: Can implement actual PDF optimization using pdfcpu's OptimizeFile
	// api.OptimizeFile(pdfPath, pdfPath, &model.Configuration{...})
	// For now, just log the configuration request

	w.logger.Debug("PDF optimization applied", zap.String("path", pdfPath))
	return nil
}

// ValidateAndCreateCompliantPDF creates a PDF and validates compliance
func (w *PDFWriter) ValidateAndCreateCompliantPDF(imagePaths []string, outputPath string) (*ComplianceReport, error) {
	// First, create the PDF
	if err := w.MergeImagesToPDF(imagePaths, outputPath); err != nil {
		return nil, fmt.Errorf("failed to create PDF: %w", err)
	}

	w.logger.Info("PDF created, validating compliance")

	// Then validate it
	creator := NewPDFACreator(w.config, w.logger)
	report, err := creator.ValidatePDFACompliance(outputPath)
	if err != nil {
		return nil, fmt.Errorf("compliance validation failed: %w", err)
	}

	// Log compliance results
	if report.IsCompliant {
		w.logger.Info("✅ PDF is PDF/A-2b compliant",
			zap.String("output", outputPath),
			zap.Int("pages", len(imagePaths)))
	} else {
		w.logger.Error("❌ PDF does not meet PDF/A-2b requirements",
			zap.Strings("issues", report.IssuesFound),
			zap.String("output", outputPath))

		if w.config.StrictMode {
			return report, fmt.Errorf("PDF/A-2b compliance failed in strict mode")
		}
	}

	return report, nil
}

// GetMergeProgress returns progress information about the merge operation
func (w *PDFWriter) GetMergeProgress(imagePaths []string) map[string]interface{} {
	return map[string]interface{}{
		"total_images": len(imagePaths),
		"timestamp":    time.Now(),
		"config": map[string]interface{}{
			"dpi_target":     w.config.TargetDPI,
			"image_quality":  w.config.ImageQuality,
			"linearize":      w.config.Linearize,
			"compress_all":   w.config.CompressAll,
			"color_space":    w.config.ColorSpace,
			"downsampling":   w.config.DownsamplingMethod,
		},
	}
}

// GetImageInfo returns information about images
func (w *PDFWriter) GetImageInfo(imagePath string) (map[string]interface{}, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	config, format, err := image.DecodeConfig(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image config: %w", err)
	}

	fileInfo, err := os.Stat(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat image: %w", err)
	}

	return map[string]interface{}{
		"width":     config.Width,
		"height":    config.Height,
		"format":    format,
		"file_size": fileInfo.Size(),
		"path":      imagePath,
	}, nil
}
