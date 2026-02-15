package pdf

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

// QPDFMergeOptions contains options for qpdf merge operations
type QPDFMergeOptions struct {
	// If true, will detect and remove duplicate images during merge
	RemoveDuplicates bool
	// If true, will encrypt the output PDF
	Encrypt bool
	// If true, will linearize the output PDF for faster web viewing
	Linearize bool
	// If true, will preserve the original encryption from input files
	PreserveEncryption bool
	// Number of copies to retain in case of duplicates (0 = remove all duplicates)
	DuplicateCopies int
}

// MergePDFsWithQPDF merges multiple PDF files using qpdf
// qpdf is preferred for simple, fast concatenation without re-encoding
func (p *Processor) MergePDFsWithQPDF(inputPaths []string, outputPath string, opts *QPDFMergeOptions) error {
	p.logger.Info("Merging PDFs with qpdf",
		zap.Int("count", len(inputPaths)),
		zap.String("output", outputPath))

	// Validate inputs
	if len(inputPaths) < 2 {
		return fmt.Errorf("at least 2 PDF files required for merging, got %d", len(inputPaths))
	}

	// Verify qpdf is available
	if _, err := exec.LookPath("qpdf"); err != nil {
		p.logger.Error("qpdf not found in PATH", zap.Error(err))
		return fmt.Errorf("qpdf is not installed or not in PATH: %w", err)
	}

	// Verify all input files exist
	for _, path := range inputPaths {
		if _, err := os.Stat(path); err != nil {
			p.logger.Error("Input file not found", zap.String("path", path), zap.Error(err))
			return fmt.Errorf("input file not found: %s", path)
		}
	}

	// Set default options if not provided
	if opts == nil {
		opts = &QPDFMergeOptions{}
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		p.logger.Error("Failed to create output directory",
			zap.String("dir", outputDir),
			zap.Error(err))
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Build qpdf command
	args := []string{"--empty"}

	// Add linearize flag if requested
	if opts.Linearize {
		args = append(args, "--linearize")
	}

	// Add pages argument and all input files
	args = append(args, "--pages")
	args = append(args, inputPaths...)
	args = append(args, "--")

	// Add output file
	args = append(args, outputPath)

	// Execute qpdf
	cmd := exec.Command("qpdf", args...)
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		p.logger.Error("qpdf merge failed", zap.Error(err))
		return fmt.Errorf("qpdf merge failed: %w", err)
	}

	// If duplicate removal is requested, use ghostscript optimization
	if opts.RemoveDuplicates {
		p.logger.Info("Applying Ghostscript optimization to remove duplicates")
		tempPath := outputPath + ".gs"
		if err := p.optimizeWithGhostscript(outputPath, tempPath); err != nil {
			p.logger.Warn("Ghostscript optimization failed, keeping qpdf output", zap.Error(err))
		} else {
			// Replace output with optimized version
			if err := os.Rename(tempPath, outputPath); err != nil {
				p.logger.Error("Failed to replace output with optimized version", zap.Error(err))
				return fmt.Errorf("failed to replace output: %w", err)
			}
		}
	}

	// Log success with file size
	fileInfo, _ := os.Stat(outputPath)
	p.logger.Info("PDFs merged successfully with qpdf",
		zap.String("output", outputPath),
		zap.Int64("size", fileInfo.Size()))

	return nil
}

// optimizeWithGhostscript applies Ghostscript optimization to a PDF
func (p *Processor) optimizeWithGhostscript(inputPath string, outputPath string) error {
	p.logger.Info("Optimizing with Ghostscript",
		zap.String("input", inputPath),
		zap.String("output", outputPath))

	// Verify ghostscript is available
	if _, err := exec.LookPath("gs"); err != nil {
		return fmt.Errorf("ghostscript (gs) is not installed or not in PATH: %w", err)
	}

	// Build ghostscript command with optimization settings
	args := []string{
		"-sDEVICE=pdfwrite",
		"-dCompatibilityLevel=1.4",
		"-dNOPAUSE",
		"-dBATCH",
		"-dDetectDuplicateImages",
		"-dCompressFonts=true",
		"-dSubsetFonts=true",
		"-r172x172",
		"-dDownsampleColorImages=true",
		"-dDownsampleGrayImages=true",
		"-dDownsampleMonoImages=true",
		"-dColorImageResolution=172",
		"-dGrayImageResolution=172",
		"-dMonoImageResolution=172",
		"-dColorImageDownsampleType=/Bicubic",
		"-dGrayImageDownsampleType=/Bicubic",
		"-dMonoImageDownsampleType=/Subsample",
		"-dEncodeColorImages=true",
		"-dEncodeGrayImages=true",
		"-dEncodeMonoImages=true",
		"-dColorImageFilter=/FlateEncode",
		"-dGrayImageFilter=/FlateEncode",
		"-dMonoImageFilter=/CCITTFaxEncode",
		"-dPreserveHalftoneInfo=false",
		"-sOutputFile=" + outputPath,
		inputPath,
	}

	cmd := exec.Command("gs", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ghostscript optimization failed: %w", err)
	}

	return nil
}

// MergePDFDirectories merges all PDFs in a directory
func (p *Processor) MergePDFDirectories(inputDir string, outputPath string, recursive bool, opts *QPDFMergeOptions) error {
	p.logger.Info("Merging PDFs from directory",
		zap.String("directory", inputDir),
		zap.Bool("recursive", recursive))

	// Verify input directory exists
	if _, err := os.Stat(inputDir); err != nil {
		p.logger.Error("Input directory not found", zap.String("dir", inputDir), zap.Error(err))
		return fmt.Errorf("input directory not found: %s", inputDir)
	}

	// Find all PDF files
	var pdfFiles []string
	if recursive {
		filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.ToLower(filepath.Ext(path)) == ".pdf" {
				pdfFiles = append(pdfFiles, path)
			}
			return nil
		})
	} else {
		files, err := os.ReadDir(inputDir)
		if err != nil {
			p.logger.Error("Failed to read directory", zap.String("dir", inputDir), zap.Error(err))
			return fmt.Errorf("failed to read directory: %w", err)
		}
		for _, file := range files {
			if !file.IsDir() && strings.ToLower(filepath.Ext(file.Name())) == ".pdf" {
				pdfFiles = append(pdfFiles, filepath.Join(inputDir, file.Name()))
			}
		}
	}

	if len(pdfFiles) == 0 {
		return fmt.Errorf("no PDF files found in directory: %s", inputDir)
	}

	// Sort files for consistent ordering
	// (Note: Go's filepath.Walk already returns sorted results on most systems)

	p.logger.Info("Found PDF files",
		zap.Int("count", len(pdfFiles)),
		zap.String("directory", inputDir))

	// Merge the files
	return p.MergePDFsWithQPDF(pdfFiles, outputPath, opts)
}

// ValidatePDFWithQPDF checks if a PDF file is valid and can be processed
func (p *Processor) ValidatePDFWithQPDF(filePath string) (bool, error) {
	p.logger.Info("Validating PDF with qpdf", zap.String("file", filePath))

	// Verify file exists
	if _, err := os.Stat(filePath); err != nil {
		p.logger.Error("File not found", zap.String("path", filePath), zap.Error(err))
		return false, fmt.Errorf("file not found: %s", filePath)
	}

	// Verify qpdf is available
	if _, err := exec.LookPath("qpdf"); err != nil {
		return false, fmt.Errorf("qpdf is not installed: %w", err)
	}

	// Run qpdf check
	cmd := exec.Command("qpdf", "--check", filePath)
	if err := cmd.Run(); err != nil {
		p.logger.Warn("PDF validation failed", zap.String("file", filePath), zap.Error(err))
		return false, nil
	}

	p.logger.Info("PDF validated successfully", zap.String("file", filePath))
	return true, nil
}

// GetPDFInfoWithQPDF retrieves information about a PDF file
type PDFInfo struct {
	FilePath   string
	PageCount  int
	Valid      bool
	Encrypted  bool
	LinearMaps bool
	Error      string
}

func (p *Processor) GetPDFInfoWithQPDF(filePath string) (*PDFInfo, error) {
	p.logger.Info("Getting PDF info with qpdf", zap.String("file", filePath))

	// Verify file exists
	if _, err := os.Stat(filePath); err != nil {
		p.logger.Error("File not found", zap.String("path", filePath), zap.Error(err))
		return nil, fmt.Errorf("file not found: %s", filePath)
	}

	info := &PDFInfo{
		FilePath: filePath,
	}

	// Verify qpdf is available
	if _, err := exec.LookPath("qpdf"); err != nil {
		return nil, fmt.Errorf("qpdf is not installed: %w", err)
	}

	// Get JSON info from qpdf
	cmd := exec.Command("qpdf", "--json", filePath)
	_, err := cmd.Output()
	if err != nil {
		info.Error = err.Error()
		return info, nil
	}

	// Parse JSON output to get page count
	// qpdf --json provides comprehensive info including pages object
	// For simplicity, we'll try to extract page count from qpdf --check output
	cmd = exec.Command("qpdf", "--check", filePath)
	checkOutput, _ := cmd.Output()

	// Look for page count in output
	outputStr := string(checkOutput)
	if strings.Contains(outputStr, "pages") {
		// Extract page count (typically shows as "File contains X pages")
		parts := strings.Split(outputStr, "pages")
		if len(parts) > 0 {
			// Simple heuristic: look for number before "pages"
			fields := strings.Fields(parts[0])
			if len(fields) > 0 {
				if count, err := strconv.Atoi(fields[len(fields)-1]); err == nil {
					info.PageCount = count
				}
			}
		}
	}

	info.Valid = err == nil
	return info, nil
}
