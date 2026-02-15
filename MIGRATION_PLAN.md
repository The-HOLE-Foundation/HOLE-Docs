# Migration Plan: HOLE-godocs → hole-docs

**Date**: 2026-02-14
**Status**: Planning phase

---

## What We Discovered

### MCP Server ✅ WORKS!

Tested the `godocs-server` binary - it's fully functional:
- ✅ Starts successfully on port 8080
- ✅ Registered 18 MCP tools
- ✅ Health endpoint responds: `{"status":"healthy","version":"0.1.0"}`
- ✅ Has PDF, image, Word, and document processing tools

**This is valuable functionality we didn't know existed!**

### Current State (3 Binaries)

**From HOLE-godocs project**:

1. **godocs** (CLI tool)
   - Commands: upload, search, list-docs, render, annotations, toc, optimize, config
   - Size: ~25MB binary
   - Working: ✅

2. **godocs-server** (MCP server)
   - 18 MCP tools for AI assistants
   - HTTP mode on port 8080
   - Working: ✅ (just tested!)

3. **godocs-pdf-merge** (PDF utilities)
   - Commands: merge, merge-dir, merge-images, validate, info
   - Specialized PDF operations
   - Working: ✅

**Problem**: Users have to download 3 separate binaries (confusing!)

---

## Goal: Unified hole-docs

### One Binary, All Functionality

**Single binary**: `hole-docs`

**Three operating modes**:

1. **CLI mode** (default) - All PDF/document operations
2. **MCP server mode** (`hole-docs serve`) - AI assistant integration
3. **Batch mode** - Process multiple files

### Command Consolidation Map

**From godocs CLI** → **hole-docs**:
```bash
godocs upload doc.pdf              → hole-docs upload doc.pdf
godocs search "query"              → hole-docs search "query"
godocs render doc.pdf              → hole-docs render doc.pdf
godocs annotations doc.pdf         → hole-docs annotations doc.pdf
godocs toc doc.pdf                 → hole-docs toc doc.pdf
godocs optimize doc.pdf            → hole-docs optimize doc.pdf
godocs config show                 → hole-docs config show
```

**From godocs-pdf-merge** → **hole-docs**:
```bash
godocs-pdf-merge merge out.pdf in.pdf  → hole-docs merge out.pdf in.pdf
godocs-pdf-merge merge-dir out.pdf dir → hole-docs merge-dir out.pdf dir
godocs-pdf-merge merge-images out.pdf  → hole-docs images-to-pdf out.pdf ...
godocs-pdf-merge validate doc.pdf      → hole-docs validate doc.pdf
godocs-pdf-merge info doc.pdf          → hole-docs info doc.pdf
```

**From godocs-server** → **hole-docs serve**:
```bash
godocs-server                      → hole-docs serve
PORT=3000 godocs-server            → hole-docs serve --port 3000
                                     hole-docs serve --stdio  # NEW: stdio mode
```

**Result**: Simpler, more intuitive interface.

---

## Migration Strategy

### Phase 1: Test Current Functionality ✅ DONE

- ✅ Tested MCP server (works!)
- ✅ Verified CLI commands (works!)
- ✅ Confirmed PDF merge utility (works!)

### Phase 2: Create New Project Structure

**New repository**: `hole-docs`

**Directory structure**:
```
hole-docs/
├── cmd/
│   └── hole-docs/          # Single unified binary
│       ├── main.go         # Entry point + command routing
│       ├── commands/       # Command implementations
│       │   ├── merge.go
│       │   ├── split.go
│       │   ├── render.go
│       │   ├── serve.go    # MCP server mode
│       │   └── ...
│       └── serve/          # Server-specific code
│           ├── http.go     # HTTP mode
│           └── stdio.go    # Stdio mode
├── internal/
│   ├── pdf/                # PDF processing (from HOLE-godocs)
│   ├── images/             # Image processing (from HOLE-godocs)
│   ├── mcp/                # MCP server logic (from HOLE-godocs)
│   ├── pipeline/           # Document pipeline (from HOLE-godocs)
│   ├── cli/                # CLI utilities
│   └── ...                 # All other internal packages
├── docs/
│   ├── installation/       # Install guides
│   ├── usage/              # Usage examples
│   ├── transparency/       # Transparency workflow guides
│   └── api/                # MCP API reference
├── README.md               # Mission-driven overview
├── INSTALL.md              # Installation guide
├── LICENSE                 # MIT license
└── Makefile                # Build automation
```

### Phase 3: Copy Working Code

**From HOLE-godocs** → **hole-docs**:

1. **Copy internal packages** (95% reusable):
   ```bash
   cp -r HOLE-godocs/internal/* hole-docs/internal/
   ```

2. **Merge CLI commands**:
   - Combine `cmd/godocs/main.go` + `cmd/pdf-merge/main.go`
   - Create unified command structure
   - Add `serve` subcommand from `cmd/server/main.go`

3. **Copy tests**:
   ```bash
   cp -r HOLE-godocs/tests/* hole-docs/tests/
   ```

4. **Migrate documentation**:
   - Update for new command structure
   - Add transparency-focused guides
   - Emphasize free/open-source mission

### Phase 4: Build & Test

```bash
# Build unified binary
cd hole-docs
make build

# Test CLI commands
./bin/hole-docs merge test.pdf a.pdf b.pdf
./bin/hole-docs render test.pdf ./output/

# Test MCP server
./bin/hole-docs serve --http &
curl http://localhost:8080/health
```

### Phase 5: Release

```bash
# Create v0.1.0 release
git tag v0.1.0
git push origin v0.1.0

# GitHub Actions builds:
# - hole-docs-v0.1.0-macos-arm64.tar.gz
# - hole-docs-v0.1.0-macos-amd64.tar.gz
# - hole-docs-v0.1.0-linux-amd64.tar.gz
# - hole-docs-v0.1.0-linux-arm64.tar.gz
```

---

## Code Migration Details

### Main Entry Point

**New**: `cmd/hole-docs/main.go`

```go
package main

import (
    "fmt"
    "os"

    "github.com/HOLE-Foundation/hole-docs/cmd/hole-docs/commands"
)

func main() {
    if len(os.Args) < 2 {
        commands.ShowHelp()
        os.Exit(1)
    }

    command := os.Args[1]

    switch command {
    case "merge":
        commands.Merge(os.Args[2:])
    case "split":
        commands.Split(os.Args[2:])
    case "render":
        commands.Render(os.Args[2:])
    case "serve":
        commands.Serve(os.Args[2:])  // MCP server mode
    case "upload":
        commands.Upload(os.Args[2:])
    // ... all other commands
    case "help", "-h", "--help":
        commands.ShowHelp()
    default:
        fmt.Printf("Unknown command: %s\n", command)
        commands.ShowHelp()
        os.Exit(1)
    }
}
```

### Serve Command

**New**: `cmd/hole-docs/commands/serve.go`

```go
package commands

import (
    "flag"
    "fmt"

    "github.com/HOLE-Foundation/hole-docs/internal/mcp"
    "github.com/HOLE-Foundation/hole-docs/internal/config"
)

func Serve(args []string) {
    fs := flag.NewFlagSet("serve", flag.ExitOnError)

    httpMode := fs.Bool("http", true, "HTTP mode (default)")
    stdioMode := fs.Bool("stdio", false, "Stdio mode for MCP clients")
    port := fs.Int("port", 8080, "Port for HTTP mode")

    fs.Parse(args)

    cfg := config.FromEnv()
    cfg.Port = *port

    if *stdioMode {
        // Start stdio MCP server
        mcp.ServeStdio(cfg)
    } else {
        // Start HTTP MCP server
        mcp.ServeHTTP(cfg)
    }
}
```

### Reusable Internal Packages

**These packages copy directly** from HOLE-godocs (no changes needed):
- `internal/pdf/` - PDF processing logic
- `internal/images/` - Image processing
- `internal/mcp/` - MCP server implementation
- `internal/pipeline/` - Document processing pipeline
- `internal/mistral/` - Mistral AI client
- `internal/voyage/` - Voyage AI client
- `internal/neon/` - Database client
- `internal/storage/` - R2 storage
- `internal/config/` - Configuration
- And ~15 more packages

**These are battle-tested and working** - we just need to consolidate the entry points.

---

## Testing Strategy

### Test Each Mode

**1. CLI mode**:
```bash
# Basic PDF operations
hole-docs merge output.pdf file1.pdf file2.pdf
hole-docs split document.pdf ./pages/
hole-docs render document.pdf ./images/

# Advanced operations
hole-docs annotations reviewed.pdf
hole-docs toc document.pdf
hole-docs optimize large.pdf small.pdf
```

**2. MCP server mode (HTTP)**:
```bash
# Start server
hole-docs serve --http --port 8080 &

# Test health
curl http://localhost:8080/health

# Test tool listing
curl http://localhost:8080/tools

# Test PDF merge via MCP
curl -X POST http://localhost:8080/call \
  -H "Content-Type: application/json" \
  -d '{"tool":"merge_pdfs","args":{"input_paths":["a.pdf","b.pdf"],"output_path":"out.pdf"}}'
```

**3. MCP server mode (Stdio)**:
```bash
# Configure in Claude Desktop
# ~/.config/claude-desktop/config.json
{
  "mcpServers": {
    "hole-docs": {
      "command": "/usr/local/bin/hole-docs",
      "args": ["serve", "--stdio"]
    }
  }
}
```

### Compatibility Testing

**Ensure all original functionality works**:
- [ ] All CLI commands from godocs
- [ ] All PDF operations from pdf-merge
- [ ] All MCP tools from server
- [ ] Configuration system
- [ ] Error handling
- [ ] Progress reporting

---

## Documentation Updates

### New Documentation Focus

**Emphasize the mission**:
- Free, not freemium
- Open source, auditable
- Privacy-respecting (local-first)
- Built for transparency advocates
- Replaces expensive proprietary tools

**Key documents**:
1. `README.md` - Mission-driven overview
2. `INSTALL.md` - Simple installation
3. `docs/usage/TRANSPARENCY_WORKFLOWS.md` - Real-world FOIA examples
4. `docs/usage/QUICK_START.md` - 5-minute guide
5. `docs/comparison/VS_ADOBE.md` - Why choose hole-docs

### Example Documentation

**Real transparency workflow** (`docs/usage/TRANSPARENCY_WORKFLOWS.md`):

```markdown
# Transparency Workflows

## Example: Processing a FOIA Response

You submitted a FOIA request to your city police department.
They sent you 3 large PDFs (150MB total) with 500+ pages.

Here's how hole-docs helps:

### Step 1: Merge the responses
```bash
hole-docs merge complete-response.pdf \
  response-part1.pdf \
  response-part2.pdf \
  response-part3.pdf
```

### Step 2: Optimize for sharing
```bash
# Reduce from 150MB to ~15MB
hole-docs optimize complete-response.pdf optimized.pdf
```

### Step 3: Extract relevant pages
```bash
# Just pages 45-67 (incident report)
hole-docs split optimized.pdf ./incident/ --pages 45-67
```

### Step 4: Render for analysis
```bash
# Convert to images for visual analysis
hole-docs render incident/pages-45-67.pdf ./photos/
```

**Time**: 5 minutes
**Cost**: $0
**Software needed**: Just hole-docs
```

---

## Timeline

### Week 1: Setup & Core CLI

**Days 1-2**: Project structure
- Create repository
- Set up build system
- Copy internal packages
- Create unified main.go

**Days 3-5**: Core CLI commands
- Implement merge, split, render
- Implement optimize, validate, info
- Implement annotations, toc
- Test all commands

**Days 6-7**: MCP server integration
- Integrate serve command
- Test HTTP mode
- Test stdio mode
- Verify all 18 tools work

### Week 2: Documentation & Release

**Days 8-10**: Documentation
- Write transparency workflow guides
- Create comparison docs (vs Adobe, etc.)
- Document all commands
- Create video tutorials (optional)

**Days 11-12**: Testing
- Test all platforms
- User acceptance testing
- Performance benchmarking

**Days 13-14**: Release
- Tag v0.1.0
- GitHub release
- Announce to community

---

## Success Criteria

**hole-docs v0.1.0 is ready when**:

- ✅ All HOLE-godocs functionality works
- ✅ Single binary (no separate tools)
- ✅ Builds for macOS and Linux
- ✅ Documentation complete
- ✅ Tests passing
- ✅ MCP server mode functional
- ✅ Installation under 1 minute
- ✅ Released on GitHub

---

## Post-Launch

### Community Building

- [ ] Announce on transparency forums
- [ ] Share with FOIA advocates
- [ ] Create tutorial videos
- [ ] Write blog post about free tools
- [ ] Gather user feedback

### Future Enhancements

**v0.2.0**:
- FOIA template integration
- Redaction tools
- Form filling

**v0.3.0**:
- Web interface (optional)
- Multi-language OCR
- Advanced search

---

## Next Steps

### Immediate Actions

1. **Create GitHub repository**:
   ```bash
   gh repo create The-HOLE-Foundation/hole-docs \
     --public \
     --description "Free PDF tools for transparency workflows" \
     --license MIT
   ```

2. **Set up project structure**:
   - Copy internal packages from HOLE-godocs
   - Create unified cmd/hole-docs/main.go
   - Set up build system
   - Write tests

3. **Initial commit**:
   ```bash
   git add .
   git commit -m "Initial commit: Unified hole-docs from HOLE-godocs

   Consolidates three binaries into one:
   - godocs (CLI)
   - godocs-server (MCP)
   - godocs-pdf-merge (PDF utils)

   Mission: Make public records accessible to everyone with
   free, open-source tools."

   git remote add origin git@github.com:The-HOLE-Foundation/hole-docs.git
   git push -u origin main
   ```

---

## Backward Compatibility

### For Existing HOLE-godocs Users

**Migration guide**:
```bash
# Old commands still work with new names
godocs merge a.pdf b.pdf    → hole-docs merge a.pdf b.pdf
godocs-server               → hole-docs serve
godocs-pdf-merge merge ...  → hole-docs merge ...
```

**Transition period**:
- HOLE-godocs repository archived (read-only)
- Links redirect to hole-docs
- Migration guide in HOLE-godocs README

---

## Repository Setup

### GitHub Repository Details

**Name**: `hole-docs`
**Organization**: The-HOLE-Foundation
**Visibility**: Public
**License**: MIT
**Description**: Free PDF tools for transparency workflows
**Topics**: `pdf`, `foia`, `transparency`, `open-records`, `cli`, `golang`, `mcp`

### Repository Structure

```
hole-docs/
├── .github/
│   └── workflows/
│       ├── release.yml      # Automated releases
│       ├── test.yml         # CI testing
│       └── lint.yml         # Code quality
├── cmd/hole-docs/           # Unified binary
├── internal/                # Shared libraries
├── docs/                    # Documentation
├── README.md                # Mission-driven overview
├── INSTALL.md               # Installation guide
├── ARCHITECTURE.md          # Technical design
├── CONTRIBUTING.md          # How to contribute
├── LICENSE                  # MIT license
└── Makefile                 # Build automation
```

---

## Communication Plan

### Announcement

**When v0.1.0 is ready**:

**GitHub release notes**:
```markdown
# hole-docs v0.1.0 🎉

**Free PDF tools for transparency workflows**

Making public records accessible to everyone - no expensive software required.

## What's New

This is the first release of hole-docs, consolidating three separate
tools into one unified binary:

- ✅ Merge, split, optimize PDFs
- ✅ Render PDFs to images
- ✅ Extract annotations and TOC
- ✅ Convert images to archival PDFs
- ✅ AI-powered document processing
- ✅ MCP server for AI assistant integration

## Why hole-docs?

Access to public records shouldn't require expensive software.
hole-docs replaces tools costing $240-360/year with a free,
open-source alternative.

Built by The HOLE Foundation for transparency advocates everywhere.

## Install

See [INSTALL.md](INSTALL.md) for installation instructions.
```

**Target audiences**:
- Transparency advocates
- FOIA requesters
- Journalists
- Researchers
- Open government advocates
- Legal professionals

---

## Risks & Mitigation

### Risk: Losing Functionality

**Mitigation**: Copy ALL internal packages first, test thoroughly

### Risk: Breaking Changes

**Mitigation**: Keep command names similar, provide migration guide

### Risk: Users Can't Find New Tool

**Mitigation**:
- Update HOLE-godocs README with redirect
- Archive old repository with clear pointer
- Announce on all channels

---

## Metrics for Success

**Launch week**:
- [ ] 50+ downloads
- [ ] 5+ stars on GitHub
- [ ] 0 critical bugs

**First month**:
- [ ] 200+ downloads
- [ ] 20+ stars
- [ ] 3+ community contributions
- [ ] Featured in transparency community

**First year**:
- [ ] 1000+ active users
- [ ] Save users $100,000+ in software costs
- [ ] 100+ stars on GitHub
- [ ] Used in real FOIA workflows

---

## Current Status

- ✅ **Tested**: MCP server works!
- ✅ **Created**: New project directory
- ✅ **Planned**: Architecture and migration
- 🔄 **Next**: Set up project structure and copy code

---

**Ready to build the unified tool!** 🚀
