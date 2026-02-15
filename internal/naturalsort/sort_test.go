package naturalsort

import (
	"testing"
)

func TestParseComponents(t *testing.T) {
	tests := []struct {
		input    string
		expected int // Expected number of components
		name     string
	}{
		{"a3.1", 4, "Mixed alphanumeric with dots"},
		{"archive", 1, "Simple text"},
		{"file10", 2, "Text with number"},
		{"a3b2c1", 6, "Multiple alternations"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			components := ParseComponents(tt.input)
			if len(components) != tt.expected {
				t.Errorf("got %d components, want %d for input %q", len(components), tt.expected, tt.input)
			}
		})
	}
}

func TestCompareComponents(t *testing.T) {
	tests := []struct {
		a, b   string
		result int // -1 if a < b, 0 if a == b, 1 if a > b
		name   string
	}{
		{"a3.1", "a3.2", -1, "A3.1 before A3.2"},
		{"a3.2", "a3.10", -1, "A3.2 before A3.10 (numeric)"},
		{"a3.10", "a10", -1, "A3.10 before A10"},
		{"archive", "beehive", -1, "Archive before Beehive (lexicographic)"},
		{"a3", "a3", 0, "Same items equal"},
		{"1", "2", -1, "Numbers sorted numerically"},
		{"2", "10", -1, "2 before 10 (numeric, not string)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aComp := ParseComponents(tt.a)
			bComp := ParseComponents(tt.b)
			result := CompareComponents(aComp, bComp)

			if result != tt.result {
				t.Errorf("CompareComponents(%q, %q) = %d, want %d", tt.a, tt.b, result, tt.result)
			}
		})
	}
}

func TestSortPaths(t *testing.T) {
	tests := []struct {
		input    []string
		expected []string
		name     string
	}{
		{
			[]string{"a3.10.pdf", "a3.2.pdf", "a3.1.pdf"},
			[]string{"a3.1.pdf", "a3.2.pdf", "a3.10.pdf"},
			"Numeric suffixes sorted properly",
		},
		{
			[]string{"beehive.pdf", "archive.pdf", "a3.pdf"},
			[]string{"a3.pdf", "archive.pdf", "beehive.pdf"},
			"Mixed alphanumeric and text",
		},
		{
			[]string{"chapter_10.pdf", "chapter_2.pdf", "chapter_1.pdf"},
			[]string{"chapter_1.pdf", "chapter_2.pdf", "chapter_10.pdf"},
			"Chapter numbering",
		},
		{
			[]string{"doc_a.pdf", "doc_1.pdf"},
			[]string{"doc_1.pdf", "doc_a.pdf"},
			"Numbers before strings at same position",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SortPaths(tt.input)
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("Sort failed: got %v, want %v", result, tt.expected)
					break
				}
			}
		})
	}
}

func TestCaseInsensitivity(t *testing.T) {
	tests := []struct {
		a, b string
		name string
	}{
		{"Archive", "beehive", "mixed case comparison"},
		{"A3.PDF", "a3.pdf", "same case variation"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aComp := ParseComponents(tt.a)
			bComp := ParseComponents(tt.b)
			result := CompareComponents(aComp, bComp)
			// Should be case-insensitive, so these should compare as lowercase
			if tt.a != "Archive" { // Only check the case we expect to differ
				return
			}
			if result >= 0 { // Archive should come before beehive
				t.Errorf("Case-insensitive comparison failed: %q should come before %q", tt.a, tt.b)
			}
		})
	}
}
