package word

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"go.uber.org/zap"
)

// ConvertToPDFWithLibreOffice converts a Word document to PDF using LibreOffice headless mode
// This is completely free and open source (no licensing required)
// Requires LibreOffice to be installed on the system
func (p *Processor) ConvertToPDFWithLibreOffice(inputPath string, outputPath string) error {
	p.logger.Info("Converting Word document to PDF with LibreOffice",
		zap.String("input", inputPath),
		zap.String("output", outputPath))

	// Verify input file exists
	if _, err := os.Stat(inputPath); err != nil {
		p.logger.Error("Input file not found", zap.String("path", inputPath), zap.Error(err))
		return fmt.Errorf("input file not found: %s", inputPath)
	}

	// Get absolute paths
	absInput, err := filepath.Abs(inputPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute input path: %w", err)
	}

	absOutput, err := filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute output path: %w", err)
	}

	outputDir := filepath.Dir(absOutput)

	// LibreOffice headless conversion command
	// --headless: Run without GUI
	// --convert-to pdf: Convert to PDF format
	// --outdir: Output directory
	cmd := exec.Command("soffice",
		"--headless",
		"--convert-to", "pdf",
		"--outdir", outputDir,
		absInput,
	)

	// Execute LibreOffice
	output, err := cmd.CombinedOutput()
	if err != nil {
		p.logger.Error("LibreOffice conversion failed",
			zap.Error(err),
			zap.String("output", string(output)))
		return fmt.Errorf("LibreOffice conversion failed: %w - %s", err, string(output))
	}

	// LibreOffice creates file with same name but .pdf extension
	// Need to rename if user specified different name
	libreOutputName := filepath.Base(absInput)
	libreOutputName = libreOutputName[:len(libreOutputName)-len(filepath.Ext(libreOutputName))] + ".pdf"
	libreOutputPath := filepath.Join(outputDir, libreOutputName)

	// Rename if needed
	if libreOutputPath != absOutput {
		if err := os.Rename(libreOutputPath, absOutput); err != nil {
			p.logger.Warn("Failed to rename output file",
				zap.String("from", libreOutputPath),
				zap.String("to", absOutput),
				zap.Error(err))
			// File still exists, just with different name
		}
	}

	// Verify output file exists
	if _, err := os.Stat(absOutput); err != nil {
		// Try the LibreOffice default name
		if _, err := os.Stat(libreOutputPath); err != nil {
			return fmt.Errorf("output file not created")
		}
		absOutput = libreOutputPath
	}

	// Log success with file sizes
	originalInfo, _ := os.Stat(absInput)
	pdfInfo, _ := os.Stat(absOutput)

	p.logger.Info("Word document converted to PDF successfully (LibreOffice)",
		zap.String("output", absOutput),
		zap.Int64("original_size", originalInfo.Size()),
		zap.Int64("pdf_size", pdfInfo.Size()))

	return nil
}

// CheckLibreOfficeInstalled checks if LibreOffice is available
func CheckLibreOfficeInstalled() bool {
	cmd := exec.Command("soffice", "--version")
	return cmd.Run() == nil
}
