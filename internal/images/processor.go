package images

import (
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"

	"github.com/The-HOLE-Foundation/hole-docs/internal/storage"
	"github.com/disintegration/imaging"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"go.uber.org/zap"
)

// Processor handles image operations
type Processor struct {
	cache  *storage.Cache
	logger *zap.Logger
}

// NewProcessor creates a new image processor
func NewProcessor(cache *storage.Cache, logger *zap.Logger) *Processor {
	return &Processor{
		cache:  cache,
		logger: logger,
	}
}

// ConvertToPDF converts image files to a PDF document
func (p *Processor) ConvertToPDF(inputPaths []string, outputPath string) error {
	p.logger.Info("Converting images to PDF", zap.Int("count", len(inputPaths)))

	// Validate inputs
	if len(inputPaths) == 0 {
		return fmt.Errorf("at least 1 image file required for PDF conversion")
	}

	// Verify all input files exist
	for _, path := range inputPaths {
		if _, err := os.Stat(path); err != nil {
			p.logger.Error("Input file not found", zap.String("path", path), zap.Error(err))
			return fmt.Errorf("input file not found: %s", path)
		}
	}

	// Use pdfcpu's ImportImagesFile to create PDF from images
	// Pass nil for import config to use defaults (scale: 1.0, fit all on page)
	if err := api.ImportImagesFile(inputPaths, outputPath, nil, nil); err != nil {
		p.logger.Error("Failed to convert images to PDF", zap.Error(err))
		return fmt.Errorf("failed to convert images to PDF: %w", err)
	}

	// Log success with file size
	fileInfo, _ := os.Stat(outputPath)
	p.logger.Info("Images converted to PDF successfully",
		zap.String("output", outputPath),
		zap.Int64("size", fileInfo.Size()))

	return nil
}

// MergeImages merges multiple images into a PDF or single image
func (p *Processor) MergeImages(inputPaths []string, outputPath string, format string) error {
	p.logger.Info("Merging images",
		zap.Int("count", len(inputPaths)),
		zap.String("format", format))

	// Validate inputs
	if len(inputPaths) < 2 {
		return fmt.Errorf("at least 2 image files required for merging, got %d", len(inputPaths))
	}

	// Check output format
	outputFormat := strings.ToLower(format)
	if outputFormat != "pdf" && outputFormat != "png" && outputFormat != "jpeg" && outputFormat != "jpg" {
		return fmt.Errorf("unsupported output format: %s (supported: pdf, png, jpeg)", format)
	}

	// If output format is PDF, use ConvertToPDF
	if outputFormat == "pdf" {
		return p.ConvertToPDF(inputPaths, outputPath)
	}

	// For image merge, create a composite image
	// Load first image to get dimensions
	firstImg, err := imaging.Open(inputPaths[0])
	if err != nil {
		p.logger.Error("Failed to open first image", zap.String("path", inputPaths[0]), zap.Error(err))
		return fmt.Errorf("failed to open image: %w", err)
	}

	firstBounds := firstImg.Bounds()
	width := firstBounds.Dx()
	height := firstBounds.Dy()

	// Stack images vertically
	totalHeight := height * len(inputPaths)
	// Use a basic image.Image interface - will convert to RGBA for saving
	var composite image.Image = image.NewRGBA(image.Rect(0, 0, width, totalHeight))

	// Paste each image
	yOffset := 0
	for _, imgPath := range inputPaths {
		img, err := imaging.Open(imgPath)
		if err != nil {
			p.logger.Error("Failed to open image", zap.String("path", imgPath), zap.Error(err))
			return fmt.Errorf("failed to open image: %w", err)
		}

		// Resize to match width if needed
		if img.Bounds().Dx() != width {
			img = imaging.Resize(img, width, 0, imaging.Lanczos)
		}

		// Paste onto composite - imaging.Paste works with any image.Image
		composited := imaging.Paste(composite, img, image.Pt(0, yOffset))
		composite = composited
		yOffset += img.Bounds().Dy()
	}

	// Save composite image
	if err := imaging.Save(composite, outputPath); err != nil {
		p.logger.Error("Failed to save merged image", zap.String("output", outputPath), zap.Error(err))
		return fmt.Errorf("failed to save merged image: %w", err)
	}

	fileInfo, _ := os.Stat(outputPath)
	p.logger.Info("Images merged successfully",
		zap.String("output", outputPath),
		zap.Int64("size", fileInfo.Size()))

	return nil
}

// ConvertFormat converts an image from one format to another
func (p *Processor) ConvertFormat(inputPath string, outputPath string, targetFormat string, quality int) error {
	p.logger.Info("Converting image format",
		zap.String("input", inputPath),
		zap.String("target", targetFormat),
		zap.Int("quality", quality))

	// Validate quality
	if quality < 1 || quality > 100 {
		quality = 85 // Default quality
	}

	// Verify input file exists
	if _, err := os.Stat(inputPath); err != nil {
		p.logger.Error("Input file not found", zap.String("path", inputPath), zap.Error(err))
		return fmt.Errorf("input file not found: %s", inputPath)
	}

	// Load image
	img, err := imaging.Open(inputPath)
	if err != nil {
		p.logger.Error("Failed to open image", zap.String("path", inputPath), zap.Error(err))
		return fmt.Errorf("failed to open image: %w", err)
	}

	// Save in target format
	targetFormat = strings.ToLower(targetFormat)
	switch targetFormat {
	case "jpeg", "jpg":
		// Convert to JPEG
		outFile, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer outFile.Close()

		opts := &jpeg.Options{Quality: quality}
		if err := jpeg.Encode(outFile, img, opts); err != nil {
			p.logger.Error("Failed to encode JPEG", zap.Error(err))
			return fmt.Errorf("failed to encode JPEG: %w", err)
		}

	case "png":
		// Convert to PNG
		if err := imaging.Save(img, outputPath); err != nil {
			p.logger.Error("Failed to save PNG", zap.Error(err))
			return fmt.Errorf("failed to save PNG: %w", err)
		}

	case "webp":
		// Note: WebP support would require additional library (libwebp)
		// For now, we'll save as PNG as fallback
		p.logger.Warn("WebP format requested but not supported, saving as PNG instead")
		if err := imaging.Save(img, strings.TrimSuffix(outputPath, filepath.Ext(outputPath))+".png"); err != nil {
			return fmt.Errorf("failed to save image: %w", err)
		}

	default:
		return fmt.Errorf("unsupported target format: %s", targetFormat)
	}

	fileInfo, _ := os.Stat(outputPath)
	p.logger.Info("Image format converted successfully",
		zap.String("output", outputPath),
		zap.String("format", targetFormat),
		zap.Int64("size", fileInfo.Size()))

	return nil
}

// GetImageMetadata extracts metadata from an image
func (p *Processor) GetImageMetadata(inputPath string) (map[string]interface{}, error) {
	p.logger.Info("Getting image metadata", zap.String("input", inputPath))

	// Verify input file exists
	if _, err := os.Stat(inputPath); err != nil {
		p.logger.Error("Input file not found", zap.String("path", inputPath), zap.Error(err))
		return nil, fmt.Errorf("input file not found: %s", inputPath)
	}

	// Load image to get dimensions
	img, err := imaging.Open(inputPath)
	if err != nil {
		p.logger.Error("Failed to open image", zap.String("path", inputPath), zap.Error(err))
		return nil, fmt.Errorf("failed to open image: %w", err)
	}

	bounds := img.Bounds()
	fileInfo, _ := os.Stat(inputPath)

	metadata := map[string]interface{}{
		"width":          bounds.Dx(),
		"height":         bounds.Dy(),
		"file_size":      fileInfo.Size(),
		"file_name":      filepath.Base(inputPath),
		"file_extension": filepath.Ext(inputPath),
		"modified_time":  fileInfo.ModTime().Unix(),
		"aspect_ratio":   float64(bounds.Dx()) / float64(bounds.Dy()),
	}

	p.logger.Info("Image metadata retrieved successfully",
		zap.Int("width", bounds.Dx()),
		zap.Int("height", bounds.Dy()),
		zap.Int64("file_size", fileInfo.Size()))

	return metadata, nil
}

// ResizeImage resizes an image to specified dimensions
func (p *Processor) ResizeImage(inputPath string, outputPath string, width int, height int) error {
	p.logger.Info("Resizing image",
		zap.String("input", inputPath),
		zap.Int("width", width),
		zap.Int("height", height))

	// Validate dimensions
	if width <= 0 || height <= 0 {
		return fmt.Errorf("invalid dimensions: width and height must be > 0, got width=%d, height=%d", width, height)
	}

	// Verify input file exists
	if _, err := os.Stat(inputPath); err != nil {
		p.logger.Error("Input file not found", zap.String("path", inputPath), zap.Error(err))
		return fmt.Errorf("input file not found: %s", inputPath)
	}

	// Load image
	img, err := imaging.Open(inputPath)
	if err != nil {
		p.logger.Error("Failed to open image", zap.String("path", inputPath), zap.Error(err))
		return fmt.Errorf("failed to open image: %w", err)
	}

	// Resize image using Lanczos filter for high quality
	resized := imaging.Resize(img, width, height, imaging.Lanczos)

	// Save resized image
	if err := imaging.Save(resized, outputPath); err != nil {
		p.logger.Error("Failed to save resized image", zap.String("output", outputPath), zap.Error(err))
		return fmt.Errorf("failed to save resized image: %w", err)
	}

	originalBounds := img.Bounds()
	fileInfo, _ := os.Stat(outputPath)
	p.logger.Info("Image resized successfully",
		zap.String("output", outputPath),
		zap.Int("original_width", originalBounds.Dx()),
		zap.Int("original_height", originalBounds.Dy()),
		zap.Int("new_width", width),
		zap.Int("new_height", height),
		zap.Int64("output_size", fileInfo.Size()))

	return nil
}
