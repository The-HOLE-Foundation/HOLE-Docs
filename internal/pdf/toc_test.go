package pdf

import (
	"testing"

	"github.com/The-HOLE-Foundation/hole-docs/internal/docling"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExtractTOC_SingleHeading tests extracting a single top-level heading
func TestExtractTOC_SingleHeading(t *testing.T) {
	// Arrange: Create a mock Docling result with one H1 heading
	result := &docling.ProcessingResult{
		Structure: docling.DocumentStructure{
			Elements: []docling.DocumentElement{
				{
					Type:    "heading",
					Level:   1,
					Content: "Chapter 1: Introduction",
					BBox:    &docling.BoundingBox{PageNumber: 1},
				},
			},
		},
	}

	// Act: Extract TOC
	tocGen := NewTOCGenerator(nil) // nil processor for unit test
	bookmarks, err := tocGen.ExtractTOC(result)

	// Assert: Expect 1 bookmark at top level
	require.NoError(t, err)
	require.Len(t, bookmarks, 1)
	assert.Equal(t, "Chapter 1: Introduction", bookmarks[0].Title)
	assert.Equal(t, 1, bookmarks[0].PageFrom)
	assert.Nil(t, bookmarks[0].Kids) // No children
}

// TestExtractTOC_MultipleHeadings tests extracting multiple H1 headings
func TestExtractTOC_MultipleHeadings(t *testing.T) {
	// Arrange: Multiple H1 headings at different pages
	result := &docling.ProcessingResult{
		Structure: docling.DocumentStructure{
			Elements: []docling.DocumentElement{
				{
					Type:    "heading",
					Level:   1,
					Content: "Chapter 1",
					BBox:    &docling.BoundingBox{PageNumber: 1},
				},
				{
					Type:    "heading",
					Level:   1,
					Content: "Chapter 2",
					BBox:    &docling.BoundingBox{PageNumber: 10},
				},
				{
					Type:    "heading",
					Level:   1,
					Content: "Chapter 3",
					BBox:    &docling.BoundingBox{PageNumber: 20},
				},
			},
		},
	}

	// Act
	tocGen := NewTOCGenerator(nil)
	bookmarks, err := tocGen.ExtractTOC(result)

	// Assert: 3 top-level bookmarks
	require.NoError(t, err)
	require.Len(t, bookmarks, 3)

	assert.Equal(t, "Chapter 1", bookmarks[0].Title)
	assert.Equal(t, 1, bookmarks[0].PageFrom)

	assert.Equal(t, "Chapter 2", bookmarks[1].Title)
	assert.Equal(t, 10, bookmarks[1].PageFrom)

	assert.Equal(t, "Chapter 3", bookmarks[2].Title)
	assert.Equal(t, 20, bookmarks[2].PageFrom)
}

// TestExtractTOC_NestedHeadings tests hierarchical heading structure (H1 > H2 > H3)
func TestExtractTOC_NestedHeadings(t *testing.T) {
	// Arrange: H1 with H2 children
	result := &docling.ProcessingResult{
		Structure: docling.DocumentStructure{
			Elements: []docling.DocumentElement{
				{
					Type:    "heading",
					Level:   1,
					Content: "Chapter 1",
					BBox:    &docling.BoundingBox{PageNumber: 1},
					Children: []docling.DocumentElement{
						{
							Type:    "heading",
							Level:   2,
							Content: "Section 1.1",
							BBox:    &docling.BoundingBox{PageNumber: 3},
						},
						{
							Type:    "heading",
							Level:   2,
							Content: "Section 1.2",
							BBox:    &docling.BoundingBox{PageNumber: 5},
						},
					},
				},
			},
		},
	}

	// Act
	tocGen := NewTOCGenerator(nil)
	bookmarks, err := tocGen.ExtractTOC(result)

	// Assert: 1 top-level bookmark with 2 children
	require.NoError(t, err)
	require.Len(t, bookmarks, 1)

	assert.Equal(t, "Chapter 1", bookmarks[0].Title)
	assert.Equal(t, 1, bookmarks[0].PageFrom)

	// Check nested structure
	require.NotNil(t, bookmarks[0].Kids)
	require.Len(t, bookmarks[0].Kids, 2)

	assert.Equal(t, "Section 1.1", bookmarks[0].Kids[0].Title)
	assert.Equal(t, 3, bookmarks[0].Kids[0].PageFrom)

	assert.Equal(t, "Section 1.2", bookmarks[0].Kids[1].Title)
	assert.Equal(t, 5, bookmarks[0].Kids[1].PageFrom)
}

// TestExtractTOC_DeepNesting tests H1 > H2 > H3 hierarchy
func TestExtractTOC_DeepNesting(t *testing.T) {
	// Arrange: 3-level hierarchy
	result := &docling.ProcessingResult{
		Structure: docling.DocumentStructure{
			Elements: []docling.DocumentElement{
				{
					Type:    "heading",
					Level:   1,
					Content: "Part I",
					BBox:    &docling.BoundingBox{PageNumber: 1},
					Children: []docling.DocumentElement{
						{
							Type:    "heading",
							Level:   2,
							Content: "Chapter 1",
							BBox:    &docling.BoundingBox{PageNumber: 2},
							Children: []docling.DocumentElement{
								{
									Type:    "heading",
									Level:   3,
									Content: "Section 1.1",
									BBox:    &docling.BoundingBox{PageNumber: 5},
								},
							},
						},
					},
				},
			},
		},
	}

	// Act
	tocGen := NewTOCGenerator(nil)
	bookmarks, err := tocGen.ExtractTOC(result)

	// Assert: Check 3-level structure
	require.NoError(t, err)
	require.Len(t, bookmarks, 1)

	// Level 1 (H1)
	assert.Equal(t, "Part I", bookmarks[0].Title)
	require.NotNil(t, bookmarks[0].Kids)
	require.Len(t, bookmarks[0].Kids, 1)

	// Level 2 (H2)
	chapter := bookmarks[0].Kids[0]
	assert.Equal(t, "Chapter 1", chapter.Title)
	require.NotNil(t, chapter.Kids)
	require.Len(t, chapter.Kids, 1)

	// Level 3 (H3)
	section := chapter.Kids[0]
	assert.Equal(t, "Section 1.1", section.Title)
}

// TestExtractTOC_NoHeadings tests document with no heading elements
func TestExtractTOC_NoHeadings(t *testing.T) {
	// Arrange: Document with paragraphs and tables, but no headings
	result := &docling.ProcessingResult{
		Structure: docling.DocumentStructure{
			Elements: []docling.DocumentElement{
				{Type: "paragraph", Content: "Some text"},
				{Type: "table", Content: "Table data"},
				{Type: "paragraph", Content: "More text"},
			},
		},
	}

	// Act
	tocGen := NewTOCGenerator(nil)
	bookmarks, err := tocGen.ExtractTOC(result)

	// Assert: Empty bookmark list (no headings found)
	require.NoError(t, err)
	assert.Len(t, bookmarks, 0)
}

// TestExtractTOC_MixedContent tests headings mixed with other content
func TestExtractTOC_MixedContent(t *testing.T) {
	// Arrange: Headings interspersed with paragraphs and tables
	result := &docling.ProcessingResult{
		Structure: docling.DocumentStructure{
			Elements: []docling.DocumentElement{
				{Type: "paragraph", Content: "Introduction paragraph"},
				{
					Type:    "heading",
					Level:   1,
					Content: "Chapter 1",
					BBox:    &docling.BoundingBox{PageNumber: 2},
				},
				{Type: "paragraph", Content: "Chapter 1 content"},
				{Type: "table", Content: "Data table"},
				{
					Type:    "heading",
					Level:   2,
					Content: "Section 1.1",
					BBox:    &docling.BoundingBox{PageNumber: 5},
				},
				{Type: "paragraph", Content: "Section content"},
			},
		},
	}

	// Act: Should filter out non-heading elements
	tocGen := NewTOCGenerator(nil)
	bookmarks, err := tocGen.ExtractTOC(result)

	// Assert: Only headings extracted
	require.NoError(t, err)
	require.Len(t, bookmarks, 1) // 1 H1
	assert.Equal(t, "Chapter 1", bookmarks[0].Title)

	// H2 should be child of H1
	require.NotNil(t, bookmarks[0].Kids)
	require.Len(t, bookmarks[0].Kids, 1)
	assert.Equal(t, "Section 1.1", bookmarks[0].Kids[0].Title)
}

// TestExtractTOC_MissingPageNumber tests handling of elements without page numbers
func TestExtractTOC_MissingPageNumber(t *testing.T) {
	// Arrange: Heading without BBox (malformed data)
	result := &docling.ProcessingResult{
		Structure: docling.DocumentStructure{
			Elements: []docling.DocumentElement{
				{
					Type:    "heading",
					Level:   1,
					Content: "Chapter 1",
					BBox:    nil, // Missing!
				},
			},
		},
	}

	// Act
	tocGen := NewTOCGenerator(nil)
	bookmarks, err := tocGen.ExtractTOC(result)

	// Assert: Should default to page 1 or handle gracefully
	require.NoError(t, err)
	require.Len(t, bookmarks, 1)
	assert.Equal(t, 1, bookmarks[0].PageFrom) // Default to page 1
}

// TestExtractTOC_PreferBuiltInTOC tests using pre-built TOC field from Docling
func TestExtractTOC_PreferBuiltInTOC(t *testing.T) {
	// Arrange: Docling result with pre-built TOC field
	result := &docling.ProcessingResult{
		Structure: docling.DocumentStructure{
			TOC: []docling.DocumentElement{
				{
					Type:    "heading",
					Level:   1,
					Content: "Chapter 1",
					BBox:    &docling.BoundingBox{PageNumber: 1},
				},
				{
					Type:    "heading",
					Level:   1,
					Content: "Chapter 2",
					BBox:    &docling.BoundingBox{PageNumber: 10},
				},
			},
			Elements: []docling.DocumentElement{
				// Has more elements, but TOC field should take precedence
			},
		},
	}

	// Act
	tocGen := NewTOCGenerator(nil)
	bookmarks, err := tocGen.ExtractTOC(result)

	// Assert: Should use pre-built TOC field (2 chapters)
	require.NoError(t, err)
	require.Len(t, bookmarks, 2)
	assert.Equal(t, "Chapter 1", bookmarks[0].Title)
	assert.Equal(t, "Chapter 2", bookmarks[1].Title)
}

// TestFilterHeadings tests extracting only heading elements from mixed content
func TestFilterHeadings(t *testing.T) {
	// Arrange: Mixed elements
	elements := []docling.DocumentElement{
		{Type: "paragraph", Content: "Text"},
		{Type: "heading", Level: 1, Content: "H1"},
		{Type: "table", Content: "Data"},
		{Type: "heading", Level: 2, Content: "H2"},
		{Type: "image", Content: "Image"},
		{Type: "heading", Level: 1, Content: "Another H1"},
	}

	// Act
	tocGen := NewTOCGenerator(nil)
	headings := tocGen.filterHeadings(elements)

	// Assert: Only 3 headings extracted
	require.Len(t, headings, 3)
	assert.Equal(t, "H1", headings[0].Content)
	assert.Equal(t, "H2", headings[1].Content)
	assert.Equal(t, "Another H1", headings[2].Content)
}

// TestBuildHierarchy tests converting flat heading list to nested structure
func TestBuildHierarchy(t *testing.T) {
	// Arrange: Flat list of headings with different levels
	headings := []docling.DocumentElement{
		{Type: "heading", Level: 1, Content: "Chapter 1", BBox: &docling.BoundingBox{PageNumber: 1}},
		{Type: "heading", Level: 2, Content: "Section 1.1", BBox: &docling.BoundingBox{PageNumber: 2}},
		{Type: "heading", Level: 2, Content: "Section 1.2", BBox: &docling.BoundingBox{PageNumber: 5}},
		{Type: "heading", Level: 1, Content: "Chapter 2", BBox: &docling.BoundingBox{PageNumber: 10}},
	}

	// Act: Build hierarchy (H2s should become children of H1s)
	tocGen := NewTOCGenerator(nil)
	bookmarks, err := tocGen.convertElements(headings)

	// Assert: 2 top-level bookmarks (H1s), each with correct children
	require.NoError(t, err)
	require.Len(t, bookmarks, 2)

	// Chapter 1 with 2 sections
	assert.Equal(t, "Chapter 1", bookmarks[0].Title)
	require.Len(t, bookmarks[0].Kids, 2)
	assert.Equal(t, "Section 1.1", bookmarks[0].Kids[0].Title)
	assert.Equal(t, "Section 1.2", bookmarks[0].Kids[1].Title)

	// Chapter 2 with no children
	assert.Equal(t, "Chapter 2", bookmarks[1].Title)
	assert.Nil(t, bookmarks[1].Kids)
}

// TestHeadingHierarchy tests that hierarchy is preserved through nesting
func TestHeadingHierarchy(t *testing.T) {
	// Arrange: H1 through H3 hierarchy
	result := &docling.ProcessingResult{
		Structure: docling.DocumentStructure{
			Elements: []docling.DocumentElement{
				{
					Type:    "heading",
					Level:   1,
					Content: "H1 Heading",
					BBox:    &docling.BoundingBox{PageNumber: 1},
				},
				{
					Type:    "heading",
					Level:   2,
					Content: "H2 Heading",
					BBox:    &docling.BoundingBox{PageNumber: 2},
				},
				{
					Type:    "heading",
					Level:   3,
					Content: "H3 Heading",
					BBox:    &docling.BoundingBox{PageNumber: 3},
				},
			},
		},
	}

	// Act
	tocGen := NewTOCGenerator(nil)
	bookmarks, err := tocGen.ExtractTOC(result)

	// Assert: Hierarchy is preserved through nesting
	require.NoError(t, err)
	require.Len(t, bookmarks, 1) // 1 top-level (H1)

	// H1 has H2 as child
	assert.Equal(t, "H1 Heading", bookmarks[0].Title)
	require.Len(t, bookmarks[0].Kids, 1)

	// H2 has H3 as child
	h2 := bookmarks[0].Kids[0]
	assert.Equal(t, "H2 Heading", h2.Title)
	require.Len(t, h2.Kids, 1)

	// H3 has no children
	h3 := h2.Kids[0]
	assert.Equal(t, "H3 Heading", h3.Title)
	assert.Nil(t, h3.Kids)
}
