# hole-docs Architecture

**Unified CLI with MCP server capabilities**

---

## Design Philosophy

**ONE binary, multiple modes of operation:**

1. **CLI mode** (default) - Direct human use
2. **MCP server mode** - AI assistant integration
3. **Batch mode** - Process multiple files

**Simple, focused, accessible.**

---

## Command Structure

### Unified CLI Design

```
hole-docs
├── merge <output> <inputs...>          # Merge PDFs
├── split <input> <output-dir>          # Split PDF
├── render <input> [output-dir]         # Render to images
├── optimize <input> [output]           # Reduce file size
├── annotations <input>                 # Extract annotations
├── toc <input>                         # Add table of contents
├── images-to-pdf <output> <images...>  # Convert images to PDF
├── upload <file>                       # Upload & process with AI
├── search <query>                      # Search documents
├── serve                               # Start MCP server
│   ├── --http                          # HTTP mode (default port 8080)
│   ├── --stdio                         # Stdio mode for MCP clients
│   └── --port <port>                   # Custom port
├── config                              # Configuration
│   ├── show                            # Show current config
│   ├── set-output-dir <path>          # Set default output
│   └── init                            # Interactive setup
└── help                                # Show help
```

### Consolidated from 3 Binaries

**Previously** (HOLE-godocs):
- ❌ `godocs` (CLI) - 10 commands
- ❌ `godocs-server` (MCP server) - HTTP only
- ❌ `godocs-pdf-merge` (PDF utils) - 5 commands

**Now** (hole-docs):
- ✅ `hole-docs` (unified) - ALL commands + serve mode

---

## Architecture Layers

```
┌─────────────────────────────────────────┐
│         hole-docs Binary                │
├─────────────────────────────────────────┤
│  CLI Commands        MCP Server         │
│  ├── merge           ├── HTTP mode      │
│  ├── split           └── Stdio mode     │
│  ├── render                             │
│  ├── optimize                           │
│  └── ...                                │
├─────────────────────────────────────────┤
│         Core Libraries                  │
│  ├── PDF processor (qpdf, pdfcpu)      │
│  ├── Image processor (MuPDF, GS)       │
│  ├── AI pipeline (Mistral, Voyage)     │
│  ├── Storage (R2, filesystem)          │
│  └── Database (Neon PostgreSQL)        │
├─────────────────────────────────────────┤
│       External Dependencies             │
│  ├── qpdf (PDF manipulation)           │
│  ├── MuPDF (rendering)                 │
│  ├── Ghostscript (optimization)        │
│  └── Optional: Mistral AI, Voyage AI   │
└─────────────────────────────────────────┘
```

---

## MCP Server Integration

### Two Modes

**1. HTTP Mode** (for network access):
```bash
hole-docs serve --http --port 8080
```

Returns MCP protocol responses over HTTP.

**2. Stdio Mode** (for Claude Desktop, other MCP clients):
```bash
hole-docs serve --stdio
```

Communicates via stdin/stdout using MCP protocol.

### Available MCP Tools

When running in `serve` mode, hole-docs exposes 18+ MCP tools:

**PDF Tools**:
- `merge_pdfs` - Merge multiple PDFs
- `split_pdf` - Split PDF by pages
- `optimize_pdf` - Reduce file size
- `render_pdf_pages` - Render to images
- `validate_pdf` - Check PDF integrity
- `add_toc_pdf` - Add table of contents
- `extract_annotations` - Pull reviewer comments

**Image Tools**:
- `images_to_pdf` - Convert images to PDF
- `merge_images` - Combine images
- `convert_image` - Change formats

**Word Tools**:
- `convert_docx_to_pdf` - DOCX to PDF
- `extract_text_from_docx` - Pull text
- `get_docx_metadata` - Document info

**Document Processing**:
- `ingest_document` - Full AI pipeline (OCR, classify, extract)
- `search_documents` - Semantic search

**See**: `docs/api/MCP_TOOLS.md` for complete reference

---

## Technology Stack

### Core Dependencies

**Language**: Go 1.21+
- Fast compilation
- Static binaries (no dependencies)
- Cross-platform
- Excellent concurrency

**PDF Processing**:
- `qpdf` - Fast, lossless merging
- `pdfcpu` - Pure Go PDF manipulation
- `MuPDF` - High-quality rendering
- `Ghostscript` - Optimization

**AI Services** (optional):
- Mistral AI - OCR and classification
- Voyage AI - Vector embeddings

**Storage**:
- Cloudflare R2 - Object storage
- Neon PostgreSQL - Metadata and search

### External Tools Required

**Minimal requirements**:
```bash
# macOS
brew install qpdf mupdf ghostscript

# Ubuntu/Debian
sudo apt install qpdf mupdf-tools ghostscript

# Fedora
sudo dnf install qpdf mupdf ghostscript
```

**Optional** (for AI features):
- API keys: Mistral, Voyage
- Database: Neon PostgreSQL
- Storage: Cloudflare R2 account

---

## Configuration

### Environment Variables

**Optional** (AI features disabled if not set):
```bash
# AI Services
MISTRAL_API_KEY="your-key"       # Enables OCR and classification
VOYAGEAI_API_KEY="your-key"      # Enables vector embeddings

# Storage
CLOUDFLARE_ACCOUNT_ID="id"
CLOUDFLARE_R2_BUCKET_NAME="bucket"
CLOUDFLARE_R2_ACCESS_KEY_ID="key"
CLOUDFLARE_R2_SECRET_ACCESS_KEY="secret"

# Database
DATABASE_URL="postgresql://..."

# Server (for serve mode only)
PORT=8080
```

### Config File

User preferences stored in `~/.hole-docs/config.json`:
```json
{
  "default_output_dir": "~/Documents/hole-docs-output",
  "default_dpi": 300,
  "default_format": "png",
  "default_processor": "mupdf"
}
```

**Set via CLI**:
```bash
hole-docs config set-output-dir ~/Desktop/pdfs
hole-docs config show
```

---

## Migration from HOLE-godocs

### What's Changing

**Binary names**:
- `godocs` → `hole-docs`
- `godocs-server` → `hole-docs serve`
- `godocs-pdf-merge` → `hole-docs merge/split/...`

**Commands consolidated**:
- All PDF operations now under main binary
- MCP server now a subcommand (`serve`)
- Simpler mental model

**What stays the same**:
- All functionality preserved
- Same underlying libraries
- Same configuration
- Same data formats
- Same quality

---

## Project Structure

```
hole-docs/
├── cmd/
│   └── hole-docs/              # Single unified binary
│       └── main.go             # CLI entrypoint
├── internal/
│   ├── cli/                    # CLI command handlers
│   ├── mcp/                    # MCP server implementation
│   ├── pdf/                    # PDF processing
│   ├── images/                 # Image processing
│   ├── pipeline/               # Document pipeline
│   ├── config/                 # Configuration
│   └── ...
├── docs/
│   ├── INDEX.md                # Documentation hub
│   ├── installation/           # Install guides
│   ├── usage/                  # Usage examples
│   └── api/                    # MCP API reference
├── README.md                   # Project overview
├── INSTALL.md                  # Installation guide
├── LICENSE                     # MIT license
└── Makefile                    # Build automation
```

---

## Build & Release

### Local Development

```bash
# Build
make build

# Test
make test

# Run
./bin/hole-docs help
```

### Cross-platform Releases

```bash
# Build all platforms
make release

# Outputs:
# dist/hole-docs-v0.3.0-macos-arm64.tar.gz
# dist/hole-docs-v0.3.0-macos-amd64.tar.gz
# dist/hole-docs-v0.3.0-linux-amd64.tar.gz
# dist/hole-docs-v0.3.0-linux-arm64.tar.gz
```

### Automated Releases

Push a tag, GitHub Actions builds and releases automatically:
```bash
git tag v0.3.0
git push origin v0.3.0
```

---

## Success Metrics

**We've succeeded if:**
- ✅ Anyone can install hole-docs in under 1 minute
- ✅ Common tasks take under 10 seconds
- ✅ Users save money (no Adobe subscriptions)
- ✅ Transparency advocates adopt it in their workflows
- ✅ Contributors join the project
- ✅ Public records become more accessible

---

**Version**: 0.3.0 (planned)
**Status**: In development
**Target**: Production ready Q1 2026
