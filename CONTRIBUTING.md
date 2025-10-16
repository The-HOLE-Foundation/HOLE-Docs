# Contributing to HOLE-Docs

Thank you for your interest in contributing to The HOLE Foundation's documentation repository!

## How to Contribute

### Adding New Tool Documentation

1. Navigate to the `docs/tools/` directory
2. Create a new directory with your tool's name (use lowercase and hyphens)
   ```bash
   mkdir docs/tools/your-tool-name
   ```
3. Create a comprehensive README.md in your tool's directory
4. Update `docs/tools/README.md` to include a link to your documentation

### Adding Foundation Work Documentation

1. Navigate to the `docs/foundation/` directory
2. Create a new directory for your project or initiative
3. Add a detailed README.md file
4. Update `docs/foundation/README.md` with a link to your documentation

## Documentation Standards

### File Structure

- Each tool or project should have its own directory
- Every directory must contain a README.md file
- Use additional files (e.g., guides, tutorials) as needed
- Keep related documentation together

### Markdown Guidelines

1. **Headers**: Use proper heading hierarchy
   ```markdown
   # Main Title (H1) - Only one per document
   ## Section (H2)
   ### Subsection (H3)
   ```

2. **Code Blocks**: Always specify the language
   ````markdown
   ```bash
   your-command here
   ```
   ````

3. **Links**: Use relative links for internal documentation
   ```markdown
   [Link Text](./relative/path/to/doc.md)
   ```

4. **Lists**: Use consistent formatting
   - Unordered lists with `-` or `*`
   - Ordered lists with `1.`, `2.`, etc.

### Content Guidelines

1. **Be Clear and Concise**: Write for your audience
2. **Include Examples**: Show, don't just tell
3. **Keep it Updated**: Documentation should reflect current state
4. **Use Proper Grammar**: Professional documentation requires professional writing
5. **Add Context**: Explain why, not just what

## Documentation Template

Use this template for new tool documentation:

```markdown
# Tool Name

Brief description of the tool.

## Overview

What does this tool do and why is it useful?

## Key Features

- Feature 1
- Feature 2
- Feature 3

## Getting Started

### Prerequisites
List any requirements

### Installation
Step-by-step installation guide

### Basic Usage
Simple examples to get started

## Best Practices

Tips for using the tool effectively

## Resources

- Links to related documentation
- External resources

## Contributing

Link back to this contributing guide
```

## Review Process

1. Ensure your documentation follows the standards above
2. Check all links work correctly
3. Verify code examples are accurate
4. Update index files appropriately
5. Submit your changes via pull request

## Questions or Issues?

If you have questions about contributing, please open an issue in the repository.

## Code of Conduct

Be respectful and constructive in all interactions. This is a professional documentation repository serving The HOLE Foundation's mission.
