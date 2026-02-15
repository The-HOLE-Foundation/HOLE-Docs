package commands

import (
	"flag"
	"fmt"
	"os"

	"github.com/The-HOLE-Foundation/hole-docs/internal/config"
	"github.com/The-HOLE-Foundation/hole-docs/internal/pdf"
	"github.com/The-HOLE-Foundation/hole-docs/internal/storage"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

// Render converts PDF pages to images
func Render(args []string) {
	fs := flag.NewFlagSet("render", flag.ExitOnError)

	// Processor options
	useMuPDF := fs.Bool("mu", false, "Use MuPDF (pixel-perfect, faster)")
	useMuPDFAlt := fs.Bool("mupdf", false, "Use MuPDF (alias for -mu)")
	useGhostscript := fs.Bool("gs", false, "Use Ghostscript (default)")

	// Output options
	dpi := fs.Int("dpi", 300, "DPI resolution (72-600)")
	format := fs.String("format", "png", "Output format: png, jpeg, tiff")

	fs.Usage = func() {
		fmt.Println(`Usage: hole-docs render [options] <input.pdf> [output-dir]

Convert PDF pages to high-quality images.

Perfect for:
  • Analyzing FOIA disclosures visually
  • Archiving documents as photographs
  • Creating evidence exhibits
  • Sharing excerpts without full PDF

Processors:
  -mu, -mupdf          Use MuPDF (pixel-perfect, recommended) ⭐
  -gs, -ghostscript    Use Ghostscript (default, robust)

Options:
  -dpi <number>        Resolution in DPI (default: 300)
                       • 150 = screen viewing
                       • 300 = print quality
                       • 600 = archival quality
  -format <type>       Output format: png, jpeg, tiff (default: png)
  -quality <1-100>     JPEG quality (default: 95, only for jpeg)

Examples:
  # Render disclosure to PNG images (default)
  hole-docs render police-report.pdf ./photos/

  # High-quality rendering with MuPDF
  hole-docs render -mu -dpi 600 document.pdf ./archive/

  # JPEG format for smaller files
  hole-docs render -format jpeg -quality 85 large.pdf ./images/

  # Specify output directory
  hole-docs render disclosure.pdf ~/Desktop/foia-photos/

Processor Comparison:

  Ghostscript (default):
    • Robust and well-tested
    • Good for general use
    • Handles edge cases well

  MuPDF (-mu) ⭐ Recommended:
    • Pixel-perfect rendering
    • Faster performance
    • Better for complex PDFs
    • Ideal for evidence/archival

Output:
  Creates one image per page: page-001.png, page-002.png, ...

Notes:
  • Free alternative to Adobe's "Export to Images" ($240/year)
  • Perfect for visual analysis of government documents
  • Images can be uploaded to analysis tools
  • Great for creating photo archives of public records
`)
	}

	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Println("Error: Input PDF file required")
		fs.Usage()
		os.Exit(1)
	}

	inputPath := fs.Arg(0)
	outputDir := ""
	if fs.NArg() >= 2 {
		outputDir = fs.Arg(1)
	}

	// Load environment
	_ = godotenv.Load()

	// Determine processor
	processor := "ghostscript"
	if *useMuPDF || *useMuPDFAlt {
		processor = "mupdf"
	} else if *useGhostscript {
		processor = "ghostscript"
	}

	// Validate DPI
	if *dpi < 72 || *dpi > 600 {
		fmt.Println("Error: DPI must be between 72 and 600")
		os.Exit(1)
	}

	// Validate format
	validFormats := map[string]bool{"png": true, "jpeg": true, "jpg": true, "tiff": true}
	if !validFormats[*format] {
		fmt.Println("Error: Format must be png, jpeg, or tiff")
		os.Exit(1)
	}

	// Get or create output directory
	if outputDir == "" {
		// Use default or prompt user
		outputDir = "./output"
		fmt.Printf("Using default output directory: %s\n", outputDir)
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Initialize PDF processor
	cfg := config.FromEnv()
	cache := storage.NewCache(cfg.TempDir)
	pdfProc := pdf.NewProcessor(cache, logger)

	// Render pages
	fmt.Printf("Rendering %s to %s using %s...\n", inputPath, outputDir, processor)

	err = pdfProc.RenderPagesToImagesWithProcessor(inputPath, outputDir, *format, *dpi, processor)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Successfully rendered PDF to %s/\n", outputDir)
}
