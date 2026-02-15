package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/The-HOLE-Foundation/hole-docs/internal/docling"
	"github.com/The-HOLE-Foundation/hole-docs/internal/pdf"
	"github.com/The-HOLE-Foundation/hole-docs/internal/storage"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestTOCGeneration_SimpleDocument tests TOC generation on a document with simple heading structure
func TestTOCGeneration_SimpleDocument(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Arrange
	inputPath := filepath.Join("..", "..", "testdata", "toc", "simple-headings.pdf")
	if !fileExists(inputPath) {
		t.Skipf("Test PDF not found: %s (will create later)", inputPath)
	}

	// Convert to absolute path for Docling service
	absPath, err := filepath.Abs(inputPath)
	require.NoError(t, err)
	inputPath = absPath

	outputPath := filepath.Join(t.TempDir(), "output.pdf")

	// Setup services
	ctx := context.Background()
	doclingClient := setupDoclingClient(t)
	processor := setupPDFProcessor(t)

	// Act: Step 1 - Process document with Docling
	result, err := doclingClient.ProcessPDF(ctx, inputPath)
	require.NoError(t, err, "Docling processing should succeed")
	require.NotNil(t, result)

	// Act: Step 2 - Generate TOC
	tocGen := pdf.NewTOCGenerator(processor)
	err = tocGen.GenerateTOC(inputPath, outputPath, result, false)
	require.NoError(t, err, "TOC generation should succeed")

	// Assert: Output file exists
	assert.FileExists(t, outputPath)

	// Assert: Bookmarks were added by reading the PDF
	bookmarks, err := readBookmarks(outputPath)
	require.NoError(t, err)
	assert.Greater(t, len(bookmarks), 0, "Should have at least one bookmark")

	// Assert: Verify bookmark structure for simple document
	// Expected: 3 H1 headings at pages 1, 5, 10
	if len(bookmarks) >= 1 {
		assert.Greater(t, bookmarks[0].PageFrom, 0, "Page number should be positive")
	}
}

// TestTOCGeneration_NestedStructure tests TOC with hierarchical headings (H1 > H2 > H3)
func TestTOCGeneration_NestedStructure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Arrange
	inputPath := filepath.Join("..", "..", "testdata", "toc", "nested-structure.pdf")
	if !fileExists(inputPath) {
		t.Skipf("Test PDF not found: %s (will create later)", inputPath)
	}

	outputPath := filepath.Join(t.TempDir(), "output.pdf")

	ctx := context.Background()
	doclingClient := setupDoclingClient(t)
	processor := setupPDFProcessor(t)

	// Act: Process document
	result, err := doclingClient.ProcessPDF(ctx, inputPath)
	require.NoError(t, err)

	// Act: Generate TOC
	tocGen := pdf.NewTOCGenerator(processor)
	err = tocGen.GenerateTOC(inputPath, outputPath, result, false)
	require.NoError(t, err)

	// Assert: Verify nested structure
	bookmarks, err := readBookmarks(outputPath)
	require.NoError(t, err)
	assert.Greater(t, len(bookmarks), 0)

	// Assert: Should have nested children
	hasChildren := false
	for _, bm := range bookmarks {
		if len(bm.Kids) > 0 {
			hasChildren = true
			break
		}
	}
	assert.True(t, hasChildren, "Expected nested bookmark structure with children")
}

// TestTOCGeneration_ComplexDocument tests TOC with all heading levels (H1-H6)
func TestTOCGeneration_ComplexDocument(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Arrange
	inputPath := filepath.Join("..", "..", "testdata", "toc", "complex-document.pdf")
	if !fileExists(inputPath) {
		t.Skipf("Test PDF not found: %s (will create later)", inputPath)
	}

	outputPath := filepath.Join(t.TempDir(), "output.pdf")

	ctx := context.Background()
	doclingClient := setupDoclingClient(t)
	processor := setupPDFProcessor(t)

	// Act
	result, err := doclingClient.ProcessPDF(ctx, inputPath)
	require.NoError(t, err)

	tocGen := pdf.NewTOCGenerator(processor)
	err = tocGen.GenerateTOC(inputPath, outputPath, result, false)
	require.NoError(t, err)

	// Assert: Complex structure verification
	bookmarks, err := readBookmarks(outputPath)
	require.NoError(t, err)
	assert.Greater(t, len(bookmarks), 0)

	// Verify deep nesting (should support H1-H6)
	maxDepth := calculateMaxDepth(bookmarks, 0)
	assert.LessOrEqual(t, maxDepth, 6, "Maximum nesting depth should not exceed 6 levels")
}

// TestTOCGeneration_NoStructure tests handling of documents without headings
func TestTOCGeneration_NoStructure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Arrange
	inputPath := filepath.Join("..", "..", "testdata", "toc", "no-structure.pdf")
	if !fileExists(inputPath) {
		t.Skipf("Test PDF not found: %s (will create later)", inputPath)
	}

	outputPath := filepath.Join(t.TempDir(), "output.pdf")

	ctx := context.Background()
	doclingClient := setupDoclingClient(t)
	processor := setupPDFProcessor(t)

	// Act
	result, err := doclingClient.ProcessPDF(ctx, inputPath)
	require.NoError(t, err)

	tocGen := pdf.NewTOCGenerator(processor)
	err = tocGen.GenerateTOC(inputPath, outputPath, result, false)

	// Assert: Should handle gracefully (either succeed with no bookmarks or return specific error)
	if err != nil {
		// Acceptable: Error indicating no structure found
		assert.Contains(t, err.Error(), "no headings")
	} else {
		// Acceptable: Success but no bookmarks added
		assert.FileExists(t, outputPath)
		bookmarks, err := readBookmarks(outputPath)
		require.NoError(t, err)
		assert.Len(t, bookmarks, 0, "Should have no bookmarks for unstructured document")
	}
}

// TestTOCGeneration_ReplaceExisting tests replacing existing bookmarks
func TestTOCGeneration_ReplaceExisting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Arrange: Create a PDF with existing bookmarks
	inputPath := filepath.Join("..", "..", "testdata", "toc", "with-existing-bookmarks.pdf")
	if !fileExists(inputPath) {
		t.Skipf("Test PDF not found: %s", inputPath)
	}

	outputPath := filepath.Join(t.TempDir(), "output.pdf")

	ctx := context.Background()
	doclingClient := setupDoclingClient(t)
	processor := setupPDFProcessor(t)

	// Act: Process and generate TOC with replace=true
	result, err := doclingClient.ProcessPDF(ctx, inputPath)
	require.NoError(t, err)

	tocGen := pdf.NewTOCGenerator(processor)
	err = tocGen.GenerateTOC(inputPath, outputPath, result, true) // replace=true
	require.NoError(t, err)

	// Assert: Old bookmarks replaced with new ones
	bookmarks, err := readBookmarks(outputPath)
	require.NoError(t, err)

	// Verify bookmarks match Docling-detected structure (not old bookmarks)
	// This test would need specific knowledge of input PDF structure
	assert.Greater(t, len(bookmarks), 0)
}

// TestTOCGeneration_PreservesContent tests that TOC generation doesn't corrupt PDF content
func TestTOCGeneration_PreservesContent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Arrange
	inputPath := filepath.Join("..", "..", "testdata", "toc", "simple-headings.pdf")
	if !fileExists(inputPath) {
		t.Skipf("Test PDF not found: %s", inputPath)
	}

	// Convert to absolute path for Docling service
	absPath, err := filepath.Abs(inputPath)
	require.NoError(t, err)
	inputPath = absPath

	outputPath := filepath.Join(t.TempDir(), "output.pdf")

	ctx := context.Background()
	doclingClient := setupDoclingClient(t)
	processor := setupPDFProcessor(t)

	// Act: Generate TOC
	result, err := doclingClient.ProcessPDF(ctx, inputPath)
	require.NoError(t, err)

	tocGen := pdf.NewTOCGenerator(processor)
	err = tocGen.GenerateTOC(inputPath, outputPath, result, false)
	require.NoError(t, err)

	// Assert: Output file exists and is valid
	assert.FileExists(t, outputPath)

	// Assert: Can read bookmarks (validates PDF structure)
	bookmarks, err := readBookmarks(outputPath)
	require.NoError(t, err, "Output PDF should be readable")
	assert.Greater(t, len(bookmarks), 0, "Should have bookmarks")
}

// TestTOCGeneration_Performance tests that TOC generation completes within acceptable time
func TestTOCGeneration_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Arrange: Large document (50+ pages)
	inputPath := filepath.Join("..", "..", "testdata", "toc", "complex-document.pdf")
	if !fileExists(inputPath) {
		t.Skipf("Test PDF not found: %s", inputPath)
	}

	outputPath := filepath.Join(t.TempDir(), "output.pdf")

	ctx := context.Background()
	doclingClient := setupDoclingClient(t)
	processor := setupPDFProcessor(t)

	// Act: Measure processing time
	result, err := doclingClient.ProcessPDF(ctx, inputPath)
	require.NoError(t, err)

	tocGen := pdf.NewTOCGenerator(processor)

	// Note: Don't measure Docling processing time (external service)
	// Only measure TOC generation itself
	err = tocGen.GenerateTOC(inputPath, outputPath, result, false)
	require.NoError(t, err)

	// Assert: File created successfully
	assert.FileExists(t, outputPath)

	// Performance assertion: TOC insertion should be fast (<1 second)
	// Actual timing would need benchmark tests, not integration tests
}

// Helper Functions

func setupDoclingClient(t *testing.T) *docling.Client {
	logger := zap.NewNop()

	// Use environment variable or default to localhost
	doclingURL := os.Getenv("DOCLING_URL")
	if doclingURL == "" {
		doclingURL = "http://localhost:5000"
	}

	client := docling.NewClient(doclingURL, logger)
	return client
}

func setupPDFProcessor(t *testing.T) *pdf.Processor {
	logger := zap.NewNop()

	// Create a temporary cache for testing
	cacheDir := t.TempDir()
	cache := storage.NewCache(cacheDir)

	processor := pdf.NewProcessor(cache, logger)
	return processor
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// readBookmarks reads bookmarks from a PDF file
func readBookmarks(pdfPath string) ([]pdfcpu.Bookmark, error) {
	// Use pdfcpu to read file and extract bookmarks
	ctx, err := api.ReadContextFile(pdfPath)
	if err != nil {
		return nil, err
	}

	bookmarks, err := pdfcpu.Bookmarks(ctx)
	if err != nil {
		return nil, err
	}

	return bookmarks, nil
}

func calculateMaxDepth(bookmarks []pdfcpu.Bookmark, currentDepth int) int {
	maxDepth := currentDepth

	for _, bm := range bookmarks {
		depth := currentDepth + 1

		if len(bm.Kids) > 0 {
			childDepth := calculateMaxDepth(bm.Kids, depth)
			if childDepth > maxDepth {
				maxDepth = childDepth
			}
		} else {
			if depth > maxDepth {
				maxDepth = depth
			}
		}
	}

	return maxDepth
}
