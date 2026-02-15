package pdf

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/The-HOLE-Foundation/hole-docs/internal/storage"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"go.uber.org/zap"
)

// OptimizationPreset defines DPI and quality settings for PDF optimization
type OptimizationPreset struct {
	Name               string // Name of the preset
	DPI                int    // Resolution in dots per inch
	JpegQuality        int    // JPEG quality 1-100
	CompressText       bool   // Use lossless compression for text
	CompressFonts      bool   // Subset and compress fonts
	DetectDuplicates   bool   // Detect and remove duplicate images
	PreserveColors     bool   // Preserve color information
	UseCase            string // Description of when to use this preset
}

// PredefinedPresets contains all available optimization presets
var PredefinedPresets = map[string]OptimizationPreset{
	"72dpi": {
		Name:             "72 DPI",
		DPI:              72,
		JpegQuality:      60,
		CompressText:     true,
		CompressFonts:    true,
		DetectDuplicates: true,
		PreserveColors:   true,
		UseCase:          "Email/Web - Maximum compression, minimal quality",
	},
	"100dpi": {
		Name:             "100 DPI",
		DPI:              100,
		JpegQuality:      65,
		CompressText:     true,
		CompressFonts:    true,
		DetectDuplicates: true,
		PreserveColors:   true,
		UseCase:          "Screen viewing - Good balance for displays",
	},
	"125dpi": {
		Name:             "125 DPI",
		DPI:              125,
		JpegQuality:      70,
		CompressText:     true,
		CompressFonts:    true,
		DetectDuplicates: true,
		PreserveColors:   true,
		UseCase:          "E-books/tablets - Portable viewing",
	},
	"150dpi": {
		Name:             "150 DPI",
		DPI:              150,
		JpegQuality:      75,
		CompressText:     true,
		CompressFonts:    true,
		DetectDuplicates: true,
		PreserveColors:   true,
		UseCase:          "Small file size - Still readable for most documents",
	},
	"172dpi": {
		Name:             "172 DPI (DEFAULT)",
		DPI:              172,
		JpegQuality:      78,
		CompressText:     true,
		CompressFonts:    true,
		DetectDuplicates: true,
		PreserveColors:   true,
		UseCase:          "STANDARD - Legal documents (optimal balance)",
	},
	"200dpi": {
		Name:             "200 DPI",
		DPI:              200,
		JpegQuality:      80,
		CompressText:     true,
		CompressFonts:    true,
		DetectDuplicates: true,
		PreserveColors:   true,
		UseCase:          "Legal - High searchability with compression",
	},
	"220dpi": {
		Name:             "220 DPI",
		DPI:              220,
		JpegQuality:      82,
		CompressText:     true,
		CompressFonts:    true,
		DetectDuplicates: true,
		PreserveColors:   true,
		UseCase:          "Legal - Enhanced readability and OCR",
	},
	"300dpi": {
		Name:             "300 DPI",
		DPI:              300,
		JpegQuality:      85,
		CompressText:     true,
		CompressFonts:    true,
		DetectDuplicates: true,
		PreserveColors:   true,
		UseCase:          "Archival/Print - High quality, still compressed",
	},
}

// Processor handles PDF operations
type Processor struct {
	cache  *storage.Cache
	logger *zap.Logger
}

// NewProcessor creates a new PDF processor
func NewProcessor(cache *storage.Cache, logger *zap.Logger) *Processor {
	return &Processor{
		cache:  cache,
		logger: logger,
	}
}

// MergePDFs merges multiple PDF files into one
func (p *Processor) MergePDFs(inputPaths []string, outputPath string) error {
	p.logger.Info("Merging PDFs", zap.Int("count", len(inputPaths)))

	// Validate inputs
	if len(inputPaths) < 2 {
		return fmt.Errorf("at least 2 PDF files required for merging, got %d", len(inputPaths))
	}

	// Verify all input files exist
	for _, path := range inputPaths {
		if _, err := os.Stat(path); err != nil {
			p.logger.Error("Input file not found", zap.String("path", path), zap.Error(err))
			return fmt.Errorf("input file not found: %s", path)
		}
	}

	// Merge using pdfcpu (MergeCreateFile with dividerPage=false for direct merge)
	if err := api.MergeCreateFile(inputPaths, outputPath, false, nil); err != nil {
		p.logger.Error("Failed to merge PDFs", zap.Error(err))
		return fmt.Errorf("failed to merge PDFs: %w", err)
	}

	// Log success with file size
	fileInfo, _ := os.Stat(outputPath)
	p.logger.Info("PDFs merged successfully",
		zap.String("output", outputPath),
		zap.Int64("size", fileInfo.Size()))

	return nil
}

// SplitPDF splits a PDF into individual pages or page ranges
// pageRanges format: []string{"1-5", "10-15"} or []string{"1,3,5"} for individual pages
func (p *Processor) SplitPDF(inputPath string, outputDir string, pageRanges []string) error {
	p.logger.Info("Splitting PDF",
		zap.String("input", inputPath),
		zap.Strings("ranges", pageRanges))

	// Verify input file exists
	if _, err := os.Stat(inputPath); err != nil {
		p.logger.Error("Input file not found", zap.String("path", inputPath), zap.Error(err))
		return fmt.Errorf("input file not found: %s", inputPath)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		p.logger.Error("Failed to create output directory",
			zap.String("dir", outputDir),
			zap.Error(err))
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Parse page ranges to integers for pdfcpu
	pageNrs := []int{}
	for _, pageRange := range pageRanges {
		// Handle ranges like "1-5" and individual pages like "1,3,5"
		parts := strings.Split(pageRange, "-")
		if len(parts) == 2 {
			// This is a range
			start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err != nil {
				return fmt.Errorf("invalid page range: %s", pageRange)
			}
			pageNrs = append(pageNrs, start)
		} else {
			// Parse individual page numbers
			for _, numStr := range strings.Split(pageRange, ",") {
				num, err := strconv.Atoi(strings.TrimSpace(numStr))
				if err != nil {
					return fmt.Errorf("invalid page number: %s", numStr)
				}
				pageNrs = append(pageNrs, num)
			}
		}
	}

	// Split using pdfcpu
	if err := api.SplitByPageNrFile(inputPath, outputDir, pageNrs, nil); err != nil {
		p.logger.Error("Failed to split PDF", zap.Error(err))
		return fmt.Errorf("failed to split PDF: %w", err)
	}

	p.logger.Info("PDF split successfully",
		zap.String("output_dir", outputDir),
		zap.Ints("page_splits", pageNrs))

	return nil
}

// OptimizePDF reduces file size while maintaining quality
// quality options: "low" (aggressive), "medium" (default), "high" (minimal compression)
func (p *Processor) OptimizePDF(inputPath string, outputPath string, quality string) error {
	p.logger.Info("Optimizing PDF",
		zap.String("input", inputPath),
		zap.String("quality", quality))

	// Verify input file exists
	if _, err := os.Stat(inputPath); err != nil {
		p.logger.Error("Input file not found", zap.String("path", inputPath), zap.Error(err))
		return fmt.Errorf("input file not found: %s", inputPath)
	}

	// Optimize using pdfcpu (quality parameter is for reference, pdfcpu uses its own optimization)
	if err := api.OptimizeFile(inputPath, outputPath, nil); err != nil {
		p.logger.Error("Failed to optimize PDF", zap.Error(err))
		return fmt.Errorf("failed to optimize PDF: %w", err)
	}

	// Log file size reduction
	originalInfo, _ := os.Stat(inputPath)
	optimizedInfo, _ := os.Stat(outputPath)
	reduction := 0.0
	if originalInfo.Size() > 0 {
		reduction = (1.0 - float64(optimizedInfo.Size())/float64(originalInfo.Size())) * 100
	}

	p.logger.Info("PDF optimized successfully",
		zap.String("output", outputPath),
		zap.Int64("original_size", originalInfo.Size()),
		zap.Int64("optimized_size", optimizedInfo.Size()),
		zap.Float64("reduction_percent", reduction))

	return nil
}

// OptimizePDFAdvanced reduces file size using Ghostscript with DPI control
// Preserves text layer for searchability while compressing images
// Supports preset names: "72dpi", "100dpi", "125dpi", "150dpi", "172dpi" (default), "200dpi", "220dpi", "300dpi"
// Or provide custom dpi/quality values
func (p *Processor) OptimizePDFAdvanced(inputPath string, outputPath string, dpi int, quality int) error {
	// Use 172 DPI as default if not specified or invalid
	if dpi < 72 || dpi > 600 {
		dpi = 172
	}
	if quality < 1 || quality > 100 {
		quality = 78
	}

	p.logger.Info("Optimizing PDF with Ghostscript",
		zap.String("input", inputPath),
		zap.Int("dpi", dpi),
		zap.Int("quality", quality))

	// Verify input file exists
	if _, err := os.Stat(inputPath); err != nil {
		p.logger.Error("Input file not found", zap.String("path", inputPath), zap.Error(err))
		return fmt.Errorf("input file not found: %s", inputPath)
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		p.logger.Error("Failed to create output directory",
			zap.String("dir", outputDir),
			zap.Error(err))
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Build Ghostscript command with optimal settings for searchable legal documents
	// Using proven parameters that preserve text searchability while compressing images

	cmd := exec.Command("gs",
		"-sDEVICE=pdfwrite",
		"-dCompatibilityLevel=1.4",
		"-dNOPAUSE",
		"-dBATCH",
		"-dDetectDuplicateImages",
		"-dCompressFonts=true",
		"-dSubsetFonts=true",
		fmt.Sprintf("-r%dx%d", dpi, dpi),
		"-dDownsampleMonoImages=true",
		"-dDownsampleGrayImages=true",
		"-dDownsampleColorImages=true",
		fmt.Sprintf("-dMonoImageResolution=%d", dpi),
		fmt.Sprintf("-dGrayImageResolution=%d", dpi),
		fmt.Sprintf("-dColorImageResolution=%d", dpi),
		"-dEncodeColorImages=true",
		"-dEncodeGrayImages=true",
		"-dEncodeMonoImages=true",
		"-dColorImageFilter=/DCTEncode",
		"-dGrayImageFilter=/DCTEncode",
		"-dMonoImageFilter=/CCITTFaxEncode",
		"-dPreserveHalftoneInfo=false",
		fmt.Sprintf("-sOutputFile=%s", outputPath),
		inputPath,
	)

	// Execute Ghostscript
	if err := cmd.Run(); err != nil {
		p.logger.Error("Failed to optimize PDF with Ghostscript", zap.Error(err))
		return fmt.Errorf("ghostscript optimization failed: %w", err)
	}

	// Verify output was created
	if _, err := os.Stat(outputPath); err != nil {
		p.logger.Error("Output file not created", zap.Error(err))
		return fmt.Errorf("output file not created: %w", err)
	}

	// Log results
	originalInfo, _ := os.Stat(inputPath)
	optimizedInfo, _ := os.Stat(outputPath)
	reduction := 0.0
	if originalInfo.Size() > 0 {
		reduction = (1.0 - float64(optimizedInfo.Size())/float64(originalInfo.Size())) * 100
	}

	p.logger.Info("PDF optimized successfully with Ghostscript",
		zap.String("output", outputPath),
		zap.Int64("original_size", originalInfo.Size()),
		zap.Int64("optimized_size", optimizedInfo.Size()),
		zap.Float64("reduction_percent", reduction),
		zap.Int("dpi", dpi),
		zap.Int("quality", quality))

	return nil
}

// RenderPagesToImages renders PDF pages to PNG/JPEG images using Ghostscript
// This creates one image per page at specified DPI for lossless quality
// format: "png" (lossless) or "jpeg" (compressed)
// dpi: resolution (150-600, use 300+ for lossless quality)
func (p *Processor) RenderPagesToImages(inputPath string, outputDir string, format string, dpi int) error {
	p.logger.Info("Rendering PDF pages to images",
		zap.String("input", inputPath),
		zap.String("format", format),
		zap.Int("dpi", dpi))

	// Verify input file exists
	if _, err := os.Stat(inputPath); err != nil {
		p.logger.Error("Input file not found", zap.String("path", inputPath), zap.Error(err))
		return fmt.Errorf("input file not found: %s", inputPath)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		p.logger.Error("Failed to create output directory",
			zap.String("dir", outputDir),
			zap.Error(err))
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Validate parameters
	if dpi < 72 || dpi > 600 {
		dpi = 300 // Default to high quality for lossless
	}

	// Determine Ghostscript device based on format
	device := "png16m" // 16 million colors, lossless
	if format == "jpeg" || format == "jpg" {
		device = "jpeg"
	}

	// Output file pattern (page numbers will be added by Ghostscript)
	baseFilename := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	outputPattern := filepath.Join(outputDir, fmt.Sprintf("%s-page-%%03d.%s", baseFilename, format))

	// Build Ghostscript command for page rendering
	// -sDEVICE=png16m: PNG with 16 million colors (lossless)
	// -r<dpi>: Resolution in dots per inch
	// -dTextAlphaBits=4 -dGraphicsAlphaBits=4: Anti-aliasing for smooth edges
	cmd := exec.Command("gs",
		fmt.Sprintf("-sDEVICE=%s", device),
		"-dNOPAUSE",
		"-dBATCH",
		"-dSAFER",
		fmt.Sprintf("-r%d", dpi),
		"-dTextAlphaBits=4",
		"-dGraphicsAlphaBits=4",
		fmt.Sprintf("-sOutputFile=%s", outputPattern),
		inputPath,
	)

	// Execute Ghostscript
	if err := cmd.Run(); err != nil {
		p.logger.Error("Failed to render PDF pages", zap.Error(err))
		return fmt.Errorf("ghostscript page rendering failed: %w", err)
	}

	p.logger.Info("PDF pages rendered successfully",
		zap.String("output_dir", outputDir),
		zap.String("format", format),
		zap.Int("dpi", dpi))

	return nil
}

// RenderPagesToImagesWithProcessor renders PDF pages to images using specified processor
// processor: "ghostscript" (default) or "mupdf"
// format: "png" (lossless) or "jpeg" (compressed)
// dpi: resolution (150-600, use 300+ for lossless quality)
func (p *Processor) RenderPagesToImagesWithProcessor(inputPath string, outputDir string, format string, dpi int, processor string) error {
	p.logger.Info("Rendering PDF pages to images with specified processor",
		zap.String("input", inputPath),
		zap.String("format", format),
		zap.Int("dpi", dpi),
		zap.String("processor", processor))

	// Verify input file exists
	if _, err := os.Stat(inputPath); err != nil {
		p.logger.Error("Input file not found", zap.String("path", inputPath), zap.Error(err))
		return fmt.Errorf("input file not found: %s", inputPath)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		p.logger.Error("Failed to create output directory",
			zap.String("dir", outputDir),
			zap.Error(err))
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Validate parameters
	if dpi < 72 || dpi > 600 {
		dpi = 300 // Default to high quality for lossless
	}

	// Route to appropriate processor
	switch processor {
	case "mupdf", "mu":
		return p.renderPagesToImagesMuPDF(inputPath, outputDir, format, dpi)
	case "ghostscript", "gs", "":
		// Default to Ghostscript
		return p.RenderPagesToImages(inputPath, outputDir, format, dpi)
	default:
		p.logger.Error("Unknown processor", zap.String("processor", processor))
		return fmt.Errorf("unknown processor: %s (use 'ghostscript' or 'mupdf')", processor)
	}
}

// renderPagesToImagesMuPDF renders PDF pages using MuPDF
func (p *Processor) renderPagesToImagesMuPDF(inputPath string, outputDir string, format string, dpi int) error {
	// Check if mutool is available
	if err := CheckMuPDFAvailable(); err != nil {
		p.logger.Error("MuPDF not available", zap.Error(err))
		return fmt.Errorf("MuPDF (mutool) not installed: %w", err)
	}

	renderer := NewMuPDFRenderer(inputPath, outputDir, format, dpi, p.logger)

	if err := renderer.Render(); err != nil {
		return fmt.Errorf("MuPDF rendering failed: %w", err)
	}

	p.logger.Info("PDF pages rendered successfully with MuPDF",
		zap.String("output_dir", outputDir),
		zap.String("format", format),
		zap.Int("dpi", dpi))

	return nil
}

// ExportToImages extracts embedded images from PDF (legacy method)
// Note: This extracts embedded images, not renders pages
// For page rendering, use RenderPagesToImages instead
func (p *Processor) ExportToImages(inputPath string, outputDir string, format string, dpi int) error {
	p.logger.Info("Exporting images from PDF",
		zap.String("input", inputPath),
		zap.String("format", format),
		zap.Int("dpi", dpi))

	// Verify input file exists
	if _, err := os.Stat(inputPath); err != nil {
		p.logger.Error("Input file not found", zap.String("path", inputPath), zap.Error(err))
		return fmt.Errorf("input file not found: %s", inputPath)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		p.logger.Error("Failed to create output directory",
			zap.String("dir", outputDir),
			zap.Error(err))
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Extract images from PDF
	// This extracts embedded images, not rendered pages
	// ExtractImagesFile(inFile, outDir, selectedPages, conf)
	if err := api.ExtractImagesFile(inputPath, outputDir, nil, nil); err != nil {
		p.logger.Error("Failed to extract images from PDF", zap.Error(err))
		return fmt.Errorf("failed to extract images from PDF: %w", err)
	}

	p.logger.Info("Images extracted from PDF successfully",
		zap.String("output_dir", outputDir))

	return nil
}

// AddTableOfContents adds bookmarks/TOC to a PDF
func (p *Processor) AddTableOfContents(inputPath string, outputPath string, entries []TOCEntry) error {
	p.logger.Info("Adding table of contents to PDF",
		zap.String("input", inputPath),
		zap.Int("entries", len(entries)))

	// Verify input file exists
	if _, err := os.Stat(inputPath); err != nil {
		p.logger.Error("Input file not found", zap.String("path", inputPath), zap.Error(err))
		return fmt.Errorf("input file not found: %s", inputPath)
	}

	// Convert TOCEntry to pdfcpu Bookmark structure
	bookmarks := convertTOCToBookmarks(entries)

	// Add bookmarks using pdfcpu
	if err := api.AddBookmarksFile(inputPath, outputPath, bookmarks, false, nil); err != nil {
		p.logger.Error("Failed to add bookmarks", zap.Error(err))
		return fmt.Errorf("failed to add bookmarks: %w", err)
	}

	p.logger.Info("Table of contents added successfully",
		zap.String("output", outputPath),
		zap.Int("bookmarks_added", len(bookmarks)))

	return nil
}

// GetMetadata extracts metadata from a PDF
func (p *Processor) GetMetadata(inputPath string) (map[string]interface{}, error) {
	p.logger.Info("Getting PDF metadata", zap.String("input", inputPath))

	// Verify input file exists
	if _, err := os.Stat(inputPath); err != nil {
		p.logger.Error("Input file not found", zap.String("path", inputPath), zap.Error(err))
		return nil, fmt.Errorf("input file not found: %s", inputPath)
	}

	metadata := make(map[string]interface{})

	// Get page count
	pageCount, err := api.PageCountFile(inputPath)
	if err != nil {
		p.logger.Error("Failed to get page count", zap.Error(err))
		return nil, fmt.Errorf("failed to get page count: %w", err)
	}
	metadata["page_count"] = pageCount

	// Get file info
	fileInfo, _ := os.Stat(inputPath)
	metadata["file_size"] = fileInfo.Size()
	metadata["file_name"] = filepath.Base(inputPath)
	metadata["modified_time"] = fileInfo.ModTime().Unix()

	// Get PDF properties (title, author, subject, etc.)
	props, err := api.Properties(nil, nil)
	if err == nil {
		metadata["properties"] = props
	}

	p.logger.Info("PDF metadata retrieved successfully",
		zap.Int("page_count", pageCount),
		zap.Int64("file_size", fileInfo.Size()))

	return metadata, nil
}

// TOCEntry represents an entry in a table of contents
type TOCEntry struct {
	Title string
	Page  int
	Level int
}

// convertTOCToBookmarks converts TOCEntry to pdfcpu Bookmark structure
func convertTOCToBookmarks(entries []TOCEntry) []pdfcpu.Bookmark {
	bookmarks := make([]pdfcpu.Bookmark, 0)

	// Group entries by level to create hierarchy
	for _, entry := range entries {
		bookmark := pdfcpu.Bookmark{
			Title:    entry.Title,
			PageFrom: entry.Page,
		}
		bookmarks = append(bookmarks, bookmark)
	}

	return bookmarks
}
