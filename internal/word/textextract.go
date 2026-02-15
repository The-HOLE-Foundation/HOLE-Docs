package word

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"

	"go.uber.org/zap"
)

// ExtractTextFree extracts text from DOCX using native Go (no external library)
// DOCX files are ZIP archives containing XML - we can parse them directly
func (p *Processor) ExtractTextFree(inputPath string) (string, error) {
	p.logger.Info("Extracting text from Word document (free method)", zap.String("input", inputPath))

	// Verify input file exists
	if _, err := os.Stat(inputPath); err != nil {
		p.logger.Error("Input file not found", zap.String("path", inputPath), zap.Error(err))
		return "", fmt.Errorf("input file not found: %s", inputPath)
	}

	// Open DOCX as ZIP
	reader, err := zip.OpenReader(inputPath)
	if err != nil {
		return "", fmt.Errorf("failed to open DOCX file: %w", err)
	}
	defer reader.Close()

	// Find document.xml (contains main text content)
	var documentXML *zip.File
	for _, file := range reader.File {
		if file.Name == "word/document.xml" {
			documentXML = file
			break
		}
	}

	if documentXML == nil {
		return "", fmt.Errorf("document.xml not found in DOCX file")
	}

	// Open and read document.xml
	rc, err := documentXML.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open document.xml: %w", err)
	}
	defer rc.Close()

	xmlData, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("failed to read document.xml: %w", err)
	}

	// Parse XML to extract text
	text := extractTextFromXML(string(xmlData))

	p.logger.Info("Text extracted successfully", zap.Int("length", len(text)))

	return text, nil
}

// extractTextFromXML extracts text content from Word document XML
func extractTextFromXML(xmlContent string) string {
	var result strings.Builder
	decoder := xml.NewDecoder(strings.NewReader(xmlContent))

	inText := false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		switch element := token.(type) {
		case xml.StartElement:
			// <w:t> tags contain text content
			if element.Name.Local == "t" {
				inText = true
			}
		case xml.EndElement:
			if element.Name.Local == "t" {
				inText = false
			}
			// Add newline after paragraphs
			if element.Name.Local == "p" {
				result.WriteString("\n")
			}
		case xml.CharData:
			if inText {
				result.Write(element)
			}
		}
	}

	return result.String()
}
