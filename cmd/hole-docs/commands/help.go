package commands

import "fmt"

// ShowHelp displays the main help message
func ShowHelp(version string) {
	fmt.Printf(`
╔══════════════════════════════════════════════════════════════════════════════╗
║                    hole-docs - Free PDF Tools for Everyone                   ║
║                                                                               ║
║          Making public records accessible without expensive software         ║
╚══════════════════════════════════════════════════════════════════════════════╝

Version: %s

MISSION

  Access to public records shouldn't require expensive software.

  hole-docs provides professional PDF tools for transparency workflows -
  completely free and open source. Built by The HOLE Foundation.

USAGE

  hole-docs <command> [options] [arguments]

COMMON COMMANDS

  PDF Operations:
    merge <output> <inputs...>       Merge multiple PDFs into one
    split <input> <output-dir>       Split PDF into pages or ranges
    optimize <input> [output]        Reduce PDF file size
    render <input> [output-dir]      Convert PDF pages to images

  Analysis:
    annotations <input>              Extract highlights and comments
    toc <input>                      Display or add table of contents
    validate <input>                 Validate PDF integrity
    info <input>                     Show PDF information

  Conversion:
    images-to-pdf <output> <imgs...> Convert images to archival PDF

  Document Processing (AI):
    upload <file>                    Upload and process with OCR
    search <query>                   Search your documents
    list-docs                        List all documents

  Server Mode:
    serve [options]                  Start MCP server for AI assistants
      --http                         HTTP mode (default, port 8080)
      --stdio                        Stdio mode (for Claude Desktop)
      --port <port>                  Custom port

  Configuration:
    config show                      Show current configuration
    config set-output-dir <path>     Set default output directory
    config init                      Interactive configuration

QUICK START EXAMPLES

  # Merge FOIA responses from multiple agencies
  hole-docs merge complete-response.pdf agency1.pdf agency2.pdf agency3.pdf

  # Render disclosure to images for analysis
  hole-docs render police-report.pdf ./evidence-photos/

  # Extract reviewer comments and highlights
  hole-docs annotations reviewed-draft.pdf

  # Optimize large file for emailing
  hole-docs optimize 50mb-file.pdf small-file.pdf

  # Convert phone photos to archival PDF
  hole-docs images-to-pdf evidence.pdf photo1.jpg photo2.jpg photo3.jpg

  # Start MCP server for AI assistant integration
  hole-docs serve --http --port 8080

TRANSPARENCY WORKFLOWS

  hole-docs is designed for people who work with public records:

  • FOIA requesters          • Journalists
  • Transparency advocates   • Researchers
  • Legal professionals      • Government accountability activists

  No subscriptions. No tracking. No lock-in.
  Just free, powerful tools for everyone.

GETTING HELP

  hole-docs <command> --help       Show help for specific command
  hole-docs help                   Show this message
  hole-docs version                Show version

DOCUMENTATION

  Installation:  https://github.com/The-HOLE-Foundation/hole-docs/blob/main/INSTALL.md
  User Guide:    https://github.com/The-HOLE-Foundation/hole-docs/blob/main/docs/usage/
  GitHub:        https://github.com/The-HOLE-Foundation/hole-docs
  Website:       https://theholetruth.org

REQUIREMENTS

  • qpdf, mupdf, ghostscript (install via package manager)
  • Optional: AI API keys for document processing features

  macOS:    brew install qpdf mupdf ghostscript
  Ubuntu:   sudo apt install qpdf mupdf-tools ghostscript
  Fedora:   sudo dnf install qpdf mupdf ghostscript

Built with ❤️  by The HOLE Foundation
Making transparency accessible to everyone, one document at a time.

`, version)
}
