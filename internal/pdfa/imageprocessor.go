package pdfa

import (
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/nfnt/resize"
	"go.uber.org/zap"
)

// ImageProcessor handles image analysis and downsampling
type ImageProcessor struct {
	config *PDFA2bConfig
	logger *zap.Logger
}

// NewImageProcessor creates a new image processor
func NewImageProcessor(config *PDFA2bConfig, logger *zap.Logger) *ImageProcessor {
	if config == nil {
		config = NewDefaultPDFA2bConfig()
	}
	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	return &ImageProcessor{
		config: config,
		logger: logger,
	}
}

// AnalyzeImage examines an image and determines if downsampling is needed
func (ip *ImageProcessor) AnalyzeImage(filepath string, estimatedDPI int) (*ImageMetadata, error) {
	ip.logger.Debug("Analyzing image", zap.String("path", filepath))

	// Open file
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	// Decode image config (without loading full image)
	config, format, err := image.DecodeConfig(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Get file info
	fileInfo, _ := os.Stat(filepath)

	// Analyze dimensions
	metadata := &ImageMetadata{
		FilePath:      filepath,
		Width:         config.Width,
		Height:        config.Height,
		ColorSpace:    "unknown",
		EstimatedDPI:  estimatedDPI,
		FileSize:      fileInfo.Size(),
		Format:        format,
		NeedsDownsampling: false,
	}

	// Check if downsampling is needed
	if estimatedDPI > ip.config.DownsampleThreshold {
		metadata.NeedsDownsampling = true

		// Calculate target dimensions
		scaleFactor := float64(ip.config.TargetDPI) / float64(estimatedDPI)
		metadata.NewWidth = int(float64(config.Width) * scaleFactor)
		metadata.NewHeight = int(float64(config.Height) * scaleFactor)

		ip.logger.Debug("Image downsampling needed",
			zap.String("file", filepath),
			zap.Int("current_dpi", estimatedDPI),
			zap.Int("target_dpi", ip.config.TargetDPI),
			zap.Int("original_width", config.Width),
			zap.Int("new_width", metadata.NewWidth))
	}

	return metadata, nil
}

// DownsampleImage applies downsampling to an image file
func (ip *ImageProcessor) DownsampleImage(inputPath, outputPath string, metadata *ImageMetadata) error {
	if !metadata.NeedsDownsampling {
		// No downsampling needed, copy file as-is
		ip.logger.Debug("No downsampling needed, copying file",
			zap.String("input", inputPath),
			zap.String("output", outputPath))
		return copyFile(inputPath, outputPath)
	}

	ip.logger.Info("Downsampling image",
		zap.String("input", inputPath),
		zap.String("method", string(ip.config.DownsamplingMethod)),
		zap.Int("from_width", metadata.Width),
		zap.Int("to_width", metadata.NewWidth))

	// Open and decode image
	file, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	// Apply downsampling based on method
	var resized image.Image
	switch ip.config.DownsamplingMethod {
	case DownsamplingMethodBicubic:
		// CatmullRom: high quality, slower
		resized = resize.Resize(uint(metadata.NewWidth), uint(metadata.NewHeight), img, resize.MitchellNetravali)
		ip.logger.Debug("Using Bicubic (MitchellNetravali) downsampling")

	case DownsamplingMethodAverage:
		// Bilinear: balanced quality/speed
		resized = resize.Resize(uint(metadata.NewWidth), uint(metadata.NewHeight), img, resize.Bilinear)
		ip.logger.Debug("Using Average (Bilinear) downsampling")

	case DownsamplingMethodSubsample:
		// NearestNeighbor: fast, lower quality
		resized = resize.Resize(uint(metadata.NewWidth), uint(metadata.NewHeight), img, resize.NearestNeighbor)
		ip.logger.Debug("Using Subsample (NearestNeighbor) downsampling")

	default:
		return fmt.Errorf("unknown downsampling method: %s", ip.config.DownsamplingMethod)
	}

	// Save downsampled image
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Encode based on input format
	switch strings.ToLower(metadata.Format) {
	case "jpeg", "jpg":
		err = encodeJPEG(outFile, resized, ip.config.ImageQuality)
	case "png":
		err = encodePNG(outFile, resized)
	case "gif":
		err = encodeGIF(outFile, resized)
	default:
		// Default to JPEG for unrecognized formats
		err = encodeJPEG(outFile, resized, ip.config.ImageQuality)
	}

	if err != nil {
		return fmt.Errorf("failed to encode image: %w", err)
	}

	return nil
}

// ProcessImages processes a batch of images with downsampling
func (ip *ImageProcessor) ProcessImages(inputPaths []string, outputDir string, estimatedDPI int) ([]string, []error) {
	results := make([]string, len(inputPaths))
	errs := make([]error, len(inputPaths))

	// Ensure output directory exists
	os.MkdirAll(outputDir, 0755)

	for i, inputPath := range inputPaths {
		// Analyze image
		metadata, err := ip.AnalyzeImage(inputPath, estimatedDPI)
		if err != nil {
			errs[i] = err
			ip.logger.Warn("Failed to analyze image",
				zap.String("path", inputPath),
				zap.Error(err))
			continue
		}

		// Generate output path
		basename := filepath.Base(inputPath)
		outputPath := filepath.Join(outputDir, basename)

		// Process image
		if err := ip.DownsampleImage(inputPath, outputPath, metadata); err != nil {
			errs[i] = err
			ip.logger.Error("Failed to process image",
				zap.String("path", inputPath),
				zap.Error(err))
			continue
		}

		results[i] = outputPath
		ip.logger.Info("Image processed successfully",
			zap.String("output", outputPath))
	}

	return results, errs
}

// Helper functions for encoding

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func encodeJPEG(file *os.File, img image.Image, quality int) error {
	// Clamp quality
	if quality < 1 {
		quality = 1
	} else if quality > 100 {
		quality = 100
	}

	return jpeg.Encode(file, img, &jpeg.Options{Quality: quality})
}

func encodePNG(file *os.File, img image.Image) error {
	return png.Encode(file, img)
}

func encodeGIF(file *os.File, img image.Image) error {
	return gif.Encode(file, img, &gif.Options{})
}

// DetectImageDPI attempts to detect DPI from image metadata
func DetectImageDPI(filepath string) int {
	// Note: This is simplified - real implementation would read EXIF/metadata
	// For now, return sensible default for scanned documents
	return 300 // Assume scanned documents at 300 DPI
}

// IsImageFile checks if a file is a supported image format
func IsImageFile(filepath string) bool {
	ext := strings.ToLower(filepath[len(filepath)-4:])
	return ext == ".png" || ext == ".jpg" || ext == ".gif" || ext == ".bmp" ||
		ext == "jpeg" || filepath[len(filepath)-5:] == ".tiff"
}

// IsPDFFile checks if a file is a PDF
func IsPDFFile(filepath string) bool {
	return strings.ToLower(filepath[len(filepath)-4:]) == ".pdf"
}
