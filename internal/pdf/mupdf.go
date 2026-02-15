package pdf

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"go.uber.org/zap"
)

// MuPDFRenderer handles PDF rendering using MuPDF (mutool)
type MuPDFRenderer struct {
	inputPath  string
	outputDir  string
	format     string
	dpi        int
	logger     *zap.Logger
}

// NewMuPDFRenderer creates a new MuPDF renderer
func NewMuPDFRenderer(inputPath string, outputDir string, format string, dpi int, logger *zap.Logger) *MuPDFRenderer {
	return &MuPDFRenderer{
		inputPath:  inputPath,
		outputDir:  outputDir,
		format:     format,
		dpi:        dpi,
		logger:     logger,
	}
}

// CheckMuPDFAvailable checks if mutool is installed and available
func CheckMuPDFAvailable() error {
	_, err := exec.LookPath("mutool")
	if err != nil {
		return fmt.Errorf("mutool not found in PATH. Install MuPDF: brew install mupdf")
	}
	return nil
}

// Render renders the PDF pages to images using mutool draw
// Based on MuPDF documentation:
// mutool draw -o page%03d.png -r 150 document.pdf
func (r *MuPDFRenderer) Render() error {
	r.logger.Info("Rendering PDF with MuPDF",
		zap.String("input", r.inputPath),
		zap.String("output_dir", r.outputDir),
		zap.String("format", r.format),
		zap.Int("dpi", r.dpi))

	// Validate format (MuPDF supports png, pnm, pam, pbm, pkm)
	// For simplicity, we'll support png and ppm (PPM is MuPDF's native format)
	switch r.format {
	case "png":
		// PNG is the default output format
	case "ppm", "pnm":
		// PPM/PNM format
	case "jpeg", "jpg":
		// MuPDF doesn't support JPEG directly, we'll use PNG
		r.logger.Warn("MuPDF doesn't support JPEG output, using PNG instead")
		r.format = "png"
	default:
		return fmt.Errorf("unsupported format for MuPDF: %s (use png or ppm)", r.format)
	}

	// Build output file pattern
	// mutool draw expects: page%03d.png for page001.png, page002.png, etc.
	outputPattern := filepath.Join(r.outputDir, fmt.Sprintf("page%%03d.%s", r.format))

	// Build mutool draw command
	// Format: mutool draw -o <output_pattern> -r <dpi> <input.pdf>
	args := []string{
		"draw",
		"-o", outputPattern,
		"-r", fmt.Sprintf("%d", r.dpi),
	}

	// Add color space flag if needed (default is RGB)
	// For grayscale: -c gray
	// For RGB: -c rgb (default)
	// We'll use default RGB

	// Add input file
	args = append(args, r.inputPath)

	r.logger.Info("Executing mutool command",
		zap.String("command", "mutool"),
		zap.Strings("args", args))

	// Execute mutool draw
	cmd := exec.Command("mutool", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		r.logger.Error("mutool draw failed", zap.Error(err))
		return fmt.Errorf("mutool draw failed: %w", err)
	}

	r.logger.Info("MuPDF rendering completed successfully",
		zap.String("output_pattern", outputPattern))

	return nil
}

// GetMuPDFVersion returns the mutool version
func GetMuPDFVersion() (string, error) {
	cmd := exec.Command("mutool", "-v")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get mutool version: %w", err)
	}
	return string(output), nil
}
