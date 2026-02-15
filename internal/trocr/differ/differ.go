package differ

import (
	"fmt"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
	"go.uber.org/zap"
)

// Differ computes differences between original and marked text
type Differ struct {
	dmp    *diffmatchpatch.DiffMatchPatch
	logger *zap.Logger
}

// Change represents a text change on a specific page
type Change struct {
	PageNum      int      // Page number (1-indexed)
	OriginalText string   // Original text from page
	MarkedText   string   // Marked/edited text from page
	DiffHTML     string   // HTML representation of diff
	DiffText     string   // Plain text representation
	Additions    []string // Lines added
	Deletions    []string // Lines deleted
	Modifications []string // Lines modified
	HasChanges   bool     // True if page has any changes
}

// NewDiffer creates a new text differ
func NewDiffer(logger *zap.Logger) *Differ {
	return &Differ{
		dmp:    diffmatchpatch.New(),
		logger: logger,
	}
}

// ComputeChanges finds all changes between original and marked page texts
// Returns slice of Change objects (only for pages with changes)
func (d *Differ) ComputeChanges(original, marked map[int]string) ([]Change, error) {
	d.logger.Info("Computing text differences",
		zap.Int("original_pages", len(original)),
		zap.Int("marked_pages", len(marked)))

	var changes []Change

	// Process each page
	for pageNum := 1; pageNum <= len(original); pageNum++ {
		origText := original[pageNum]
		markText := marked[pageNum]

		// Skip if no changes
		if origText == markText {
			continue
		}

		// Compute diff
		change := d.computePageDiff(pageNum, origText, markText)
		changes = append(changes, change)

		d.logger.Info("Changes detected",
			zap.Int("page", pageNum),
			zap.Int("additions", len(change.Additions)),
			zap.Int("deletions", len(change.Deletions)),
			zap.Int("modifications", len(change.Modifications)))
	}

	d.logger.Info("Diff computation complete",
		zap.Int("pages_with_changes", len(changes)),
		zap.Int("total_pages", len(original)))

	return changes, nil
}

// computePageDiff computes detailed diff for a single page
func (d *Differ) computePageDiff(pageNum int, origText, markText string) Change {
	// Compute line-level diff
	diffs := d.dmp.DiffMain(origText, markText, false)

	// Cleanup for readability (semantic cleanup)
	diffs = d.dmp.DiffCleanupSemantic(diffs)

	// Generate representations
	diffHTML := d.dmp.DiffPrettyHtml(diffs)
	diffText := d.dmp.DiffPrettyText(diffs)

	// Extract additions, deletions, modifications
	additions := []string{}
	deletions := []string{}
	modifications := []string{}

	for _, diff := range diffs {
		switch diff.Type {
		case diffmatchpatch.DiffInsert:
			// Text added in marked version
			lines := strings.Split(strings.TrimSpace(diff.Text), "\n")
			additions = append(additions, lines...)
		case diffmatchpatch.DiffDelete:
			// Text removed in marked version
			lines := strings.Split(strings.TrimSpace(diff.Text), "\n")
			deletions = append(deletions, lines...)
		case diffmatchpatch.DiffEqual:
			// No change
		}
	}

	// If we have both additions and deletions at same location, treat as modification
	if len(additions) > 0 && len(deletions) > 0 {
		modifications = append(modifications, fmt.Sprintf("%s → %s",
			strings.Join(deletions, "; "),
			strings.Join(additions, "; ")))
	}

	return Change{
		PageNum:       pageNum,
		OriginalText:  origText,
		MarkedText:    markText,
		DiffHTML:      diffHTML,
		DiffText:      diffText,
		Additions:     additions,
		Deletions:     deletions,
		Modifications: modifications,
		HasChanges:    len(additions) > 0 || len(deletions) > 0,
	}
}

// GenerateSummary creates a human-readable summary of all changes
func (d *Differ) GenerateSummary(changes []Change) string {
	if len(changes) == 0 {
		return "No changes detected between original and marked PDFs."
	}

	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("=== Visual Diff Summary ===\n"))
	summary.WriteString(fmt.Sprintf("Total pages with changes: %d\n\n", len(changes)))

	for _, change := range changes {
		summary.WriteString(fmt.Sprintf("Page %d:\n", change.PageNum))

		if len(change.Additions) > 0 {
			summary.WriteString(fmt.Sprintf("  Additions (%d):\n", len(change.Additions)))
			for _, add := range change.Additions {
				summary.WriteString(fmt.Sprintf("    + %s\n", truncate(add, 100)))
			}
		}

		if len(change.Deletions) > 0 {
			summary.WriteString(fmt.Sprintf("  Deletions (%d):\n", len(change.Deletions)))
			for _, del := range change.Deletions {
				summary.WriteString(fmt.Sprintf("    - %s\n", truncate(del, 100)))
			}
		}

		if len(change.Modifications) > 0 {
			summary.WriteString(fmt.Sprintf("  Modifications (%d):\n", len(change.Modifications)))
			for _, mod := range change.Modifications {
				summary.WriteString(fmt.Sprintf("    ~ %s\n", truncate(mod, 150)))
			}
		}

		summary.WriteString("\n")
	}

	return summary.String()
}

// truncate limits string length for display
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ExportToJSON converts changes to JSON for programmatic access
func (d *Differ) ExportToJSON(changes []Change) ([]byte, error) {
	// This would use encoding/json to export changes
	// For now, return empty to avoid import
	return nil, nil
}
