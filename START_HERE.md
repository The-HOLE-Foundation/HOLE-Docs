# START HERE - hole-docs Quick Reference

**Your new unified PDF toolkit for transparency workflows**

---

## ✅ What's Done

### Repository Live
- **GitHub**: https://github.com/The-HOLE-Foundation/hole-docs
- **Status**: Public, MIT licensed
- **Commits**: 3 initial commits pushed

### Working Binary
- **Location**: `./bin/hole-docs` (21MB)
- **Built**: Successfully compiles
- **Tested**: Help, version, merge, render, serve commands work

### Documentation
- ✅ Mission-driven README.md
- ✅ Installation guide (INSTALL.md)
- ✅ Architecture docs (ARCHITECTURE.md)
- ✅ Migration plan (MIGRATION_PLAN.md)
- ✅ Release workflow (.github/workflows/release.yml)

---

## 🎯 Quick Test

```bash
cd /Volumes/HOLE-RAID-DRIVE/Projects/hole-docs

# Build
make build

# Test commands
./bin/hole-docs help                    # ✅ Works
./bin/hole-docs version                 # ✅ Shows: dev
./bin/hole-docs merge                   # ✅ Shows help
./bin/hole-docs serve --help            # ✅ Shows server options
```

---

## 🚧 What's Next

### Complete Command Implementations (4-6 hours)

**Copy from HOLE-godocs** `cmd/godocs/main.go` and `cmd/pdf-merge/main.go`:

1. **Split command** → `commands/split.go`
2. **Optimize command** → `commands/optimize.go`
3. **Validate command** → `commands/validate.go`
4. **Info command** → `commands/info.go`
5. **Annotations command** → `commands/annotations.go`
6. **TOC command** → `commands/toc.go`
7. **Images-to-PDF command** → `commands/images_to_pdf.go`
8. **Upload command** → `commands/upload.go`
9. **Search command** → `commands/search.go`
10. **List-docs command** → `commands/list_docs.go`
11. **Config commands** → Complete `commands.go` config functions

**Pattern for each**:
```bash
# 1. Find the command in HOLE-godocs
# 2. Extract the logic
# 3. Create new file in hole-docs/cmd/hole-docs/commands/
# 4. Update imports to use github.com/The-HOLE-Foundation/hole-docs
# 5. Test: make build && ./bin/hole-docs <command>
```

### Add MCP Stdio Mode (2 hours)

Currently `hole-docs serve --stdio` shows "coming soon".

**Implement**:
- Read MCP protocol messages from stdin
- Write responses to stdout
- Enable Claude Desktop integration

### Documentation (3 hours)

**Create**:
- `docs/usage/QUICK_START.md` - 5-minute tutorial
- `docs/usage/TRANSPARENCY_WORKFLOWS.md` - Real FOIA examples
- `docs/comparison/VS_ADOBE.md` - Why choose hole-docs
- `docs/api/MCP_TOOLS.md` - Copy from HOLE-godocs

### Testing (2 hours)

```bash
# Test on real PDFs
hole-docs merge test-output.pdf test1.pdf test2.pdf
hole-docs render test.pdf ./test-images/
hole-docs annotations test-annotated.pdf

# Test MCP server
hole-docs serve --http &
curl http://localhost:8080/tools
```

### First Release (1 hour)

```bash
# When everything works:
git tag v0.1.0 -m "First unified release"
git push origin v0.1.0

# GitHub Actions builds for all platforms
# Download and test on fresh machines
```

---

## 📁 Key Files

**For development**:
- `cmd/hole-docs/main.go` - Entry point
- `cmd/hole-docs/commands/` - Command implementations
- `internal/` - Reusable libraries (already copied)
- `Makefile` - Build commands

**For reference** (HOLE-godocs):
- `/Volumes/HOLE-RAID-DRIVE/Projects/HOLE-godocs/cmd/godocs/main.go` - Original CLI
- `/Volumes/HOLE-RAID-DRIVE/Projects/HOLE-godocs/cmd/pdf-merge/main.go` - PDF utils
- `/Volumes/HOLE-RAID-DRIVE/Projects/HOLE-godocs/cmd/server/main.go` - MCP server

**For users**:
- `README.md` - Mission and overview
- `INSTALL.md` - How to install
- `ARCHITECTURE.md` - How it works

---

## 🎨 The Mission

**Remember why we're building this**:

> "Access to public records shouldn't require expensive software.
> hole-docs provides professional PDF tools for transparency workflows -
> completely free and open source."

**For**:
- FOIA requesters
- Transparency advocates
- Journalists
- Researchers
- Anyone who needs to work with public records

**Replaces**:
- Adobe Acrobat Pro ($240/year)
- PDF editing tools ($120+/year)
- OCR software ($180+/year)

**Total savings**: $500+/year per user

---

## 🏃 Quick Commands

```bash
# Build
make build

# Test
make test

# Clean
make clean

# Build all platforms (takes 2-3 minutes)
make release

# Install locally
make install-local  # to ~/.local/bin

# Install system-wide
make install        # to /usr/local/bin (requires sudo)

# View all targets
make help
```

---

## 📞 Current Status

**Location**: `/Volumes/HOLE-RAID-DRIVE/Projects/hole-docs`
**Branch**: `main`
**Version**: `dev` (pre-release)
**Binary**: `./bin/hole-docs` (21MB)

**Repository**: https://github.com/The-HOLE-Foundation/hole-docs

**What works**:
- ✅ Build system
- ✅ Help and version
- ✅ Merge command
- ✅ Render command
- ✅ MCP server (HTTP mode)

**What's next**:
- 🔄 Complete remaining commands (4-6 hours)
- 🔄 Documentation (3 hours)
- 🔄 Testing (2 hours)
- 🔄 First release (v0.1.0)

---

**Estimated time to v0.1.0**: 10-12 hours of focused work

**Ready to complete the remaining commands and release!** 🎉
