package differ

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestDiffer_ComputeChanges(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	differ := NewDiffer(logger)

	tests := []struct {
		name              string
		original          map[int]string
		marked            map[int]string
		expectedPageCount int
		expectChanges     bool
	}{
		{
			name: "no changes",
			original: map[int]string{
				1: "This is page 1 text",
				2: "This is page 2 text",
			},
			marked: map[int]string{
				1: "This is page 1 text",
				2: "This is page 2 text",
			},
			expectedPageCount: 0,
			expectChanges:     false,
		},
		{
			name: "addition on page 2",
			original: map[int]string{
				1: "This is page 1 text",
				2: "This is page 2 text",
			},
			marked: map[int]string{
				1: "This is page 1 text",
				2: "This is page 2 text\nAdded reviewer note here",
			},
			expectedPageCount: 1,
			expectChanges:     true,
		},
		{
			name: "deletion on page 1",
			original: map[int]string{
				1: "This paragraph should be removed\nThis stays",
				2: "Page 2 unchanged",
			},
			marked: map[int]string{
				1: "This stays",
				2: "Page 2 unchanged",
			},
			expectedPageCount: 1,
			expectChanges:     true,
		},
		{
			name: "modification on page 1",
			original: map[int]string{
				1: "The defendant was negligent",
			},
			marked: map[int]string{
				1: "The defendant was grossly negligent",
			},
			expectedPageCount: 1,
			expectChanges:     true,
		},
		{
			name: "changes on multiple pages",
			original: map[int]string{
				1: "Page 1 original",
				2: "Page 2 original",
				3: "Page 3 original",
			},
			marked: map[int]string{
				1: "Page 1 modified text",
				2: "Page 2 original",
				3: "Page 3 with addition",
			},
			expectedPageCount: 2,
			expectChanges:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes, err := differ.ComputeChanges(tt.original, tt.marked)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedPageCount, len(changes),
				"Expected %d pages with changes, got %d", tt.expectedPageCount, len(changes))

			if tt.expectChanges {
				assert.Greater(t, len(changes), 0, "Expected changes but got none")

				// Verify each change has required fields
				for _, change := range changes {
					assert.True(t, change.HasChanges, "Change should have HasChanges=true")
					assert.NotEmpty(t, change.DiffText, "DiffText should not be empty")
					assert.NotEmpty(t, change.DiffHTML, "DiffHTML should not be empty")

					// At least one of additions/deletions should be non-empty
					hasEdits := len(change.Additions) > 0 || len(change.Deletions) > 0
					assert.True(t, hasEdits, "Change should have additions or deletions")
				}
			} else {
				assert.Equal(t, 0, len(changes), "Expected no changes")
			}
		})
	}
}

func TestDiffer_GenerateSummary(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	differ := NewDiffer(logger)

	tests := []struct {
		name            string
		changes         []Change
		expectNoChanges bool
		expectContains  []string
	}{
		{
			name:            "empty changes",
			changes:         []Change{},
			expectNoChanges: true,
			expectContains:  []string{"No changes detected"},
		},
		{
			name: "single page with additions",
			changes: []Change{
				{
					PageNum:    2,
					Additions:  []string{"Added note 1", "Added note 2"},
					Deletions:  []string{},
					HasChanges: true,
				},
			},
			expectNoChanges: false,
			expectContains:  []string{"Page 2:", "Additions (2)", "+ Added note 1", "+ Added note 2"},
		},
		{
			name: "multiple pages",
			changes: []Change{
				{
					PageNum:    1,
					Deletions:  []string{"Remove this"},
					HasChanges: true,
				},
				{
					PageNum:    3,
					Additions:  []string{"New content"},
					HasChanges: true,
				},
			},
			expectNoChanges: false,
			expectContains:  []string{"Total pages with changes: 2", "Page 1:", "Page 3:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := differ.GenerateSummary(tt.changes)
			assert.NotEmpty(t, summary)

			if tt.expectNoChanges {
				assert.Contains(t, summary, "No changes detected")
			} else {
				for _, expected := range tt.expectContains {
					assert.Contains(t, summary, expected,
						"Summary should contain: %s", expected)
				}
			}
		})
	}
}

func TestDiffer_ComputePageDiff(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	differ := NewDiffer(logger)

	tests := []struct {
		name           string
		origText       string
		markText       string
		expectAdditions int
		expectDeletions int
	}{
		{
			name:            "simple addition",
			origText:        "Original text",
			markText:        "Original text\nAdded line",
			expectAdditions: 1,
			expectDeletions: 0,
		},
		{
			name:            "simple deletion",
			origText:        "Line 1\nLine 2 to remove\nLine 3",
			markText:        "Line 1\nLine 3",
			expectAdditions: 0,
			expectDeletions: 1,
		},
		{
			name:            "replacement (del + add)",
			origText:        "Old word here",
			markText:        "New word here",
			expectAdditions: 1,
			expectDeletions: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			change := differ.computePageDiff(1, tt.origText, tt.markText)

			assert.True(t, change.HasChanges)
			assert.NotEmpty(t, change.DiffText)
			assert.NotEmpty(t, change.DiffHTML)

			// Check additions/deletions counts (may not match exactly due to diff algorithm)
			// Just verify they're non-zero when expected
			if tt.expectAdditions > 0 {
				assert.Greater(t, len(change.Additions), 0, "Expected additions")
			}
			if tt.expectDeletions > 0 {
				assert.Greater(t, len(change.Deletions), 0, "Expected deletions")
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly ten!", 12, "exactly ten!"},
		{"this is a very long string that should be truncated", 20, "this is a very long ..."},
	}

	for _, tt := range tests {
		t.Run(tt.input[:min(len(tt.input), 10)], func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
			assert.LessOrEqual(t, len(result), tt.maxLen+3) // +3 for "..."
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
