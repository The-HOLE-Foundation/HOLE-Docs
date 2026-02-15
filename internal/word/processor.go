package word

import (
	"fmt"
	"os"
	"strings"

	"github.com/The-HOLE-Foundation/hole-docs/internal/storage"
	"go.uber.org/zap"
)

// UniOffice imports commented out - requires commercial license
// Using LibreOffice and native Go instead (completely free)
// import "github.com/unidoc/unioffice/document"
// import "github.com/unidoc/unioffice/document/convert"

// Processor handles Word document operations
type Processor struct {
	cache  *storage.Cache
	logger *zap.Logger
}

// NewProcessor creates a new Word document processor
func NewProcessor(cache *storage.Cache, logger *zap.Logger) *Processor {
	return &Processor{
		cache:  cache,
		logger: logger,
	}
}

// ConvertToPDF converts a Word document (.docx) to PDF format
// Uses LibreOffice headless mode (free, open source, no licensing required)
// Preserves formatting, tables, images, headers/footers
func (p *Processor) ConvertToPDF(inputPath string, outputPath string) error {
	// Use LibreOffice for conversion (free, no license required)
	return p.ConvertToPDFWithLibreOffice(inputPath, outputPath)
}

// ConvertToPDFWithUniOffice converts using UniOffice library (requires license)
// Commented out - requires commercial license for production use
// Use LibreOffice instead (see ConvertToPDFWithLibreOffice)
/*
func (p *Processor) ConvertToPDFWithUniOffice(inputPath string, outputPath string) error {
	p.logger.Info("Converting Word document to PDF with UniOffice",
		zap.String("input", inputPath),
		zap.String("output", outputPath))

	// Verify input file exists
	if _, err := os.Stat(inputPath); err != nil {
		p.logger.Error("Input file not found", zap.String("path", inputPath), zap.Error(err))
		return fmt.Errorf("input file not found: %s", inputPath)
	}

	// Open Word document
	doc, err := document.Open(inputPath)
	if err != nil {
		p.logger.Error("Failed to open Word document", zap.Error(err))
		return fmt.Errorf("failed to open Word document: %w", err)
	}
	defer doc.Close()

	// Convert to PDF
	pdfCreator := convert.ConvertToPdf(doc)

	// Write PDF to file
	if err := pdfCreator.WriteToFile(outputPath); err != nil {
		p.logger.Error("Failed to write PDF", zap.String("output", outputPath), zap.Error(err))
		return fmt.Errorf("failed to write PDF: %w", err)
	}

	// Log success with file sizes
	originalInfo, _ := os.Stat(inputPath)
	pdfInfo, _ := os.Stat(outputPath)

	p.logger.Info("Word document converted to PDF successfully",
		zap.String("output", outputPath),
		zap.Int64("original_size", originalInfo.Size()),
		zap.Int64("pdf_size", pdfInfo.Size()))

	return nil
}
*/

// ExtractText extracts all text content from a Word document
// Uses native Go ZIP/XML parsing (no external library or license required)
func (p *Processor) ExtractText(inputPath string) (string, error) {
	// Use free native Go extraction
	return p.ExtractTextFree(inputPath)
}

// ExtractTextWithUniOffice extracts text using UniOffice library (requires license)
// Commented out - requires commercial license
// Use ExtractTextFree instead (native Go, no license needed)
/*
func (p *Processor) ExtractTextWithUniOffice(inputPath string) (string, error) {
	p.logger.Info("Extracting text from Word document", zap.String("input", inputPath))

	// Verify input file exists
	if _, err := os.Stat(inputPath); err != nil {
		p.logger.Error("Input file not found", zap.String("path", inputPath), zap.Error(err))
		return "", fmt.Errorf("input file not found: %s", inputPath)
	}

	// Open Word document
	doc, err := document.Open(inputPath)
	if err != nil {
		p.logger.Error("Failed to open Word document", zap.Error(err))
		return "", fmt.Errorf("failed to open Word document: %w", err)
	}
	defer doc.Close()

	var textContent strings.Builder

	// Extract text from paragraphs
	for _, para := range doc.Paragraphs() {
		for _, run := range para.Runs() {
			textContent.WriteString(run.Text())
		}
		textContent.WriteString("\n")
	}

	// Extract text from tables
	for _, tbl := range doc.Tables() {
		for _, row := range tbl.Rows() {
			for i, cell := range row.Cells() {
				if i > 0 {
					textContent.WriteString("\t")
				}
				for _, para := range cell.Paragraphs() {
					for _, run := range para.Runs() {
						textContent.WriteString(run.Text())
					}
				}
			}
			textContent.WriteString("\n")
		}
		textContent.WriteString("\n")
	}

	text := textContent.String()

	p.logger.Info("Text extracted successfully",
		zap.Int("length", len(text)))

	return text, nil
}
*/

// GetMetadata extracts metadata from a Word document
// Uses basic file stats (no external library required)
func (p *Processor) GetMetadata(inputPath string) (map[string]interface{}, error) {
	p.logger.Info("Getting Word document metadata", zap.String("input", inputPath))

	// Verify input file exists
	if _, err := os.Stat(inputPath); err != nil {
		p.logger.Error("Input file not found", zap.String("path", inputPath), zap.Error(err))
		return nil, fmt.Errorf("input file not found: %s", inputPath)
	}

	// Get file info
	fileInfo, _ := os.Stat(inputPath)

	// Build basic metadata
	metadata := map[string]interface{}{
		"file_name":     fileInfo.Name(),
		"file_size":     fileInfo.Size(),
		"modified_time": fileInfo.ModTime().Unix(),
		"format":        "docx",
	}

	// Extract text to get character count
	text, err := p.ExtractTextFree(inputPath)
	if err == nil {
		metadata["text_length"] = len(text)
		metadata["word_count"] = len(strings.Fields(text))
	}

	p.logger.Info("Metadata extracted successfully")

	return metadata, nil
}
