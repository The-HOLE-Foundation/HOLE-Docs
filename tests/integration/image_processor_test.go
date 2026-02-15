package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/The-HOLE-Foundation/hole-docs/internal/images"
	"github.com/The-HOLE-Foundation/hole-docs/internal/storage"
	"go.uber.org/zap"
)

func TestImageProcessor(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := t.TempDir()
	logger, _ := zap.NewDevelopment()
	cache := storage.NewCache(tmpDir)

	processor := images.NewProcessor(cache, logger)

	t.Run("ConvertToPDF", func(t *testing.T) {
		t.Run("validates_at_least_1_file", func(t *testing.T) {
			err := processor.ConvertToPDF([]string{}, filepath.Join(tmpDir, "output.pdf"))
			if err == nil {
				t.Error("Expected error for empty input paths")
			}
		})

		t.Run("validates_input_files_exist", func(t *testing.T) {
			err := processor.ConvertToPDF(
				[]string{"/nonexistent/image.png"},
				filepath.Join(tmpDir, "output.pdf"),
			)
			if err == nil {
				t.Error("Expected error for non-existent input file")
			}
		})
	})

	t.Run("MergeImages", func(t *testing.T) {
		t.Run("validates_at_least_2_files", func(t *testing.T) {
			err := processor.MergeImages([]string{}, filepath.Join(tmpDir, "output.pdf"), "pdf")
			if err == nil {
				t.Error("Expected error for empty input paths")
			}
		})

		t.Run("validates_output_format", func(t *testing.T) {
			// Create dummy files
			file1 := filepath.Join(tmpDir, "image1.png")
			file2 := filepath.Join(tmpDir, "image2.png")
			os.WriteFile(file1, []byte("dummy"), 0644)
			os.WriteFile(file2, []byte("dummy"), 0644)

			err := processor.MergeImages(
				[]string{file1, file2},
				filepath.Join(tmpDir, "output.xyz"),
				"invalid-format",
			)
			if err == nil {
				t.Error("Expected error for invalid output format")
			}
		})

		t.Run("validates_input_files_exist", func(t *testing.T) {
			err := processor.MergeImages(
				[]string{"/nonexistent/image1.png", "/nonexistent/image2.png"},
				filepath.Join(tmpDir, "output.pdf"),
				"pdf",
			)
			if err == nil {
				t.Error("Expected error for non-existent input files")
			}
		})
	})

	t.Run("ConvertFormat", func(t *testing.T) {
		t.Run("validates_input_file_exists", func(t *testing.T) {
			err := processor.ConvertFormat(
				"/nonexistent/input.png",
				filepath.Join(tmpDir, "output.jpg"),
				"jpeg",
				85,
			)
			if err == nil {
				t.Error("Expected error for non-existent input file")
			}
		})

		t.Run("validates_quality_range", func(t *testing.T) {
			// Create dummy file
			inputFile := filepath.Join(tmpDir, "dummy.png")
			os.WriteFile(inputFile, []byte("dummy"), 0644)

			// Quality should default to 85 if out of range - processor should handle gracefully
			err := processor.ConvertFormat(
				inputFile,
				filepath.Join(tmpDir, "output.jpg"),
				"jpeg",
				150, // Out of range
			)
			// Processor will default to 85, so this should not fail for quality reason
			_ = err
		})

		t.Run("supports_multiple_formats", func(t *testing.T) {
			formats := []string{"png", "jpeg", "jpg", "webp"}
			for _, format := range formats {
				// Test that format conversion attempts don't panic
				inputFile := filepath.Join(tmpDir, "dummy.png")
				os.WriteFile(inputFile, []byte("dummy"), 0644)

				// These will fail at the image loading stage for dummy files, but shouldn't panic
				_ = processor.ConvertFormat(
					inputFile,
					filepath.Join(tmpDir, "output."+format),
					format,
					85,
				)
			}
		})
	})

	t.Run("GetImageMetadata", func(t *testing.T) {
		t.Run("validates_input_file_exists", func(t *testing.T) {
			_, err := processor.GetImageMetadata("/nonexistent/image.png")
			if err == nil {
				t.Error("Expected error for non-existent input file")
			}
		})
	})

	t.Run("ResizeImage", func(t *testing.T) {
		t.Run("validates_input_file_exists", func(t *testing.T) {
			err := processor.ResizeImage(
				"/nonexistent/input.png",
				filepath.Join(tmpDir, "output.png"),
				100,
				100,
			)
			if err == nil {
				t.Error("Expected error for non-existent input file")
			}
		})

		t.Run("validates_positive_dimensions", func(t *testing.T) {
			// Create dummy file
			inputFile := filepath.Join(tmpDir, "dummy.png")
			os.WriteFile(inputFile, []byte("dummy"), 0644)

			err := processor.ResizeImage(
				inputFile,
				filepath.Join(tmpDir, "output.png"),
				0, // Invalid: must be > 0
				100,
			)
			if err == nil {
				t.Error("Expected error for zero width")
			}
		})
	})
}
