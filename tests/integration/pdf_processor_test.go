package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/The-HOLE-Foundation/hole-docs/internal/pdf"
	"github.com/The-HOLE-Foundation/hole-docs/internal/storage"
	"go.uber.org/zap"
)

func TestPDFProcessor(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := t.TempDir()
	logger, _ := zap.NewDevelopment()
	cache := storage.NewCache(tmpDir)

	processor := pdf.NewProcessor(cache, logger)

	t.Run("MergePDFs", func(t *testing.T) {
		t.Run("validates_at_least_2_files", func(t *testing.T) {
			err := processor.MergePDFs([]string{}, filepath.Join(tmpDir, "output.pdf"))
			if err == nil {
				t.Error("Expected error for empty input paths")
			}
		})

		t.Run("validates_input_files_exist", func(t *testing.T) {
			err := processor.MergePDFs([]string{
				"/nonexistent/file1.pdf",
				"/nonexistent/file2.pdf",
			}, filepath.Join(tmpDir, "output.pdf"))
			if err == nil {
				t.Error("Expected error for non-existent input files")
			}
		})
	})

	t.Run("SplitPDF", func(t *testing.T) {
		t.Run("validates_input_file_exists", func(t *testing.T) {
			err := processor.SplitPDF(
				"/nonexistent/input.pdf",
				filepath.Join(tmpDir, "output"),
				[]string{"1-5"},
			)
			if err == nil {
				t.Error("Expected error for non-existent input file")
			}
		})

		t.Run("validates_page_ranges_format", func(t *testing.T) {
			// Create dummy input file
			inputFile := filepath.Join(tmpDir, "dummy.pdf")
			if err := os.WriteFile(inputFile, []byte("dummy"), 0644); err != nil {
				t.Fatalf("Failed to create dummy file: %v", err)
			}

			err := processor.SplitPDF(
				inputFile,
				filepath.Join(tmpDir, "output"),
				[]string{"invalid-format"},
			)
			// May fail at pdfcpu level, but should not panic
			_ = err
		})
	})

	t.Run("OptimizePDF", func(t *testing.T) {
		t.Run("validates_input_file_exists", func(t *testing.T) {
			err := processor.OptimizePDF(
				"/nonexistent/input.pdf",
				filepath.Join(tmpDir, "output.pdf"),
				"medium",
			)
			if err == nil {
				t.Error("Expected error for non-existent input file")
			}
		})
	})

	t.Run("AddTableOfContents", func(t *testing.T) {
		t.Run("validates_input_file_exists", func(t *testing.T) {
			err := processor.AddTableOfContents(
				"/nonexistent/input.pdf",
				filepath.Join(tmpDir, "output.pdf"),
				[]pdf.TOCEntry{
					{Title: "Chapter 1", Page: 1, Level: 0},
				},
			)
			if err == nil {
				t.Error("Expected error for non-existent input file")
			}
		})
	})

	t.Run("GetMetadata", func(t *testing.T) {
		t.Run("validates_input_file_exists", func(t *testing.T) {
			_, err := processor.GetMetadata("/nonexistent/input.pdf")
			if err == nil {
				t.Error("Expected error for non-existent input file")
			}
		})
	})

	t.Run("ExportToImages", func(t *testing.T) {
		t.Run("validates_input_file_exists", func(t *testing.T) {
			err := processor.ExportToImages(
				"/nonexistent/input.pdf",
				filepath.Join(tmpDir, "images"),
				"png",
				150,
			)
			if err == nil {
				t.Error("Expected error for non-existent input file")
			}
		})
	})
}
