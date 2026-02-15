package annotations

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"go.uber.org/zap"
)

// PDFExtractor extracts annotations from PDF files using pdfcpu.
type PDFExtractor struct {
	logger *zap.Logger
}

// NewPDFExtractor creates a new PDF annotation extractor.
func NewPDFExtractor(logger *zap.Logger) *PDFExtractor {
	return &PDFExtractor{logger: logger}
}

// ExtractFromFile extracts annotations from a PDF file.
func (e *PDFExtractor) ExtractFromFile(filePath string, opts *ExtractionOptions) (*AnnotationSet, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open PDF: %w", err)
	}
	defer f.Close()

	e.logger.Info("Extracting annotations from PDF",
		zap.String("file", filepath.Base(filePath)))

	pgAnnots, err := api.Annotations(f, opts.SelectedPages, nil)
	if err != nil {
		return nil, fmt.Errorf("pdfcpu annotations: %w", err)
	}

	// Count pages from the PDF info
	pageCount := 0
	if len(pgAnnots) > 0 {
		for pg := range pgAnnots {
			if pg > pageCount {
				pageCount = pg
			}
		}
	}

	result := &AnnotationSet{
		SourceFile: filepath.Base(filePath),
		SourceType: SourcePDF,
		PageCount:  pageCount,
	}

	for pageNr, annots := range pgAnnots {
		pageErrors := e.extractPage(pageNr, annots, opts, result)
		for _, pe := range pageErrors {
			result.Errors = append(result.Errors, pe)
		}
	}

	result.BuildSummary()

	e.logger.Info("Annotation extraction complete",
		zap.Int("total", result.Summary.TotalCount),
		zap.Int("pages_annotated", result.Summary.PagesAnnotated))

	return result, nil
}

// extractPage processes all annotations on a single page. Returns per-page errors
// without aborting the whole extraction.
func (e *PDFExtractor) extractPage(pageNr int, pgAnnots model.PgAnnots, opts *ExtractionOptions, result *AnnotationSet) []string {
	var errors []string

	for annType, annot := range pgAnnots {
		// Filter out non-review annotation types by default
		if shouldSkip(annType, opts) {
			continue
		}

		for _, renderer := range annot.Map {
			ea, err := e.convertAnnotation(pageNr, annType, renderer)
			if err != nil {
				errors = append(errors, fmt.Sprintf("page %d: %v", pageNr, err))
				continue
			}
			if ea != nil {
				result.Annotations = append(result.Annotations, *ea)
			}
		}
	}
	return errors
}

// convertAnnotation converts a pdfcpu AnnotationRenderer into an ExtractedAnnotation.
func (e *PDFExtractor) convertAnnotation(pageNr int, annType model.AnnotationType, renderer model.AnnotationRenderer) (*ExtractedAnnotation, error) {
	rawType := model.AnnotTypeStrings[annType]
	if rawType == "" {
		rawType = fmt.Sprintf("Unknown(%d)", int(annType))
	}

	content := stripQuotes(renderer.ContentString())

	ea := &ExtractedAnnotation{
		ID:             renderer.ID(),
		Source:         SourcePDF,
		Page:           pageNr,
		RawType:        rawType,
		Category:       mapCategory(annType, content),
		AnnotationText: content,
		RectString:     renderer.RectString(),
		Priority:       mapPriority(annType),
	}

	// Extract author and date from markup annotations via type assertion.
	// pdfcpu's concrete types embed MarkupAnnotation which has T (author) and CreationDate.
	e.extractMarkupFields(annType, renderer, ea)

	return ea, nil
}

// extractMarkupFields attempts to type-assert the renderer to concrete annotation
// types that carry author/date fields from MarkupAnnotation.
func (e *PDFExtractor) extractMarkupFields(annType model.AnnotationType, renderer model.AnnotationRenderer, ea *ExtractedAnnotation) {
	// pdfcpu returns annotations as value types (not pointers), so we try both.
	switch annType {
	case model.AnnHighLight:
		if a, ok := renderer.(model.HighlightAnnotation); ok {
			ea.Author = a.T
			ea.Date = parseDate(a.CreationDate)
		} else if a, ok := renderer.(*model.HighlightAnnotation); ok {
			ea.Author = a.T
			ea.Date = parseDate(a.CreationDate)
		}
	case model.AnnUnderline:
		if a, ok := renderer.(model.UnderlineAnnotation); ok {
			ea.Author = a.T
			ea.Date = parseDate(a.CreationDate)
		} else if a, ok := renderer.(*model.UnderlineAnnotation); ok {
			ea.Author = a.T
			ea.Date = parseDate(a.CreationDate)
		}
	case model.AnnStrikeOut:
		if a, ok := renderer.(model.StrikeOutAnnotation); ok {
			ea.Author = a.T
			ea.Date = parseDate(a.CreationDate)
		} else if a, ok := renderer.(*model.StrikeOutAnnotation); ok {
			ea.Author = a.T
			ea.Date = parseDate(a.CreationDate)
		}
	case model.AnnSquiggly:
		if a, ok := renderer.(model.SquigglyAnnotation); ok {
			ea.Author = a.T
			ea.Date = parseDate(a.CreationDate)
		} else if a, ok := renderer.(*model.SquigglyAnnotation); ok {
			ea.Author = a.T
			ea.Date = parseDate(a.CreationDate)
		}
	case model.AnnText:
		if a, ok := renderer.(model.TextAnnotation); ok {
			ea.Author = a.T
			ea.Date = parseDate(a.CreationDate)
		} else if a, ok := renderer.(*model.TextAnnotation); ok {
			ea.Author = a.T
			ea.Date = parseDate(a.CreationDate)
		}
	case model.AnnFreeText:
		if a, ok := renderer.(model.FreeTextAnnotation); ok {
			ea.Author = a.T
			ea.Date = parseDate(a.CreationDate)
		} else if a, ok := renderer.(*model.FreeTextAnnotation); ok {
			ea.Author = a.T
			ea.Date = parseDate(a.CreationDate)
		}
	case model.AnnInk:
		if a, ok := renderer.(model.InkAnnotation); ok {
			ea.Author = a.T
			ea.Date = parseDate(a.CreationDate)
		} else if a, ok := renderer.(*model.InkAnnotation); ok {
			ea.Author = a.T
			ea.Date = parseDate(a.CreationDate)
		}
	case model.AnnCaret:
		if a, ok := renderer.(model.CaretAnnotation); ok {
			ea.Author = a.T
			ea.Date = parseDate(a.CreationDate)
		} else if a, ok := renderer.(*model.CaretAnnotation); ok {
			ea.Author = a.T
			ea.Date = parseDate(a.CreationDate)
		}
	// AnnStamp and AnnRedact don't have exported concrete types in pdfcpu v0.11.1,
	// so we can only access AnnotationRenderer interface methods for those.
	}
}

// shouldSkip returns true for annotation types that are typically not
// reviewer feedback (links, widgets, popups, etc.).
func shouldSkip(annType model.AnnotationType, opts *ExtractionOptions) bool {
	switch annType {
	case model.AnnPopup, model.AnnWidget, model.AnnPrinterMark,
		model.AnnTrapNet, model.AnnWatermark, model.Ann3D,
		model.AnnSound, model.AnnMovie, model.AnnScreen,
		model.AnnFileAttachment:
		return true
	case model.AnnLink:
		return !opts.IncludeLinks
	}
	return false
}

// mapCategory converts a pdfcpu annotation type to a FeedbackCategory.
func mapCategory(annType model.AnnotationType, content string) FeedbackCategory {
	switch annType {
	case model.AnnText:
		// Sticky notes: question if content ends with "?"
		if strings.HasSuffix(strings.TrimSpace(content), "?") {
			return CategoryQuestion
		}
		return CategoryComment
	case model.AnnHighLight:
		return CategoryEmphasis
	case model.AnnStrikeOut:
		return CategoryDeletion
	case model.AnnSquiggly:
		return CategoryCorrection
	case model.AnnFreeText:
		return CategorySuggestion
	case model.AnnCaret:
		return CategoryInsertion
	case model.AnnRedact:
		return CategoryDeletion
	case model.AnnStamp:
		return CategoryApproval
	case model.AnnUnderline:
		return CategoryEmphasis
	case model.AnnInk:
		return CategoryComment
	case model.AnnLine, model.AnnSquare, model.AnnCircle,
		model.AnnPolygon, model.AnnPolyLine:
		return CategoryComment
	case model.AnnLink:
		return CategoryComment
	default:
		return CategoryComment
	}
}

// mapPriority assigns a priority based on annotation type.
func mapPriority(annType model.AnnotationType) string {
	switch annType {
	case model.AnnStrikeOut, model.AnnRedact:
		return "high"
	case model.AnnSquiggly, model.AnnCaret, model.AnnFreeText:
		return "medium"
	default:
		return "low"
	}
}

// stripQuotes removes surrounding double quotes from a string.
// pdfcpu's ContentString() wraps content in quotes like "\"text\"".
func stripQuotes(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// parseDate tries to parse a PDF date string (D:YYYYMMDDHHmmSS+HH'mm' format).
func parseDate(s string) *time.Time {
	if s == "" {
		return nil
	}

	// Strip the "D:" prefix if present
	s = strings.TrimPrefix(s, "D:")

	// PDF date formats to try, from most to least specific
	formats := []string{
		"20060102150405-07'00'",
		"20060102150405+07'00'",
		"20060102150405Z",
		"20060102150405",
		"200601021504",
		"2006010215",
		"20060102",
	}

	for _, layout := range formats {
		if t, err := time.Parse(layout, s); err == nil {
			return &t
		}
	}
	return nil
}
