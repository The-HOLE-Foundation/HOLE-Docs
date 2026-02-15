package annotations

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleAnnotationSet() *AnnotationSet {
	set := &AnnotationSet{
		SourceFile: "test-document.pdf",
		SourceType: SourcePDF,
		PageCount:  3,
		Annotations: []ExtractedAnnotation{
			{
				ID:             "a1",
				Source:         SourcePDF,
				Page:           1,
				RawType:        "Text",
				Category:       CategoryComment,
				AnnotationText: "This paragraph needs a citation",
				Author:         "Dr. Smith",
				RectString:     "[72.00, 700.00, 92.00, 720.00]",
				Priority:       "low",
			},
			{
				ID:         "a2",
				Source:     SourcePDF,
				Page:       1,
				RawType:    "Highlight",
				Category:   CategoryEmphasis,
				Author:     "Reviewer A",
				RectString: "[100.00, 600.00, 500.00, 615.00]",
				Priority:   "low",
			},
			{
				ID:             "a3",
				Source:         SourcePDF,
				Page:           2,
				RawType:        "StrikeOut",
				Category:       CategoryDeletion,
				AnnotationText: "Remove this sentence",
				Author:         "Dr. Smith",
				RectString:     "[100.00, 550.00, 400.00, 565.00]",
				Priority:       "high",
			},
		},
	}
	set.BuildSummary()
	return set
}

func emptyAnnotationSet() *AnnotationSet {
	set := &AnnotationSet{
		SourceFile: "empty.pdf",
		SourceType: SourcePDF,
	}
	set.BuildSummary()
	return set
}

func TestFormatLLM_Structure(t *testing.T) {
	output := FormatForLLM(sampleAnnotationSet())

	assert.Contains(t, output, "## Revision Instructions")
	assert.Contains(t, output, "test-document.pdf")
	assert.Contains(t, output, "3 annotations")
	assert.Contains(t, output, "### Summary")
	assert.Contains(t, output, "### Page 1")
	assert.Contains(t, output, "### Page 2")
	assert.Contains(t, output, "COMMENT")
	assert.Contains(t, output, "EMPHASIS")
	assert.Contains(t, output, "DELETION")
	assert.Contains(t, output, "Dr. Smith")
}

func TestFormatLLM_WithOCR(t *testing.T) {
	set := sampleAnnotationSet()
	set.OCRText = map[int]string{
		1: "This is the OCR text for page 1 with visible annotations.",
	}

	output := FormatForLLM(set)
	assert.Contains(t, output, "Full Text (from OCR)")
	assert.Contains(t, output, "visible annotations")
}

func TestFormatLLM_Empty(t *testing.T) {
	output := FormatForLLM(emptyAnnotationSet())
	assert.Contains(t, output, "No annotations found")
}

func TestFormatLLM_Nil(t *testing.T) {
	output := FormatForLLM(nil)
	assert.Contains(t, output, "No annotations found")
}

func TestFormatJSON_Roundtrip(t *testing.T) {
	set := sampleAnnotationSet()
	data, err := FormatAsJSON(set)
	require.NoError(t, err)

	var decoded AnnotationSet
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, set.SourceFile, decoded.SourceFile)
	assert.Equal(t, len(set.Annotations), len(decoded.Annotations))
	assert.Equal(t, set.Summary.TotalCount, decoded.Summary.TotalCount)
}

func TestFormatJSON_ValidJSON(t *testing.T) {
	data, err := FormatAsJSON(sampleAnnotationSet())
	require.NoError(t, err)
	assert.True(t, json.Valid(data), "output should be valid JSON")
}

func TestFormatMarkdown_Structure(t *testing.T) {
	output := FormatAsMarkdown(sampleAnnotationSet())

	assert.True(t, strings.HasPrefix(output, "# Annotations:"))
	assert.Contains(t, output, "| Category | Count |")
	assert.Contains(t, output, "## Page 1")
	assert.Contains(t, output, "## Page 2")
	assert.Contains(t, output, "**Text**")
	assert.Contains(t, output, "*by Dr. Smith*")
}

func TestFormatMarkdown_Empty(t *testing.T) {
	output := FormatAsMarkdown(emptyAnnotationSet())
	assert.Contains(t, output, "No annotations found")
}

func TestFormatChat_Compact(t *testing.T) {
	output := FormatAsChatContext(sampleAnnotationSet())

	assert.True(t, strings.HasPrefix(output, "ANNOTATIONS: 3 total"))
	assert.Contains(t, output, "p1 COMMENT")
	assert.Contains(t, output, "p1 EMPHASIS")
	assert.Contains(t, output, "p2 DELETION")
	assert.Contains(t, output, "[Dr. Smith]")

	// Should be compact — fewer lines than LLM format
	chatLines := strings.Count(output, "\n")
	llmLines := strings.Count(FormatForLLM(sampleAnnotationSet()), "\n")
	assert.Less(t, chatLines, llmLines, "chat format should be more compact than LLM format")
}

func TestFormatChat_Truncation(t *testing.T) {
	set := &AnnotationSet{
		SourceFile: "test.pdf",
		SourceType: SourcePDF,
		Annotations: []ExtractedAnnotation{
			{
				ID:             "long",
				Source:         SourcePDF,
				Page:           1,
				RawType:        "Text",
				Category:       CategoryComment,
				AnnotationText: strings.Repeat("a", 200), // Very long text
				Priority:       "low",
			},
		},
	}
	set.BuildSummary()

	output := FormatAsChatContext(set)
	// Should be truncated with "..."
	assert.Contains(t, output, "...")
}

func TestFormatChat_Empty(t *testing.T) {
	output := FormatAsChatContext(emptyAnnotationSet())
	assert.Equal(t, "ANNOTATIONS: None\n", output)
}

func TestFormatChat_Nil(t *testing.T) {
	output := FormatAsChatContext(nil)
	assert.Equal(t, "ANNOTATIONS: None\n", output)
}

func TestFormat_SingleAnnotation(t *testing.T) {
	set := &AnnotationSet{
		SourceFile: "single.pdf",
		SourceType: SourcePDF,
		PageCount:  1,
		Annotations: []ExtractedAnnotation{
			{
				ID:             "only",
				Source:         SourcePDF,
				Page:           1,
				RawType:        "Text",
				Category:       CategoryComment,
				AnnotationText: "Only annotation",
				Priority:       "low",
			},
		},
	}
	set.BuildSummary()

	// All formats should work with single annotation
	llm := FormatForLLM(set)
	assert.Contains(t, llm, "1 annotations")
	assert.Contains(t, llm, "Only annotation")

	md := FormatAsMarkdown(set)
	assert.Contains(t, md, "Only annotation")

	chat := FormatAsChatContext(set)
	assert.Contains(t, chat, "1 total")

	data, err := FormatAsJSON(set)
	require.NoError(t, err)
	assert.True(t, json.Valid(data))
}
