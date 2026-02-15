package naturalsort

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// Component represents a parsed segment of a filename (either string or int)
type Component struct {
	isNum bool
	numVal int
	strVal string
}

// SortEntry holds a filename and its parsed components
type SortEntry struct {
	Path       string
	Components []Component
}

// Natural regex to extract numeric parts
var numericRegex = regexp.MustCompile(`\d+`)

// ParseComponents breaks a filename into natural components
// E.g., "A3.1_doc.pdf" → [Component{str:"a"}, Component{num:3}, Component{str:"."}, Component{num:1}, Component{str:"_doc.pdf"}]
func ParseComponents(filename string) []Component {
	components := []Component{}
	lower := strings.ToLower(filename)

	i := 0
	for i < len(lower) {
		if unicode.IsDigit(rune(lower[i])) {
			// Extract the full number
			numStart := i
			for i < len(lower) && unicode.IsDigit(rune(lower[i])) {
				i++
			}
			numStr := lower[numStart:i]
			num, _ := strconv.Atoi(numStr)
			components = append(components, Component{isNum: true, numVal: num})
		} else {
			// Extract non-numeric characters until next digit or end
			strStart := i
			for i < len(lower) && !unicode.IsDigit(rune(lower[i])) {
				i++
			}
			components = append(components, Component{isNum: false, strVal: lower[strStart:i]})
		}
	}

	return components
}

// CompareComponents compares two component slices
// Returns: -1 if a < b, 0 if a == b, 1 if a > b
func CompareComponents(a, b []Component) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		aComp := a[i]
		bComp := b[i]

		if aComp.isNum && bComp.isNum {
			// Both numeric - compare numbers
			if aComp.numVal < bComp.numVal {
				return -1
			} else if aComp.numVal > bComp.numVal {
				return 1
			}
		} else if !aComp.isNum && !bComp.isNum {
			// Both strings - compare lexicographically
			if aComp.strVal < bComp.strVal {
				return -1
			} else if aComp.strVal > bComp.strVal {
				return 1
			}
		} else {
			// Mixed: numbers come before strings
			if aComp.isNum {
				return -1 // a is number, b is string
			} else {
				return 1 // a is string, b is number
			}
		}
	}

	// All compared components are equal, shorter comes first
	if len(a) < len(b) {
		return -1
	} else if len(a) > len(b) {
		return 1
	}
	return 0
}

// SortPaths sorts filenames using natural sort order
// Handles: A3.1 < A3.2 < A3.10 < Archive < beehive, etc.
func SortPaths(paths []string) []string {
	entries := make([]SortEntry, len(paths))
	for i, path := range paths {
		entries[i] = SortEntry{
			Path:       path,
			Components: ParseComponents(path),
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return CompareComponents(entries[i].Components, entries[j].Components) < 0
	})

	result := make([]string, len(entries))
	for i, entry := range entries {
		result[i] = entry.Path
	}
	return result
}

// SortPathsWithBasename sorts paths while preserving directory structure
// Only the basename is used for natural sorting, directory prefixes are preserved
func SortPathsWithBasename(paths []string) []string {
	// Map basenames to full paths (handling duplicates)
	type PathEntry struct {
		fullPath string
		basename string
		components []Component
	}

	entries := make([]PathEntry, len(paths))
	for i, path := range paths {
		basename := path
		if idx := strings.LastIndex(path, "/"); idx >= 0 {
			basename = path[idx+1:]
		}
		entries[i] = PathEntry{
			fullPath:   path,
			basename:   basename,
			components: ParseComponents(basename),
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return CompareComponents(entries[i].components, entries[j].components) < 0
	})

	result := make([]string, len(entries))
	for i, entry := range entries {
		result[i] = entry.fullPath
	}
	return result
}
