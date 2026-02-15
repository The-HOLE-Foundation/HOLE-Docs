# hole-docs

**Free, fast, and easy PDF tools for transparency workflows**

> Making public records accessible to everyone - no expensive software required.

---

## Mission

**Access to public records shouldn't require expensive software.**

Whether you're filing a FOIA request, analyzing government disclosures, or archiving transparency documents, `hole-docs` provides professional-grade PDF tools - completely free and open source.

We believe that **transparency work belongs to everyone**, not just those who can afford Adobe Acrobat or enterprise document management systems.

---

## What is hole-docs?

A unified command-line tool for working with PDFs in transparency and FOIA workflows:

✅ **Merge PDFs** - Combine disclosure responses, append exhibits, consolidate records
✅ **Split PDFs** - Extract relevant pages, separate documents, organize files
✅ **Render to Images** - Convert PDFs to photographs for analysis, archiving, evidence
✅ **Optimize PDFs** - Reduce file sizes for email, sharing, archiving
✅ **Extract Annotations** - Pull reviewer comments, highlights, and notes
✅ **Table of Contents** - Auto-generate bookmarks and navigation
✅ **Convert Images** - Turn photos and scans into archival PDFs
✅ **Process Documents** - OCR, classify, extract entities with AI

**All in one simple command-line tool. No subscriptions. No license fees. No lock-in.**

---

## Quick Start

### Installation

**macOS (Apple Silicon)**:
```bash
curl -LO https://github.com/The-HOLE-Foundation/hole-docs/releases/latest/download/hole-docs-latest-macos-arm64.tar.gz
tar -xzf hole-docs-latest-macos-arm64.tar.gz
sudo mv hole-docs /usr/local/bin/
hole-docs help
```

**Ubuntu/Linux (AMD64)**:
```bash
curl -LO https://github.com/The-HOLE-Foundation/hole-docs/releases/latest/download/hole-docs-latest-linux-amd64.tar.gz
tar -xzf hole-docs-latest-linux-amd64.tar.gz
sudo mv hole-docs /usr/local/bin/
hole-docs help
```

See [INSTALL.md](INSTALL.md) for all platforms and installation methods.

### Common Tasks

**Merge multiple PDFs** (combine FOIA responses):
```bash
hole-docs merge output.pdf response1.pdf response2.pdf response3.pdf
```

**Render PDF to images** (for analysis or archiving):
```bash
hole-docs render disclosure.pdf ./images/
```

**Extract highlighted text** (pull reviewer comments):
```bash
hole-docs annotations reviewed-document.pdf
```

**Optimize large PDF** (reduce file size for email):
```bash
hole-docs optimize large-disclosure.pdf optimized.pdf
```

**Split PDF** (extract specific pages):
```bash
hole-docs split document.pdf ./pages/ --pages 1-10,25,30-35
```

**Convert images to PDF** (scan to archival format):
```bash
hole-docs images-to-pdf output.pdf photo1.jpg photo2.jpg photo3.jpg
```

---

## Why We Built This

### The Problem

**Public records access is unnecessarily expensive:**
- Adobe Acrobat Pro: $20-30/month ($240-360/year)
- PDF editing tools: $10-50/month
- OCR software: $15-100/month
- Document management: $50-500/month

**For transparency advocates, journalists, researchers, and citizens:**
- These costs add up quickly
- Many can't afford professional tools
- Proprietary formats create lock-in
- Privacy concerns with cloud services

### Our Solution

**hole-docs is:**
- ✅ **Free** - Open source, no subscriptions, no fees
- ✅ **Fast** - Command-line efficiency, batch processing
- ✅ **Private** - Runs locally, no cloud uploads
- ✅ **Powerful** - Professional features rivaling paid tools
- ✅ **Simple** - Clean CLI, clear documentation
- ✅ **Open** - Contribute, modify, share freely

---

## Use Cases

### FOIA Workflows

**Combine responses** from multiple agencies:
```bash
hole-docs merge complete-response.pdf \
  agency1-response.pdf \
  agency2-response.pdf \
  agency3-response.pdf
```

**Extract pages** for specific requests:
```bash
hole-docs split large-disclosure.pdf ./extracted/ --pages 45-67
```

**Optimize before sharing** (reduce 50MB to 5MB):
```bash
hole-docs optimize huge-file.pdf small-file.pdf
```

### Analysis & Archiving

**Render to images** for visual analysis:
```bash
hole-docs render police-report.pdf ./evidence-photos/
```

**Extract annotations** for revision tracking:
```bash
hole-docs annotations reviewed-draft.pdf > reviewer-comments.json
```

**Convert photos** to archival PDFs:
```bash
hole-docs images-to-pdf archive.pdf photo1.jpg photo2.jpg photo3.jpg
```

### Journalism & Research

**Merge source documents**:
```bash
hole-docs merge investigation.pdf interview1.pdf records.pdf timeline.pdf
```

**Create searchable PDFs** from scans:
```bash
hole-docs process scanned-document.pdf  # OCR + classification
```

---

## Features

### PDF Operations

- **Merge** - Combine multiple PDFs (lossless, fast)
- **Split** - Extract pages or page ranges
- **Optimize** - Reduce file size intelligently
- **Render** - Convert pages to high-quality images
- **Validate** - Check PDF integrity
- **Linearize** - Optimize for web streaming

### Image Operations

- **Convert** - PNG, JPEG, WebP, TIFF formats
- **To PDF** - Create archival PDFs from images
- **Merge** - Combine images into single PDF
- **Extract** - Pull images from PDFs

### Document Processing (AI-powered)

- **OCR** - Extract text from scanned documents
- **Classify** - Auto-detect document types
- **Extract Entities** - Find people, dates, locations, charges
- **Search** - Full-text semantic search
- **Vectorize** - Create embeddings for similarity search

### Advanced Features

- **Annotations** - Extract highlights, comments, notes
- **Table of Contents** - Auto-generate PDF bookmarks
- **PDF/A** - Create archival-compliant PDFs
- **Batch Processing** - Handle hundreds of files
- **Metadata** - Read/edit document properties

---

## Technology

**Built with:**
- Go 1.21+ (fast, reliable, static binaries)
- qpdf (lossless PDF manipulation)
- MuPDF (high-quality rendering)
- Ghostscript (optimization and conversion)
- Mistral AI (OCR and classification) - optional
- Voyage AI (embeddings) - optional

**Runs on:**
- macOS (Apple Silicon + Intel)
- Linux (Ubuntu, Debian, Fedora, etc.)
- Raspberry Pi (ARM64)

**No dependencies** - Static binaries work anywhere.

---

## Privacy & Philosophy

### Your Data Stays Yours

- ✅ **Runs locally** - No cloud uploads required
- ✅ **Optional AI** - Use AI features only if you want
- ✅ **Your choice** - Self-host or use your own API keys
- ✅ **No tracking** - We don't collect anything
- ✅ **Open source** - Audit the code yourself

### Why Open Source Matters

**Transparency work requires transparency tools:**
- See exactly what the code does
- Verify privacy and security yourself
- Contribute improvements
- Fork and customize for your needs
- No vendor lock-in

**We open source everything we use in our workflows** because we believe tools for transparency should themselves be transparent.

---

## Roadmap

### Current (v0.3.0)
- ✅ Core PDF operations (merge, split, optimize, render)
- ✅ Image processing and conversion
- ✅ Document ingestion with AI
- ✅ Annotation extraction
- ✅ Table of contents generation

### Next (v0.4.0)
- 🔄 Unified CLI (consolidating all tools)
- 🔄 Batch processing improvements
- 🔄 Better error messages
- 🔄 Progress reporting

### Future
- 📋 FOIA template integration
- 📋 Redaction tools
- 📋 Form filling automation
- 📋 Multi-language OCR
- 📋 Web interface (optional)

---

## Community

### Contributing

We welcome contributions! Whether you're:
- Fixing bugs
- Adding features
- Improving documentation
- Sharing use cases
- Reporting issues

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Support

- **Documentation**: [docs/](docs/)
- **Issues**: [GitHub Issues](https://github.com/The-HOLE-Foundation/hole-docs/issues)
- **Discussions**: [GitHub Discussions](https://github.com/The-HOLE-Foundation/hole-docs/discussions)

### Who We Are

**The HOLE Foundation** (Honoring Our Legal Ecosystem) is a nonprofit dedicated to government transparency, accountability, and access to public records.

- Website: [theholetruth.org](https://theholetruth.org)
- Mission: Make public records accessible to everyone
- Values: Open source, privacy, transparency

---

## License

MIT License - Use freely, modify as needed, share with others.

See [LICENSE](LICENSE) for details.

---

## Alternatives

**Why choose hole-docs over alternatives?**

| Feature | hole-docs | Adobe Acrobat | PDFtk | PyPDF2 |
|---------|-----------|---------------|-------|--------|
| **Price** | Free (forever) | $240/year | Free | Free |
| **Install** | One binary | 2GB download | Separate tools | Python required |
| **Privacy** | Local only | Cloud sync | Local | Local |
| **Speed** | Fast (Go) | Slow (Electron) | Fast | Slow (Python) |
| **AI Features** | ✅ OCR, classify | ✅ Paid add-ons | ❌ | ❌ |
| **Merge PDFs** | ✅ Lossless | ✅ | ✅ | ⚠️ Lossy |
| **Optimize** | ✅ | ✅ | ❌ | ❌ |
| **CLI** | ✅ | ❌ | ✅ | ✅ Script |
| **Open Source** | ✅ MIT | ❌ | ✅ GPL | ✅ BSD |

**hole-docs** combines the best of all worlds: free, fast, powerful, and privacy-respecting.

---

## Quick Examples

### Transparency Workflow Example

A real-world example of how you might use hole-docs:

```bash
# 1. Received FOIA response (3 separate PDFs)
hole-docs merge complete-response.pdf part1.pdf part2.pdf part3.pdf

# 2. Too large to email (50MB) - optimize it
hole-docs optimize complete-response.pdf optimized-response.pdf

# 3. Need to analyze specific pages - render to images
hole-docs render optimized-response.pdf ./analysis/ --pages 10-25

# 4. Extract highlighted sections from legal review
hole-docs annotations reviewed-draft.pdf > reviewer-notes.json

# 5. Create archival PDF from phone photos of documents
hole-docs images-to-pdf evidence.pdf photo1.jpg photo2.jpg photo3.jpg
```

**Total cost**: $0
**Time saved**: Hours
**Software required**: Just hole-docs

---

**Built with ❤️ by [The HOLE Foundation](https://theholetruth.org)**

*Making transparency accessible to everyone, one document at a time.*
