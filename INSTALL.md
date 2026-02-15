# Installation Guide

**hole-docs** - Free PDF tools for transparency workflows

---

## Quick Install

### macOS

**Apple Silicon (M1/M2/M3 - Recommended)**:
```bash
curl -LO https://github.com/The-HOLE-Foundation/hole-docs/releases/latest/download/hole-docs-latest-macos-arm64.tar.gz
tar -xzf hole-docs-latest-macos-arm64.tar.gz
sudo mv hole-docs /usr/local/bin/
hole-docs help
```

**Intel Macs**:
```bash
curl -LO https://github.com/The-HOLE-Foundation/hole-docs/releases/latest/download/hole-docs-latest-macos-amd64.tar.gz
tar -xzf hole-docs-latest-macos-amd64.tar.gz
sudo mv hole-docs /usr/local/bin/
hole-docs help
```

### Linux

**Ubuntu/Debian (AMD64)**:
```bash
curl -LO https://github.com/The-HOLE-Foundation/hole-docs/releases/latest/download/hole-docs-latest-linux-amd64.tar.gz
tar -xzf hole-docs-latest-linux-amd64.tar.gz
sudo mv hole-docs /usr/local/bin/
hole-docs help
```

**Raspberry Pi / ARM (ARM64)**:
```bash
curl -LO https://github.com/The-HOLE-Foundation/hole-docs/releases/latest/download/hole-docs-latest-linux-arm64.tar.gz
tar -xzf hole-docs-latest-linux-arm64.tar.gz
sudo mv hole-docs /usr/local/bin/
hole-docs help
```

---

## System Dependencies

hole-docs requires these free tools (install once):

### macOS
```bash
brew install qpdf mupdf ghostscript
```

### Ubuntu/Debian
```bash
sudo apt update
sudo apt install qpdf mupdf-tools ghostscript
```

### Fedora/RHEL
```bash
sudo dnf install qpdf mupdf ghostscript
```

**That's it!** No other dependencies needed.

---

## Verify Installation

```bash
# Check hole-docs
hole-docs version

# Test merge command
hole-docs merge --help

# Test render command
hole-docs render --help
```

Expected output: Command help text without errors.

---

## Configuration (Optional)

For AI-powered features, set environment variables:

```bash
# Create .env file (optional)
cat > ~/.hole-docs.env << 'EOF'
# Optional: AI features (OCR, classification)
MISTRAL_API_KEY=your-mistral-key
VOYAGEAI_API_KEY=your-voyage-key

# Optional: Storage (for upload/search commands)
DATABASE_URL=postgresql://user:pass@host/db
CLOUDFLARE_ACCOUNT_ID=your-account
CLOUDFLARE_R2_BUCKET_NAME=your-bucket
EOF
```

**Note**: All PDF operations work WITHOUT these keys. They're only needed for AI features (upload, search, classification).

---

## Usage Examples

### Merge FOIA Responses
```bash
hole-docs merge complete-response.pdf \
  agency1-response.pdf \
  agency2-response.pdf \
  agency3-response.pdf
```

### Render PDF to Images
```bash
hole-docs render police-report.pdf ./evidence-photos/
```

### Extract Reviewer Comments
```bash
hole-docs annotations reviewed-document.pdf
```

### Optimize Large PDF
```bash
hole-docs optimize 50mb-file.pdf small-file.pdf
```

---

## Alternative Installation (No sudo)

Install to user directory if you don't have sudo access:

```bash
# Create user bin directory
mkdir -p ~/.local/bin

# Download and extract (choose your platform)
curl -LO https://github.com/The-HOLE-Foundation/hole-docs/releases/latest/download/hole-docs-latest-linux-amd64.tar.gz
tar -xzf hole-docs-latest-linux-amd64.tar.gz

# Move to user directory
mv hole-docs ~/.local/bin/

# Add to PATH (add this to ~/.bashrc or ~/.zshrc)
export PATH="$HOME/.local/bin:$PATH"

# Reload shell
source ~/.bashrc  # or source ~/.zshrc

# Test
hole-docs help
```

---

## Upgrading

To upgrade to a new version:

```bash
# Download new version (replace with actual version)
curl -LO https://github.com/The-HOLE-Foundation/hole-docs/releases/download/v0.2.0/hole-docs-v0.2.0-linux-amd64.tar.gz

# Extract
tar -xzf hole-docs-v0.2.0-linux-amd64.tar.gz

# Replace existing binary
sudo mv hole-docs /usr/local/bin/

# Verify
hole-docs version
```

---

## Uninstall

```bash
# Remove binary
sudo rm /usr/local/bin/hole-docs

# Remove configuration (optional)
rm -rf ~/.hole-docs/
rm ~/.hole-docs.env
```

---

## Troubleshooting

### Permission Denied

```bash
# Make binary executable
chmod +x hole-docs
```

### Command Not Found

```bash
# Check if installed
which hole-docs

# If using ~/.local/bin, ensure PATH is set
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

### qpdf Not Found

```bash
# Install system dependencies
brew install qpdf mupdf ghostscript  # macOS
sudo apt install qpdf                 # Ubuntu
```

### macOS Security Warning

First run may show: "hole-docs cannot be opened"

**Solution**:
```bash
# Remove quarantine flag
xattr -d com.apple.quarantine hole-docs

# Or: System Preferences → Security & Privacy → Allow
```

---

## Support

- **Documentation**: https://github.com/The-HOLE-Foundation/hole-docs
- **Issues**: https://github.com/The-HOLE-Foundation/hole-docs/issues
- **Website**: https://theholetruth.org

---

**Making public records accessible to everyone** 🌟

Built by The HOLE Foundation
