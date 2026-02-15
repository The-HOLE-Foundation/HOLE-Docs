package annotations

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/The-HOLE-Foundation/hole-docs/internal/pdf"
	"go.uber.org/zap"
)

// OCRExtractor renders PDF pages to images (with annotations visible) and
// runs Tesseract OCR to capture the full visible text including annotation content.
type OCRExtractor struct {
	logger *zap.Logger
}

// NewOCRExtractor creates a new OCR-based annotation text extractor.
func NewOCRExtractor(logger *zap.Logger) *OCRExtractor {
	return &OCRExtractor{logger: logger}
}

// CheckDependencies verifies that mutool and tesseract are available.
func CheckDependencies() error {
	if err := pdf.CheckMuPDFAvailable(); err != nil {
		return fmt.Errorf("MuPDF required for annotation OCR: %w", err)
	}
	if _, err := exec.LookPath("tesseract"); err != nil {
		return fmt.Errorf("tesseract not found in PATH. Install: brew install tesseract")
	}
	return nil
}

// RenderAndOCR renders all pages of the PDF to images using MuPDF (which renders
// annotations visibly), then OCRs each page image with Tesseract.
// Returns a map of page number (1-indexed) to OCR'd text.
func (o *OCRExtractor) RenderAndOCR(filePath string) (map[int]string, error) {
	if err := CheckDependencies(); err != nil {
		return nil, err
	}

	// Create temp directory for rendered images
	tempDir, err := os.MkdirTemp("", "godocs-annot-ocr-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	o.logger.Info("Rendering PDF pages for OCR",
		zap.String("file", filepath.Base(filePath)),
		zap.String("temp_dir", tempDir))

	// Render pages using MuPDF at 200 DPI (good balance of quality and speed for OCR)
	renderer := pdf.NewMuPDFRenderer(filePath, tempDir, "png", 200, o.logger)
	if err := renderer.Render(); err != nil {
		return nil, fmt.Errorf("MuPDF render: %w", err)
	}

	// Collect rendered page images (MuPDF creates page001.png, page002.png, ...)
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return nil, fmt.Errorf("read temp dir: %w", err)
	}

	var pageImages []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".png") {
			pageImages = append(pageImages, filepath.Join(tempDir, entry.Name()))
		}
	}
	sort.Strings(pageImages)

	if len(pageImages) == 0 {
		return nil, fmt.Errorf("MuPDF produced no page images")
	}

	o.logger.Info("OCR'ing rendered pages",
		zap.Int("pages", len(pageImages)))

	result := make(map[int]string, len(pageImages))
	for _, imgPath := range pageImages {
		pageNr := pageNumberFromFilename(filepath.Base(imgPath))
		if pageNr == 0 {
			continue
		}

		text, err := o.ocrPage(imgPath)
		if err != nil {
			o.logger.Warn("OCR failed for page, skipping",
				zap.Int("page", pageNr),
				zap.Error(err))
			continue
		}
		result[pageNr] = text
	}

	o.logger.Info("OCR extraction complete",
		zap.Int("pages_ocrd", len(result)))

	return result, nil
}

// ocrPage runs Tesseract on a single page image and returns the extracted text.
func (o *OCRExtractor) ocrPage(imgPath string) (string, error) {
	cmd := exec.Command("tesseract", imgPath, "stdout")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("tesseract failed: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// pageNumberFromFilename extracts the page number from filenames like "page001.png".
func pageNumberFromFilename(name string) int {
	name = strings.TrimSuffix(name, filepath.Ext(name))
	name = strings.TrimPrefix(name, "page")
	n, err := strconv.Atoi(name)
	if err != nil {
		return 0
	}
	return n
}
