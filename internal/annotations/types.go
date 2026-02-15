package annotations

import (
	"encoding/json"
	"time"
)

// AnnotationSource identifies the document format the annotation came from.
type AnnotationSource string

const (
	SourcePDF  AnnotationSource = "pdf"
	SourceDOCX AnnotationSource = "docx" // Future: tracked changes
)

// FeedbackCategory normalizes raw annotation types into revision-oriented categories.
type FeedbackCategory string

const (
	CategoryCorrection FeedbackCategory = "correction"
	CategorySuggestion FeedbackCategory = "suggestion"
	CategoryQuestion   FeedbackCategory = "question"
	CategoryApproval   FeedbackCategory = "approval"
	CategoryDeletion   FeedbackCategory = "deletion"
	CategoryInsertion  FeedbackCategory = "insertion"
	CategoryComment    FeedbackCategory = "comment"
	CategoryEmphasis   FeedbackCategory = "emphasis"
)

// ExtractedAnnotation is a single annotation extracted from a document.
type ExtractedAnnotation struct {
	ID              string           `json:"id"`
	Source          AnnotationSource `json:"source"`
	Page            int              `json:"page"`
	RawType         string           `json:"raw_type"`
	Category        FeedbackCategory `json:"category"`
	AnnotationText  string           `json:"annotation_text,omitempty"`
	AnchorText      string           `json:"anchor_text,omitempty"`
	SurroundingText string           `json:"surrounding_text,omitempty"`
	Author          string           `json:"author,omitempty"`
	Date            *time.Time       `json:"date,omitempty"`
	RectString      string           `json:"rect,omitempty"`
	Priority        string           `json:"priority"`
}

// AnnotationSet holds all annotations from a single document extraction.
type AnnotationSet struct {
	SourceFile  string                `json:"source_file"`
	SourceType  AnnotationSource      `json:"source_type"`
	PageCount   int                   `json:"page_count"`
	Annotations []ExtractedAnnotation `json:"annotations"`
	Summary     AnnotationSummary     `json:"summary"`
	OCRText     map[int]string        `json:"ocr_text,omitempty"`
	Errors      []string              `json:"errors,omitempty"`
}

// AnnotationSummary provides aggregate statistics about the annotation set.
type AnnotationSummary struct {
	TotalCount     int            `json:"total_count"`
	ByCategory     map[string]int `json:"by_category"`
	ByAuthor       map[string]int `json:"by_author"`
	PagesAnnotated int            `json:"pages_annotated"`
	HasMarkupAnnots bool          `json:"has_markup_annots"`
}

// ExtractionOptions configures annotation extraction behaviour.
type ExtractionOptions struct {
	IncludeLinks  bool     `json:"include_links"`
	SelectedPages []string `json:"selected_pages,omitempty"`
	RunOCR        bool     `json:"run_ocr"`
	ContextChars  int      `json:"context_chars"`
}

// DefaultOptions returns ExtractionOptions with sensible defaults.
func DefaultOptions() *ExtractionOptions {
	return &ExtractionOptions{
		IncludeLinks: false,
		RunOCR:       false,
		ContextChars: 200,
	}
}

// BuildSummary computes summary statistics from the annotations in the set.
func (as *AnnotationSet) BuildSummary() {
	as.Summary = AnnotationSummary{
		TotalCount: len(as.Annotations),
		ByCategory: make(map[string]int),
		ByAuthor:   make(map[string]int),
	}

	pages := make(map[int]bool)
	for _, a := range as.Annotations {
		as.Summary.ByCategory[string(a.Category)]++
		if a.Author != "" {
			as.Summary.ByAuthor[a.Author]++
		}
		pages[a.Page] = true
		if a.Category == CategoryEmphasis || a.Category == CategoryDeletion ||
			a.Category == CategoryCorrection || a.Category == CategoryInsertion {
			as.Summary.HasMarkupAnnots = true
		}
	}
	as.Summary.PagesAnnotated = len(pages)
}

// ToJSON serializes the AnnotationSet to indented JSON.
func (as *AnnotationSet) ToJSON() ([]byte, error) {
	return json.MarshalIndent(as, "", "  ")
}
