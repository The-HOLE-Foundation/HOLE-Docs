package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/The-HOLE-Foundation/hole-docs/internal/annotations"
	"github.com/The-HOLE-Foundation/hole-docs/internal/config"
	"github.com/The-HOLE-Foundation/hole-docs/internal/images"
	"github.com/The-HOLE-Foundation/hole-docs/internal/pdf"
	"github.com/The-HOLE-Foundation/hole-docs/internal/storage"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

// Split splits a PDF into pages or ranges
func Split(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: hole-docs split <input.pdf> <output-dir> [page-ranges]")
		fmt.Println("")
		fmt.Println("Split PDF into separate files by page.")
		fmt.Println("")
		fmt.Println("Page ranges (optional):")
		fmt.Println("  1-5       Pages 1 through 5")
		fmt.Println("  1,3,5     Specific pages")
		fmt.Println("  1-5,10    Mixed ranges")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  # Split at pages 10 and 20")
		fmt.Println("  hole-docs split document.pdf ./pages/ 10,20")
		fmt.Println("")
		fmt.Println("  # Extract pages 1-5 and page 10")
		fmt.Println("  hole-docs split document.pdf ./output/ 1-5,10")
		os.Exit(0)
	}

	inputFile := args[0]
	outputDir := args[1]
	pageRanges := []string{}
	if len(args) > 2 {
		pageRanges = strings.Split(args[2], ",")
	}

	if _, err := os.Stat(inputFile); err != nil {
		fmt.Printf("Error: File not found: %s\n", inputFile)
		os.Exit(1)
	}

	_ = godotenv.Load()
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg := config.FromEnv()
	cache := storage.NewCache(cfg.TempDir)
	processor := pdf.NewProcessor(cache, logger)

	fmt.Printf("Splitting PDF: %s\n", filepath.Base(inputFile))
	fmt.Printf("Output directory: %s\n", outputDir)

	err := processor.SplitPDF(inputFile, outputDir, pageRanges)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ PDF split successfully\n")
	fmt.Printf("   Output: %s/\n", outputDir)
}

// Optimize reduces PDF file size
func Optimize(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: hole-docs optimize [options] <input.pdf> [output.pdf]")
		fmt.Println("")
		fmt.Println("Reduce PDF file size using intelligent compression.")
		fmt.Println("")
		fmt.Println("Options:")
		fmt.Println("  -dpi <number>        DPI for images (default: 172)")
		fmt.Println("  -quality <1-100>     JPEG quality (default: 78)")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  # Optimize with defaults (output: input-optimized.pdf)")
		fmt.Println("  hole-docs optimize large-file.pdf")
		fmt.Println("")
		fmt.Println("  # Specify output file")
		fmt.Println("  hole-docs optimize input.pdf output.pdf")
		fmt.Println("")
		fmt.Println("  # High quality (larger file)")
		fmt.Println("  hole-docs optimize -dpi 300 -quality 90 input.pdf")
		fmt.Println("")
		fmt.Println("Notes:")
		fmt.Println("  • Free alternative to Adobe's 'Optimize PDF' feature")
		fmt.Println("  • Typical reduction: 40-70% file size")
		fmt.Println("  • Uses Ghostscript for intelligent compression")
		os.Exit(0)
	}

	inputFile := args[0]
	outputFile := ""
	dpi := 172
	quality := 78

	// Parse simple flags
	for i := 1; i < len(args); i++ {
		if args[i] == "-dpi" && i+1 < len(args) {
			fmt.Sscanf(args[i+1], "%d", &dpi)
			i++
		} else if args[i] == "-quality" && i+1 < len(args) {
			fmt.Sscanf(args[i+1], "%d", &quality)
			i++
		} else {
			outputFile = args[i]
		}
	}

	// Default output filename if not specified
	if outputFile == "" {
		base := filepath.Base(inputFile)
		ext := filepath.Ext(base)
		nameWithoutExt := strings.TrimSuffix(base, ext)
		outputFile = filepath.Join(filepath.Dir(inputFile), nameWithoutExt+"-optimized.pdf")
	}

	// Verify input exists
	originalInfo, err := os.Stat(inputFile)
	if err != nil {
		fmt.Printf("Error: Input file not found: %s\n", inputFile)
		os.Exit(1)
	}

	_ = godotenv.Load()
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg := config.FromEnv()
	cache := storage.NewCache(cfg.TempDir)
	processor := pdf.NewProcessor(cache, logger)

	originalSize := originalInfo.Size()
	fmt.Printf("📄 Optimizing PDF...\n")
	fmt.Printf("  Input:   %s (%.2f MB)\n", filepath.Base(inputFile), float64(originalSize)/(1024*1024))
	fmt.Printf("  DPI:     %d\n", dpi)
	fmt.Printf("  Quality: %d/100\n\n", quality)

	err = processor.OptimizePDFAdvanced(inputFile, outputFile, dpi, quality)
	if err != nil {
		fmt.Printf("Error: Optimization failed: %v\n", err)
		os.Exit(1)
	}

	// Show results
	optimizedInfo, _ := os.Stat(outputFile)
	optimizedSize := optimizedInfo.Size()
	reduction := 0.0
	if originalSize > 0 {
		reduction = (1.0 - float64(optimizedSize)/float64(originalSize)) * 100
	}

	fmt.Printf("✅ Success!\n")
	fmt.Printf("  Original: %.2f MB\n", float64(originalSize)/(1024*1024))
	fmt.Printf("  Optimized: %.2f MB\n", float64(optimizedSize)/(1024*1024))
	fmt.Printf("  Reduction: %.1f%%\n", reduction)
	fmt.Printf("  Output: %s\n", outputFile)
}

// Validate checks PDF integrity
func Validate(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: hole-docs validate <file.pdf>")
		fmt.Println("")
		fmt.Println("Validate PDF file integrity using qpdf.")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  hole-docs validate document.pdf")
		fmt.Println("  hole-docs validate disclosure-response.pdf")
		os.Exit(1)
	}

	filePath := args[0]

	// Check file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("Error: File does not exist: %s\n", filePath)
		os.Exit(1)
	}

	_ = godotenv.Load()
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg := config.FromEnv()
	cache := storage.NewCache(cfg.TempDir)
	processor := pdf.NewProcessor(cache, logger)

	fmt.Printf("Validating PDF: %s\n", filePath)

	valid, err := processor.ValidatePDFWithQPDF(filePath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if valid {
		fmt.Println("✅ PDF is valid")
	} else {
		fmt.Println("❌ PDF is invalid or corrupted")
		os.Exit(1)
	}
}

// Info shows PDF information
func Info(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: hole-docs info <file.pdf>")
		fmt.Println("")
		fmt.Println("Display detailed PDF information.")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  hole-docs info document.pdf")
		os.Exit(1)
	}

	filePath := args[0]

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("Error: File does not exist: %s\n", filePath)
		os.Exit(1)
	}

	_ = godotenv.Load()
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg := config.FromEnv()
	cache := storage.NewCache(cfg.TempDir)
	processor := pdf.NewProcessor(cache, logger)

	fmt.Printf("Getting info for: %s\n\n", filePath)

	info, err := processor.GetPDFInfoWithQPDF(filePath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("File Path: %s\n", info.FilePath)
	fmt.Printf("Pages: %d\n", info.PageCount)
	fmt.Printf("Valid: %v\n", info.Valid)
	fmt.Printf("Encrypted: %v\n", info.Encrypted)
	fmt.Printf("Linearized: %v\n", info.LinearMaps)
	if info.Error != "" {
		fmt.Printf("Warnings: %s\n", info.Error)
	}
}

// ImagesToPDF converts images to PDF
func ImagesToPDF(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: hole-docs images-to-pdf <output.pdf> <image1> [image2] [image3] ...")
		fmt.Println("")
		fmt.Println("Convert image files to archival-grade PDF/A-2b compliant PDF.")
		fmt.Println("")
		fmt.Println("Perfect for:")
		fmt.Println("  • Converting phone photos of documents")
		fmt.Println("  • Creating archival PDFs from scans")
		fmt.Println("  • Combining evidence photos into one PDF")
		fmt.Println("")
		fmt.Println("Supported formats: JPG, PNG, TIFF, WebP")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  # Convert photos to PDF")
		fmt.Println("  hole-docs images-to-pdf evidence.pdf photo1.jpg photo2.jpg")
		fmt.Println("")
		fmt.Println("  # Create archival PDF from scans")
		fmt.Println("  hole-docs images-to-pdf archive.pdf scan*.tiff")
		os.Exit(0)
	}

	outputFile := args[0]
	imagePaths := args[1:]

	// Verify images exist
	for _, img := range imagePaths {
		if _, err := os.Stat(img); err != nil {
			fmt.Printf("Error: Image not found: %s\n", img)
			os.Exit(1)
		}
	}

	_ = godotenv.Load()
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg := config.FromEnv()
	cache := storage.NewCache(cfg.TempDir)

	fmt.Printf("Converting %d image(s) to PDF...\n", len(imagePaths))
	fmt.Printf("Output: %s\n", outputFile)

	// Use the images processor to convert
	imgProc := images.NewProcessor(cache, logger)
	err := imgProc.ConvertToPDF(imagePaths, outputFile)
	if err != nil {
		fmt.Printf("Error: Conversion failed: %v\n", err)
		os.Exit(1)
	}

	info, _ := os.Stat(outputFile)
	fmt.Printf("✅ Success!\n")
	fmt.Printf("   Created: %s (%.2f MB)\n", filepath.Base(outputFile), float64(info.Size())/(1024*1024))
}

// Annotations extracts PDF annotations
func Annotations(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: hole-docs annotations <file.pdf>")
		fmt.Println("")
		fmt.Println("Extract reviewer annotations (highlights, comments, notes) from PDF.")
		fmt.Println("")
		fmt.Println("Perfect for:")
		fmt.Println("  • Extracting reviewer feedback")
		fmt.Println("  • Analyzing redacted documents")
		fmt.Println("  • Pulling highlighted sections")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  hole-docs annotations reviewed-draft.pdf")
		fmt.Println("  hole-docs annotations disclosure.pdf > annotations.json")
		os.Exit(0)
	}

	inputFile := args[0]
	if _, err := os.Stat(inputFile); err != nil {
		fmt.Printf("Error: File not found: %s\n", inputFile)
		os.Exit(1)
	}

	_ = godotenv.Load()
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Use annotations extractor
	extractor := annotations.NewPDFExtractor(logger)
	opts := annotations.DefaultOptions()

	annotSet, err := extractor.ExtractFromFile(inputFile, opts)
	if err != nil {
		fmt.Printf("Error: Extraction failed: %v\n", err)
		os.Exit(1)
	}

	// Format and output
	output := annotations.FormatForLLM(annotSet)
	fmt.Println(output)
}

// TableOfContents displays or adds table of contents
func TableOfContents(args []string) {
	// TODO: Implement toc command
	// Uses internal/pdf processor
	fmt.Println("Table of Contents command - coming soon")
	fmt.Println("Run: hole-docs help")
	os.Exit(1)
}

// Upload uploads and processes a document with AI
func Upload(args []string) {
	fmt.Println("⚠️  Upload command requires AI API keys and database configuration")
	fmt.Println("")
	fmt.Println("This command uses:")
	fmt.Println("  • Mistral AI for OCR and classification")
	fmt.Println("  • Voyage AI for vector embeddings")
	fmt.Println("  • Neon PostgreSQL for storage")
	fmt.Println("  • Cloudflare R2 for file storage")
	fmt.Println("")
	fmt.Println("To enable, set environment variables:")
	fmt.Println("  DATABASE_URL=postgresql://...")
	fmt.Println("  MISTRAL_API_KEY=your-key")
	fmt.Println("  VOYAGEAI_API_KEY=your-key")
	fmt.Println("  CLOUDFLARE_R2_* variables")
	fmt.Println("")
	fmt.Println("For basic PDF operations without AI, use:")
	fmt.Println("  hole-docs merge, render, optimize, split, etc.")
	fmt.Println("")
	fmt.Println("Full implementation available in HOLE-godocs repository.")
	os.Exit(1)
}

// Search searches documents
func Search(args []string) {
	fmt.Println("⚠️  Search command requires database configuration")
	fmt.Println("")
	fmt.Println("This command searches documents you've uploaded.")
	fmt.Println("")
	fmt.Println("To enable, set:")
	fmt.Println("  DATABASE_URL=postgresql://...")
	fmt.Println("")
	fmt.Println("Then use 'hole-docs upload' to add documents first.")
	fmt.Println("")
	fmt.Println("For basic PDF search, use system tools like:")
	fmt.Println("  pdfgrep \"search term\" file.pdf")
	os.Exit(1)
}

// ListDocs lists all documents
func ListDocs(args []string) {
	fmt.Println("⚠️  List-docs command requires database configuration")
	fmt.Println("")
	fmt.Println("This command lists documents you've uploaded.")
	fmt.Println("")
	fmt.Println("To enable, set:")
	fmt.Println("  DATABASE_URL=postgresql://...")
	fmt.Println("")
	fmt.Println("Then use 'hole-docs upload' to add documents first.")
	os.Exit(1)
}

// Config manages configuration
func Config(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: hole-docs config <subcommand>")
		fmt.Println("")
		fmt.Println("Subcommands:")
		fmt.Println("  show                  Show current configuration")
		fmt.Println("  set-output-dir <path> Set default output directory")
		fmt.Println("  init                  Interactive configuration")
		os.Exit(1)
	}

	subcommand := args[0]

	switch subcommand {
	case "show":
		configShow()
	case "set-output-dir":
		if len(args) < 2 {
			fmt.Println("Error: Path required")
			fmt.Println("Usage: hole-docs config set-output-dir <path>")
			os.Exit(1)
		}
		configSetOutputDir(args[1])
	case "init":
		configInit()
	default:
		fmt.Printf("Unknown config subcommand: %s\n", subcommand)
		os.Exit(1)
	}
}

func configShow() {
	// TODO: Implement config show
	fmt.Println("Configuration - coming soon")
}

func configSetOutputDir(path string) {
	// TODO: Implement config set-output-dir
	fmt.Printf("Set output directory to: %s - coming soon\n", path)
}

func configInit() {
	// TODO: Implement config init
	fmt.Println("Interactive configuration - coming soon")
}
