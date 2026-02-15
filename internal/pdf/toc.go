package pdf

import (
	"fmt"

	"github.com/The-HOLE-Foundation/hole-docs/internal/docling"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

// TOCGenerator generates table of contents from document structure
type TOCGenerator struct {
	processor *Processor
}

// NewTOCGenerator creates a new TOC generator
func NewTOCGenerator(processor *Processor) *TOCGenerator {
	return &TOCGenerator{
		processor: processor,
	}
}

// ExtractTOC extracts table of contents from Docling processing result
func (tg *TOCGenerator) ExtractTOC(result *docling.ProcessingResult) ([]pdfcpu.Bookmark, error) {
	if result == nil || result.Structure.Elements == nil {
		return nil, fmt.Errorf("invalid processing result")
	}

	// Prefer pre-built TOC field if available
	var elements []docling.DocumentElement
	if len(result.Structure.TOC) > 0 {
		elements = result.Structure.TOC
	} else {
		// Extract headings from Elements
		elements = tg.filterHeadings(result.Structure.Elements)
	}

	if len(elements) == 0 {
		return []pdfcpu.Bookmark{}, nil // No headings found
	}

	// Convert Docling elements to pdfcpu bookmarks
	bookmarks, err := tg.convertElements(elements)
	if err != nil {
		return nil, fmt.Errorf("failed to convert elements: %w", err)
	}

	return bookmarks, nil
}

// filterHeadings extracts only heading elements from mixed content
// Preserves hierarchical structure by keeping Children intact
func (tg *TOCGenerator) filterHeadings(elements []docling.DocumentElement) []docling.DocumentElement {
	var headings []docling.DocumentElement

	for _, elem := range elements {
		if elem.Type == "heading" {
			// Filter children recursively but keep them as children
			filtered := elem
			if len(elem.Children) > 0 {
				filtered.Children = tg.filterHeadings(elem.Children)
			}
			headings = append(headings, filtered)
		} else {
			// Not a heading, but check children for headings
			if len(elem.Children) > 0 {
				childHeadings := tg.filterHeadings(elem.Children)
				headings = append(headings, childHeadings...)
			}
		}
	}

	return headings
}

// convertElements converts Docling document elements to pdfcpu bookmarks
func (tg *TOCGenerator) convertElements(elements []docling.DocumentElement) ([]pdfcpu.Bookmark, error) {
	// First, check if elements already have hierarchical structure (Children field populated)
	hasHierarchy := false
	for _, elem := range elements {
		if len(elem.Children) > 0 {
			hasHierarchy = true
			break
		}
	}

	if hasHierarchy {
		// Elements are already hierarchical, convert directly
		return tg.convertHierarchical(elements)
	}

	// Elements are flat, need to build hierarchy from levels
	return tg.convertFlat(elements)
}

// convertHierarchical converts elements that already have Children populated
func (tg *TOCGenerator) convertHierarchical(elements []docling.DocumentElement) ([]pdfcpu.Bookmark, error) {
	var bookmarks []pdfcpu.Bookmark

	for _, elem := range elements {
		if elem.Type != "heading" {
			continue
		}

		pageNr := 1
		if elem.BBox != nil && elem.BBox.PageNumber > 0 {
			pageNr = elem.BBox.PageNumber
		}

		bookmark := pdfcpu.Bookmark{
			Title:    elem.Content,
			PageFrom: pageNr,
		}

		// Recursively convert children
		if len(elem.Children) > 0 {
			childBookmarks, err := tg.convertHierarchical(elem.Children)
			if err != nil {
				return nil, err
			}
			if len(childBookmarks) > 0 {
				bookmark.Kids = childBookmarks
			}
		}

		bookmarks = append(bookmarks, bookmark)
	}

	return bookmarks, nil
}

// convertFlat converts flat list of heading elements to hierarchical bookmarks
func (tg *TOCGenerator) convertFlat(elements []docling.DocumentElement) ([]pdfcpu.Bookmark, error) {
	if len(elements) == 0 {
		return []pdfcpu.Bookmark{}, nil
	}

	// Build hierarchy based on heading levels
	type stackEntry struct {
		bookmark *pdfcpu.Bookmark
		level    int
	}

	var result []pdfcpu.Bookmark
	var stack []stackEntry

	for _, elem := range elements {
		if elem.Type != "heading" {
			continue
		}

		pageNr := 1
		if elem.BBox != nil && elem.BBox.PageNumber > 0 {
			pageNr = elem.BBox.PageNumber
		}

		bookmark := pdfcpu.Bookmark{
			Title:    elem.Content,
			PageFrom: pageNr,
		}

		// Pop stack until we find appropriate parent
		for len(stack) > 0 && stack[len(stack)-1].level >= elem.Level {
			stack = stack[:len(stack)-1]
		}

		if len(stack) == 0 {
			// Top-level bookmark
			result = append(result, bookmark)
			stack = append(stack, stackEntry{
				bookmark: &result[len(result)-1],
				level:    elem.Level,
			})
		} else {
			// Child bookmark
			parent := stack[len(stack)-1].bookmark
			parent.Kids = append(parent.Kids, bookmark)
			childRef := &parent.Kids[len(parent.Kids)-1]
			stack = append(stack, stackEntry{
				bookmark: childRef,
				level:    elem.Level,
			})
		}
	}

	return result, nil
}


// GenerateTOC generates a table of contents and inserts it into a PDF
func (tg *TOCGenerator) GenerateTOC(inputPath, outputPath string, result *docling.ProcessingResult, replace bool) error {
	// Extract bookmarks from Docling result
	bookmarks, err := tg.ExtractTOC(result)
	if err != nil {
		return fmt.Errorf("failed to extract TOC: %w", err)
	}

	if len(bookmarks) == 0 {
		return fmt.Errorf("no headings found in document")
	}

	// Remove existing bookmarks if replacing
	if replace {
		// Note: pdfcpu doesn't have a direct "replace" - we remove then add
		// For now, we'll just add bookmarks (existing ones will remain)
		// TODO: Implement bookmark removal if needed
	}

	// Add bookmarks to PDF
	// Note: api.AddBookmarksFile appends bookmarks to existing ones
	err = api.AddBookmarksFile(inputPath, outputPath, bookmarks, false, nil)
	if err != nil {
		return fmt.Errorf("failed to add bookmarks: %w", err)
	}

	return nil
}
