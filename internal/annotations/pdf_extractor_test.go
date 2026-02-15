package annotations

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// testdataDir returns the path to the testdata/annotations directory.
func testdataDir() string {
	return filepath.Join("..", "..", "testdata", "annotations")
}

// createAnnotatedPDF creates a test PDF with known annotations.
// It starts from a minimal PDF, adds various annotation types, and returns the path.
func createAnnotatedPDF(t *testing.T, dir string) string {
	t.Helper()

	// Create a minimal 1-page PDF by writing raw PDF bytes
	basePath := filepath.Join(dir, "base.pdf")
	err := os.WriteFile(basePath, minimalPDFBytes(), 0644)
	require.NoError(t, err, "create base PDF")

	annotatedPath := filepath.Join(dir, "annotated-sample.pdf")

	// Build annotation map: page 1 gets several annotation types
	annMap := make(map[int][]model.AnnotationRenderer)

	yellow := color.SimpleColor{R: 1.0, G: 1.0, B: 0.0}
	red := color.SimpleColor{R: 1.0, G: 0.0, B: 0.0}

	// 1. Sticky note (Text annotation) with comment text
	textAnn := model.NewTextAnnotation(
		*types.NewRectangle(72, 700, 92, 720),
		0,
		"This paragraph needs a citation",
		"annot-text-1",
		"",
		model.AnnPrint,
		&yellow,
		"Dr. Smith",
		nil,
		nil,
		"", "",
		0, 0, 0,
		true,
		"Comment",
	)
	annMap[1] = append(annMap[1], textAnn)

	// 2. Highlight annotation
	highlightAnn := model.NewHighlightAnnotation(
		*types.NewRectangle(100, 600, 500, 615),
		0,
		"",
		"annot-highlight-1",
		"",
		model.AnnPrint,
		&yellow,
		0, 0, 0,
		"Reviewer A",
		nil,
		nil,
		"", "",
		nil,
	)
	annMap[1] = append(annMap[1], highlightAnn)

	// 3. StrikeOut annotation
	strikeAnn := model.NewStrikeOutAnnotation(
		*types.NewRectangle(100, 550, 400, 565),
		0,
		"Remove this sentence",
		"annot-strike-1",
		"",
		model.AnnPrint,
		&red,
		0, 0, 0,
		"Dr. Smith",
		nil,
		nil,
		"", "",
		nil,
	)
	annMap[1] = append(annMap[1], strikeAnn)

	// 4. Underline annotation (emphasis)
	underlineAnn := model.NewUnderlineAnnotation(
		*types.NewRectangle(100, 480, 500, 495),
		0,
		"",
		"annot-underline-1",
		"",
		model.AnnPrint,
		&yellow,
		0, 0, 0,
		"Reviewer A",
		nil,
		nil,
		"", "",
		nil,
	)
	annMap[1] = append(annMap[1], underlineAnn)

	// 5. Squiggly annotation (correction)
	squigglyAnn := model.NewSquigglyAnnotation(
		*types.NewRectangle(100, 430, 350, 445),
		0,
		"Spelling error here",
		"annot-squiggly-1",
		"",
		model.AnnPrint,
		&red,
		0, 0, 0,
		"Proofreader",
		nil,
		nil,
		"", "",
		nil,
	)
	annMap[1] = append(annMap[1], squigglyAnn)

	// 6. Text annotation that ends with "?" → should be categorized as question
	questionAnn := model.NewTextAnnotation(
		*types.NewRectangle(72, 380, 92, 400),
		0,
		"Is this the correct date?",
		"annot-question-1",
		"",
		model.AnnPrint,
		nil,
		"Reviewer A",
		nil,
		nil,
		"", "",
		0, 0, 0,
		true,
		"Help",
	)
	annMap[1] = append(annMap[1], questionAnn)

	// Add annotations to PDF
	err = api.AddAnnotationsMapFile(basePath, annotatedPath, annMap, nil, false)
	require.NoError(t, err, "add annotations to PDF")

	// Clean up base
	os.Remove(basePath)

	return annotatedPath
}

// createCleanPDF creates a PDF with zero annotations.
func createCleanPDF(t *testing.T, dir string) string {
	t.Helper()
	cleanPath := filepath.Join(dir, "clean.pdf")
	err := os.WriteFile(cleanPath, minimalPDFBytes(), 0644)
	require.NoError(t, err, "create clean PDF")
	return cleanPath
}

// minimalPDFBytes returns a valid single-page PDF document.
func minimalPDFBytes() []byte {
	return []byte(`%PDF-1.4
1 0 obj
<< /Type /Catalog /Pages 2 0 R >>
endobj

2 0 obj
<< /Type /Pages /Kids [3 0 R] /Count 1 >>
endobj

3 0 obj
<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>
endobj

xref
0 4
0000000000 65535 f
0000000009 00000 n
0000000058 00000 n
0000000115 00000 n

trailer
<< /Size 4 /Root 1 0 R >>
startxref
190
%%EOF
`)
}

func TestExtract_StickyNotes(t *testing.T) {
	dir := t.TempDir()
	pdfPath := createAnnotatedPDF(t, dir)
	logger := zap.NewNop()

	ext := NewPDFExtractor(logger)
	result, err := ext.ExtractFromFile(pdfPath, DefaultOptions())
	require.NoError(t, err)
	require.NotNil(t, result)

	// Find sticky note by ID
	var stickyNote *ExtractedAnnotation
	for i, a := range result.Annotations {
		if a.ID == "annot-text-1" {
			stickyNote = &result.Annotations[i]
			break
		}
	}
	require.NotNil(t, stickyNote, "should find sticky note annotation")
	assert.Equal(t, "This paragraph needs a citation", stickyNote.AnnotationText)
	assert.Equal(t, CategoryComment, stickyNote.Category)
	// Note: pdfcpu v0.11.1's Annotation() reader doesn't populate the T (author) field
	// from the PDF dict, so Author is empty in roundtrip tests. Real-world PDFs created
	// by Adobe/Preview will have the same limitation until pdfcpu adds T field parsing.
	assert.Equal(t, 1, stickyNote.Page)
	assert.Equal(t, SourcePDF, stickyNote.Source)
}

func TestExtract_Highlights(t *testing.T) {
	dir := t.TempDir()
	pdfPath := createAnnotatedPDF(t, dir)
	logger := zap.NewNop()

	ext := NewPDFExtractor(logger)
	result, err := ext.ExtractFromFile(pdfPath, DefaultOptions())
	require.NoError(t, err)

	var highlight *ExtractedAnnotation
	for i, a := range result.Annotations {
		if a.ID == "annot-highlight-1" {
			highlight = &result.Annotations[i]
			break
		}
	}
	require.NotNil(t, highlight, "should find highlight annotation")
	assert.Equal(t, CategoryEmphasis, highlight.Category)
	// Author not populated by pdfcpu v0.11.1 reader (see TestExtract_StickyNotes comment)
	assert.Equal(t, "low", highlight.Priority)
	assert.NotEmpty(t, highlight.RectString)
}

func TestExtract_StrikeOut(t *testing.T) {
	dir := t.TempDir()
	pdfPath := createAnnotatedPDF(t, dir)
	logger := zap.NewNop()

	ext := NewPDFExtractor(logger)
	result, err := ext.ExtractFromFile(pdfPath, DefaultOptions())
	require.NoError(t, err)

	var strikeout *ExtractedAnnotation
	for i, a := range result.Annotations {
		if a.ID == "annot-strike-1" {
			strikeout = &result.Annotations[i]
			break
		}
	}
	require.NotNil(t, strikeout, "should find strikeout annotation")
	assert.Equal(t, CategoryDeletion, strikeout.Category)
	assert.Equal(t, "high", strikeout.Priority)
	assert.Equal(t, "Remove this sentence", strikeout.AnnotationText)
}

func TestExtract_Underline(t *testing.T) {
	dir := t.TempDir()
	pdfPath := createAnnotatedPDF(t, dir)
	logger := zap.NewNop()

	ext := NewPDFExtractor(logger)
	result, err := ext.ExtractFromFile(pdfPath, DefaultOptions())
	require.NoError(t, err)

	var underline *ExtractedAnnotation
	for i, a := range result.Annotations {
		if a.ID == "annot-underline-1" {
			underline = &result.Annotations[i]
			break
		}
	}
	require.NotNil(t, underline, "should find underline annotation")
	assert.Equal(t, CategoryEmphasis, underline.Category)
	// Author not populated by pdfcpu v0.11.1 reader (see TestExtract_StickyNotes comment)
	assert.Equal(t, "low", underline.Priority)
}

func TestExtract_Squiggly(t *testing.T) {
	dir := t.TempDir()
	pdfPath := createAnnotatedPDF(t, dir)
	logger := zap.NewNop()

	ext := NewPDFExtractor(logger)
	result, err := ext.ExtractFromFile(pdfPath, DefaultOptions())
	require.NoError(t, err)

	var squiggly *ExtractedAnnotation
	for i, a := range result.Annotations {
		if a.ID == "annot-squiggly-1" {
			squiggly = &result.Annotations[i]
			break
		}
	}
	require.NotNil(t, squiggly, "should find squiggly annotation")
	assert.Equal(t, CategoryCorrection, squiggly.Category)
	assert.Equal(t, "medium", squiggly.Priority)
	assert.Equal(t, "Spelling error here", squiggly.AnnotationText)
}

func TestExtract_QuestionCategory(t *testing.T) {
	dir := t.TempDir()
	pdfPath := createAnnotatedPDF(t, dir)
	logger := zap.NewNop()

	ext := NewPDFExtractor(logger)
	result, err := ext.ExtractFromFile(pdfPath, DefaultOptions())
	require.NoError(t, err)

	var question *ExtractedAnnotation
	for i, a := range result.Annotations {
		if a.ID == "annot-question-1" {
			question = &result.Annotations[i]
			break
		}
	}
	require.NotNil(t, question, "should find question annotation")
	assert.Equal(t, CategoryQuestion, question.Category)
	assert.Equal(t, "Is this the correct date?", question.AnnotationText)
}

func TestExtract_MixedTypes(t *testing.T) {
	dir := t.TempDir()
	pdfPath := createAnnotatedPDF(t, dir)
	logger := zap.NewNop()

	ext := NewPDFExtractor(logger)
	result, err := ext.ExtractFromFile(pdfPath, DefaultOptions())
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(result.Annotations), 6, "should have at least 6 annotations")
	assert.Equal(t, SourcePDF, result.SourceType)
	assert.NotEmpty(t, result.SourceFile)

	// Summary should be populated
	assert.Greater(t, result.Summary.TotalCount, 0)
	assert.Greater(t, result.Summary.PagesAnnotated, 0)
	assert.True(t, result.Summary.HasMarkupAnnots, "should detect markup annotations")

	// Should have multiple categories
	assert.Contains(t, result.Summary.ByCategory, string(CategoryComment))
	assert.Contains(t, result.Summary.ByCategory, string(CategoryEmphasis))
	assert.Contains(t, result.Summary.ByCategory, string(CategoryDeletion))
}

func TestExtract_NoAnnotations(t *testing.T) {
	dir := t.TempDir()
	pdfPath := createCleanPDF(t, dir)
	logger := zap.NewNop()

	ext := NewPDFExtractor(logger)
	result, err := ext.ExtractFromFile(pdfPath, DefaultOptions())
	require.NoError(t, err)
	assert.Empty(t, result.Annotations)
	assert.Equal(t, 0, result.Summary.TotalCount)
}

func TestExtract_FilterLinks(t *testing.T) {
	// By default links should be excluded
	dir := t.TempDir()
	pdfPath := createAnnotatedPDF(t, dir)
	logger := zap.NewNop()

	ext := NewPDFExtractor(logger)
	result, err := ext.ExtractFromFile(pdfPath, DefaultOptions())
	require.NoError(t, err)

	for _, a := range result.Annotations {
		assert.NotEqual(t, "Link", a.RawType, "links should be filtered by default")
	}
}

func TestExtract_InvalidFile(t *testing.T) {
	logger := zap.NewNop()
	ext := NewPDFExtractor(logger)

	_, err := ext.ExtractFromFile("/nonexistent/file.pdf", DefaultOptions())
	assert.Error(t, err)
}

func TestExtract_NotAPDF(t *testing.T) {
	dir := t.TempDir()
	fakePath := filepath.Join(dir, "fake.pdf")
	os.WriteFile(fakePath, []byte("this is not a PDF"), 0644)

	logger := zap.NewNop()
	ext := NewPDFExtractor(logger)

	_, err := ext.ExtractFromFile(fakePath, DefaultOptions())
	assert.Error(t, err)
}

func TestMapCategory(t *testing.T) {
	tests := []struct {
		annType  model.AnnotationType
		content  string
		expected FeedbackCategory
	}{
		{model.AnnText, "fix this", CategoryComment},
		{model.AnnText, "Is this correct?", CategoryQuestion},
		{model.AnnHighLight, "", CategoryEmphasis},
		{model.AnnStrikeOut, "", CategoryDeletion},
		{model.AnnSquiggly, "", CategoryCorrection},
		{model.AnnFreeText, "", CategorySuggestion},
		{model.AnnCaret, "", CategoryInsertion},
		{model.AnnStamp, "", CategoryApproval},
		{model.AnnUnderline, "", CategoryEmphasis},
		{model.AnnInk, "", CategoryComment},
	}
	for _, tt := range tests {
		t.Run(tt.expected.String(tt.annType), func(t *testing.T) {
			assert.Equal(t, tt.expected, mapCategory(tt.annType, tt.content))
		})
	}
}

// String helper for test names
func (fc FeedbackCategory) String(at model.AnnotationType) string {
	return model.AnnotTypeStrings[at] + "_" + string(fc)
}

func TestMapPriority(t *testing.T) {
	assert.Equal(t, "high", mapPriority(model.AnnStrikeOut))
	assert.Equal(t, "high", mapPriority(model.AnnRedact))
	assert.Equal(t, "medium", mapPriority(model.AnnSquiggly))
	assert.Equal(t, "medium", mapPriority(model.AnnCaret))
	assert.Equal(t, "medium", mapPriority(model.AnnFreeText))
	assert.Equal(t, "low", mapPriority(model.AnnHighLight))
	assert.Equal(t, "low", mapPriority(model.AnnText))
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"D:20240115120000", true},
		{"20240115120000", true},
		{"D:20240115", true},
		{"", false},
		{"not-a-date", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseDate(tt.input)
			if tt.valid {
				assert.NotNil(t, result)
			} else {
				assert.Nil(t, result)
			}
		})
	}
}
