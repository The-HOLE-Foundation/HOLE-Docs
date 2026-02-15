package pdfa

import (
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

// TestPDFWriterInit tests PDFWriter initialization
func TestPDFWriterInit(t *testing.T) {
	logger, _ := zap.NewProduction()
	config := NewDefaultPDFA2bConfig()
	writer := NewPDFWriter(config, logger)

	if writer == nil {
		t.Error("Failed to create PDFWriter")
	}
	if writer.config != config {
		t.Error("Config not properly set")
	}
}

func TestPDFWriterInitWithNilConfig(t *testing.T) {
	logger, _ := zap.NewProduction()
	writer := NewPDFWriter(nil, logger)

	if writer == nil {
		t.Error("Failed to create PDFWriter with nil config")
	}
	if writer.config == nil {
		t.Error("Default config not created for nil input")
	}
}

func TestPDFWriterInitWithNilLogger(t *testing.T) {
	config := NewDefaultPDFA2bConfig()
	writer := NewPDFWriter(config, nil)

	if writer == nil {
		t.Error("Failed to create PDFWriter with nil logger")
	}
	if writer.logger == nil {
		t.Error("Default logger not created for nil input")
	}
}

// TestMergeImagesToPDFNoImages tests PDF creation with no images
func TestMergeImagesToPDFNoImages(t *testing.T) {
	logger, _ := zap.NewProduction()
	writer := NewPDFWriter(NewDefaultPDFA2bConfig(), logger)

	tmpDir, err := os.MkdirTemp("", "pdf_test_")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	outputPath := filepath.Join(tmpDir, "output.pdf")

	// Should fail with empty image list
	err = writer.MergeImagesToPDF([]string{}, outputPath)
	if err == nil {
		t.Error("MergeImagesToPDF should fail with empty image list")
	}
}

func TestMergeImagesToPDFNonexistentImage(t *testing.T) {
	logger, _ := zap.NewProduction()
	writer := NewPDFWriter(NewDefaultPDFA2bConfig(), logger)

	tmpDir, err := os.MkdirTemp("", "pdf_test_")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	outputPath := filepath.Join(tmpDir, "output.pdf")

	// Should fail with nonexistent image
	err = writer.MergeImagesToPDF(
		[]string{"/nonexistent/path/image.jpg"},
		outputPath,
	)
	if err == nil {
		t.Error("MergeImagesToPDF should fail with nonexistent image")
	}
}

// TestValidateAndCreateCompliantPDFNoImages tests validation with no images
func TestValidateAndCreateCompliantPDFNoImages(t *testing.T) {
	logger, _ := zap.NewProduction()
	writer := NewPDFWriter(NewDefaultPDFA2bConfig(), logger)

	tmpDir, err := os.MkdirTemp("", "pdf_test_")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	outputPath := filepath.Join(tmpDir, "output.pdf")

	// Should fail with no images
	_, err = writer.ValidateAndCreateCompliantPDF([]string{}, outputPath)
	if err == nil {
		t.Error("ValidateAndCreateCompliantPDF should fail with no images")
	}
}

// TestGetMergeProgress tests progress information
func TestGetMergeProgress(t *testing.T) {
	logger, _ := zap.NewProduction()
	config := NewDefaultPDFA2bConfig()
	config.TargetDPI = 150
	writer := NewPDFWriter(config, logger)

	imagePaths := []string{"img1.jpg", "img2.jpg", "img3.jpg"}
	progress := writer.GetMergeProgress(imagePaths)

	if progress == nil {
		t.Error("Progress info is nil")
	}

	totalImages := progress["total_images"]
	if totalImages != 3 {
		t.Errorf("Expected 3 images in progress, got %v", totalImages)
	}

	configInfo := progress["config"]
	if configInfo == nil {
		t.Error("Config info in progress is nil")
	}
}

// TestGetImageInfo tests image information retrieval
func TestGetImageInfoNonexistentFile(t *testing.T) {
	logger, _ := zap.NewProduction()
	writer := NewPDFWriter(NewDefaultPDFA2bConfig(), logger)

	_, err := writer.GetImageInfo("/nonexistent/path/image.jpg")
	if err == nil {
		t.Error("GetImageInfo should fail for nonexistent file")
	}
}

// Test writer initialization with different configs
func TestWriterWithHighQualityConfig(t *testing.T) {
	logger, _ := zap.NewProduction()
	config := NewHighQualityPDFA2bConfig()
	writer := NewPDFWriter(config, logger)

	if writer == nil {
		t.Error("Failed to create writer with high quality config")
	}
	if writer.config.TargetDPI != 300 {
		t.Errorf("Expected DPI 300, got %d", writer.config.TargetDPI)
	}
}

func TestWriterWithCompressedConfig(t *testing.T) {
	logger, _ := zap.NewProduction()
	config := NewCompressedPDFA2bConfig()
	writer := NewPDFWriter(config, logger)

	if writer == nil {
		t.Error("Failed to create writer with compressed config")
	}
	if writer.config.TargetDPI != 100 {
		t.Errorf("Expected DPI 100, got %d", writer.config.TargetDPI)
	}
}

// Test progress with different image counts
func TestGetMergeProgressVariousImageCounts(t *testing.T) {
	logger, _ := zap.NewProduction()
	writer := NewPDFWriter(NewDefaultPDFA2bConfig(), logger)

	testCases := []int{0, 1, 5, 10, 100}

	for _, count := range testCases {
		imagePaths := make([]string, count)
		for i := 0; i < count; i++ {
			imagePaths[i] = "image.jpg"
		}

		progress := writer.GetMergeProgress(imagePaths)
		total := progress["total_images"].(int)
		if total != count {
			t.Errorf("Expected %d images, got %d", count, total)
		}
	}
}
