package commands

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/The-HOLE-Foundation/hole-docs/internal/config"
	"github.com/The-HOLE-Foundation/hole-docs/internal/pdf"
	"github.com/The-HOLE-Foundation/hole-docs/internal/storage"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

// Merge merges multiple PDF files into one
func Merge(args []string) {
	fs := flag.NewFlagSet("merge", flag.ExitOnError)

	removeDuplicates := fs.Bool("remove-duplicates", false, "Remove duplicate images and fonts")
	linearize := fs.Bool("linearize", false, "Linearize PDF for faster web viewing")

	fs.Usage = func() {
		fmt.Println(`Usage: hole-docs merge [options] <output.pdf> <input1.pdf> [input2.pdf ...]

Merge multiple PDF files into a single document.

Options:
  -remove-duplicates    Remove duplicate images and fonts (reduces file size)
  -linearize           Optimize PDF for fast web viewing
  -recursive           Include PDFs in subdirectories (for merge-dir)

Examples:
  # Merge FOIA responses
  hole-docs merge complete-response.pdf agency1.pdf agency2.pdf agency3.pdf

  # Merge with optimization
  hole-docs merge -remove-duplicates -linearize output.pdf *.pdf

  # Merge all PDFs in a directory
  hole-docs merge output.pdf /path/to/pdfs/*.pdf

Alternative Commands:
  hole-docs merge-dir <output.pdf> <directory>    Merge all PDFs in directory

Notes:
  • Uses qpdf for fast, lossless merging
  • Preserves all PDF features (bookmarks, forms, metadata)
  • Output quality identical to input quality
  • Free alternative to Adobe Acrobat's "Combine Files"
`)
	}

	fs.Parse(args)

	if fs.NArg() < 2 {
		fmt.Println("Error: Requires at least output path and one input PDF")
		fs.Usage()
		os.Exit(1)
	}

	// Load environment
	_ = godotenv.Load()

	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Get output and input paths
	outputPath := fs.Arg(0)
	inputPaths := fs.Args()[1:]

	// Validate input files exist
	for _, path := range inputPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			fmt.Printf("Error: Input file does not exist: %s\n", path)
			os.Exit(1)
		}
	}

	// Create cache
	cfg := config.FromEnv()
	cache := storage.NewCache(cfg.TempDir)

	// Create PDF processor
	processor := pdf.NewProcessor(cache, logger)

	// Merge PDFs
	logger.Info("Merging PDFs",
		zap.Int("input_count", len(inputPaths)),
		zap.String("output", outputPath))

	opts := &pdf.QPDFMergeOptions{
		RemoveDuplicates: *removeDuplicates,
		Linearize:        *linearize,
	}

	err = processor.MergePDFsWithQPDF(inputPaths, outputPath, opts)
	if err != nil {
		logger.Error("Failed to merge PDFs", zap.Error(err))
		fmt.Printf("Error: Failed to merge PDFs: %v\n", err)
		os.Exit(1)
	}

	// Get output file size
	info, err := os.Stat(outputPath)
	if err == nil {
		sizeMB := float64(info.Size()) / (1024 * 1024)
		fmt.Printf("✅ Successfully merged %d PDFs\n", len(inputPaths))
		fmt.Printf("   Output: %s (%.1f MB)\n", outputPath, sizeMB)
	} else {
		fmt.Printf("✅ Successfully merged %d PDFs to %s\n", len(inputPaths), outputPath)
	}

	logger.Info("Merge completed successfully",
		zap.String("output", outputPath))
}

// MergeDir merges all PDFs in a directory
func MergeDir(args []string) {
	fs := flag.NewFlagSet("merge-dir", flag.ExitOnError)

	removeDuplicates := fs.Bool("remove-duplicates", false, "Remove duplicate images and fonts")
	linearize := fs.Bool("linearize", false, "Linearize PDF for faster web viewing")
	recursive := fs.Bool("recursive", false, "Include subdirectories")

	fs.Usage = func() {
		fmt.Println(`Usage: hole-docs merge-dir [options] <output.pdf> <directory>

Merge all PDF files in a directory.

Options:
  -remove-duplicates    Remove duplicate images and fonts
  -linearize           Optimize for web viewing
  -recursive           Include PDFs in subdirectories

Examples:
  # Merge all PDFs in folder
  hole-docs merge-dir combined.pdf /path/to/pdfs/

  # Merge recursively with optimization
  hole-docs merge-dir -recursive -remove-duplicates output.pdf ./documents/
`)
	}

	fs.Parse(args)

	if fs.NArg() != 2 {
		fmt.Println("Error: Requires output path and directory")
		fs.Usage()
		os.Exit(1)
	}

	outputPath := fs.Arg(0)
	dirPath := fs.Arg(1)

	// Check directory exists
	info, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		fmt.Printf("Error: Directory does not exist: %s\n", dirPath)
		os.Exit(1)
	}
	if !info.IsDir() {
		fmt.Printf("Error: Not a directory: %s\n", dirPath)
		os.Exit(1)
	}

	// Load environment
	_ = godotenv.Load()

	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Find all PDFs in directory
	var pdfFiles []string
	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".pdf" {
			pdfFiles = append(pdfFiles, path)
		}
		return nil
	}

	if *recursive {
		err = filepath.Walk(dirPath, walkFn)
	} else {
		// Just top-level directory
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			fmt.Printf("Error reading directory: %v\n", err)
			os.Exit(1)
		}
		for _, entry := range entries {
			if !entry.IsDir() && filepath.Ext(entry.Name()) == ".pdf" {
				pdfFiles = append(pdfFiles, filepath.Join(dirPath, entry.Name()))
			}
		}
	}

	if len(pdfFiles) == 0 {
		fmt.Printf("Error: No PDF files found in %s\n", dirPath)
		os.Exit(1)
	}

	// Create cache and processor
	cfg := config.FromEnv()
	cache := storage.NewCache(cfg.TempDir)
	processor := pdf.NewProcessor(cache, logger)

	// Merge PDFs
	logger.Info("Merging PDFs from directory",
		zap.Int("pdf_count", len(pdfFiles)),
		zap.String("directory", dirPath),
		zap.String("output", outputPath))

	opts := &pdf.QPDFMergeOptions{
		RemoveDuplicates: *removeDuplicates,
		Linearize:        *linearize,
	}

	err = processor.MergePDFsWithQPDF(pdfFiles, outputPath, opts)
	if err != nil {
		logger.Error("Failed to merge PDFs", zap.Error(err))
		fmt.Printf("Error: Failed to merge PDFs: %v\n", err)
		os.Exit(1)
	}

	// Show results
	outInfo, err := os.Stat(outputPath)
	if err == nil {
		sizeMB := float64(outInfo.Size()) / (1024 * 1024)
		fmt.Printf("✅ Successfully merged %d PDFs from %s\n", len(pdfFiles), dirPath)
		fmt.Printf("   Output: %s (%.1f MB)\n", outputPath, sizeMB)
	} else {
		fmt.Printf("✅ Successfully merged %d PDFs to %s\n", len(pdfFiles), outputPath)
	}
}
