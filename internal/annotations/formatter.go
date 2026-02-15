package annotations

import (
	"fmt"
	"sort"
	"strings"
)

// FormatForLLM produces the primary revision-instruction format designed for LLM consumption.
// It groups annotations by page, includes summary statistics, and appends OCR text
// when available.
func FormatForLLM(set *AnnotationSet) string {
	if set == nil || len(set.Annotations) == 0 {
		return "## Revision Instructions\n\nNo annotations found in document.\n"
	}

	var b strings.Builder

	// Header
	b.WriteString("## Revision Instructions\n\n")
	b.WriteString(fmt.Sprintf("Source: %s\n", set.SourceFile))
	b.WriteString(fmt.Sprintf("%d annotations", set.Summary.TotalCount))
	if len(set.Summary.ByAuthor) > 0 {
		authors := sortedKeys(set.Summary.ByAuthor)
		b.WriteString(fmt.Sprintf(" from %d reviewer(s)", len(authors)))
	}
	b.WriteString(fmt.Sprintf(" across %d page(s).\n\n", set.Summary.PagesAnnotated))

	// Summary
	b.WriteString("### Summary\n")
	for cat, count := range set.Summary.ByCategory {
		b.WriteString(fmt.Sprintf("- %d %s\n", count, cat))
	}
	if len(set.Summary.ByAuthor) > 0 {
		for _, author := range sortedKeys(set.Summary.ByAuthor) {
			b.WriteString(fmt.Sprintf("- %d from %s\n", set.Summary.ByAuthor[author], author))
		}
	}
	b.WriteString("\n")

	// Group annotations by page
	byPage := groupByPage(set.Annotations)
	pages := sortedIntKeys(byPage)

	idx := 1
	for _, pageNr := range pages {
		b.WriteString(fmt.Sprintf("### Page %d\n\n", pageNr))

		for _, a := range byPage[pageNr] {
			b.WriteString(fmt.Sprintf("[%d] %s (%s)",
				idx, strings.ToUpper(string(a.Category)), a.Priority))
			if a.Author != "" {
				b.WriteString(fmt.Sprintf(" by %s", a.Author))
			}
			b.WriteString("\n")

			if a.AnnotationText != "" {
				b.WriteString(fmt.Sprintf("  Comment: %q\n", a.AnnotationText))
			}
			if a.AnchorText != "" {
				b.WriteString(fmt.Sprintf("  Anchored to: %q\n", a.AnchorText))
			}
			if a.RectString != "" {
				b.WriteString(fmt.Sprintf("  Position: %s\n", a.RectString))
			}
			if a.AnnotationText == "" && a.AnchorText == "" {
				b.WriteString("  Note: No comment text — see OCR text below for context.\n")
			}
			b.WriteString("\n")
			idx++
		}

		// Append OCR text for this page if available
		if ocrText, ok := set.OCRText[pageNr]; ok && ocrText != "" {
			b.WriteString(fmt.Sprintf("### Page %d — Full Text (from OCR)\n", pageNr))
			b.WriteString(ocrText)
			b.WriteString("\n\n")
		}
	}

	// Errors section
	if len(set.Errors) > 0 {
		b.WriteString("### Extraction Errors\n")
		for _, e := range set.Errors {
			b.WriteString(fmt.Sprintf("- %s\n", e))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// FormatAsJSON returns the AnnotationSet as indented JSON bytes.
func FormatAsJSON(set *AnnotationSet) ([]byte, error) {
	return set.ToJSON()
}

// FormatAsMarkdown produces a human-readable Markdown document.
func FormatAsMarkdown(set *AnnotationSet) string {
	if set == nil || len(set.Annotations) == 0 {
		return "# Annotations\n\nNo annotations found.\n"
	}

	var b strings.Builder

	b.WriteString(fmt.Sprintf("# Annotations: %s\n\n", set.SourceFile))
	b.WriteString(fmt.Sprintf("**Total**: %d annotations on %d page(s)\n\n",
		set.Summary.TotalCount, set.Summary.PagesAnnotated))

	// Summary table
	b.WriteString("| Category | Count |\n|---|---|\n")
	for cat, count := range set.Summary.ByCategory {
		b.WriteString(fmt.Sprintf("| %s | %d |\n", cat, count))
	}
	b.WriteString("\n")

	// Annotations by page
	byPage := groupByPage(set.Annotations)
	pages := sortedIntKeys(byPage)

	for _, pageNr := range pages {
		b.WriteString(fmt.Sprintf("## Page %d\n\n", pageNr))
		for i, a := range byPage[pageNr] {
			b.WriteString(fmt.Sprintf("%d. **%s** (%s) — %s",
				i+1, a.RawType, a.Category, a.Priority))
			if a.Author != "" {
				b.WriteString(fmt.Sprintf(" *by %s*", a.Author))
			}
			b.WriteString("\n")
			if a.AnnotationText != "" {
				b.WriteString(fmt.Sprintf("   > %s\n", a.AnnotationText))
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}

// FormatAsChatContext produces a compact format suitable for LLM system prompts
// where token budget is tight.
func FormatAsChatContext(set *AnnotationSet) string {
	if set == nil || len(set.Annotations) == 0 {
		return "ANNOTATIONS: None\n"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("ANNOTATIONS: %d total\n", set.Summary.TotalCount))

	for i, a := range set.Annotations {
		// Compact format: index. page CATEGORY "text" [Author]
		b.WriteString(fmt.Sprintf("%d. p%d %s", i+1, a.Page, strings.ToUpper(string(a.Category))))
		if a.AnnotationText != "" {
			text := a.AnnotationText
			if len(text) > 120 {
				text = text[:117] + "..."
			}
			b.WriteString(fmt.Sprintf(" %q", text))
		}
		if a.AnchorText != "" {
			anchor := a.AnchorText
			if len(anchor) > 80 {
				anchor = anchor[:77] + "..."
			}
			b.WriteString(fmt.Sprintf(" on %q", anchor))
		}
		if a.Author != "" {
			b.WriteString(fmt.Sprintf(" [%s]", a.Author))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// --- helpers ---

func groupByPage(annotations []ExtractedAnnotation) map[int][]ExtractedAnnotation {
	m := make(map[int][]ExtractedAnnotation)
	for _, a := range annotations {
		m[a.Page] = append(m[a.Page], a)
	}
	return m
}

func sortedIntKeys(m map[int][]ExtractedAnnotation) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
