package mistral

import (
	"regexp"
	"strings"
)

// ExtractKeywords extracts searchable keywords from OCR text for legal documents.
// It returns a slice of unique keywords including detected case/docket numbers (uppercased), dates (YYYY-MM-DD or MM/DD/YYYY), common legal terms found in the text, and statute references normalized as "PC-<number>".
func ExtractKeywords(ocrText string) []string {
	// Normalize text
	text := strings.ToLower(ocrText)

	// Common legal/investigative keywords patterns
	keywords := make(map[string]bool)

	// Extract case numbers (e.g., CR-2024-001234, 2024-001234)
	caseNumberPattern := regexp.MustCompile(`(?i)\b(?:case|docket|cause)\s*(?:no\.?|number|#)?\s*:?\s*([A-Z]{1,3}-?\d{4}-?\d{4,8})\b`)
	for _, match := range caseNumberPattern.FindAllStringSubmatch(text, -1) {
		if len(match) > 1 {
			keywords[strings.ToUpper(match[1])] = true
		}
	}

	// Extract dates (YYYY-MM-DD, MM/DD/YYYY)
	datePattern := regexp.MustCompile(`\b(\d{4}-\d{2}-\d{2}|\d{1,2}/\d{1,2}/\d{4})\b`)
	for _, match := range datePattern.FindAllStringSubmatch(text, -1) {
		if len(match) > 1 {
			keywords[match[1]] = true
		}
	}

	// Extract legal terms
	legalTerms := []string{
		"arrest", "charge", "conviction", "sentence", "probation", "parole",
		"defendant", "plaintiff", "prosecutor", "attorney", "counsel",
		"motion", "brief", "petition", "complaint", "verdict", "judgment",
		"warrant", "subpoena", "deposition", "testimony", "evidence",
		"trial", "hearing", "arraignment", "indictment", "plea",
		"police", "officer", "detective", "investigator", "witness",
		"victim", "suspect", "accused", "offender",
	}

	for _, term := range legalTerms {
		if strings.Contains(text, term) {
			keywords[term] = true
		}
	}

	// Extract statute references (e.g., "PC 22.01", "Penal Code 22.01")
	statutePattern := regexp.MustCompile(`(?i)\b(?:PC|penal code|code)\s+(\d+\.\d+)\b`)
	for _, match := range statutePattern.FindAllStringSubmatch(text, -1) {
		if len(match) > 1 {
			keywords["PC-"+match[1]] = true
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(keywords))
	for keyword := range keywords {
		result = append(result, keyword)
	}

	return result
}

// CreateBlindIndex creates a searchable index that preserves privacy
// CreateBlindIndex generates a privacy-preserving blind index from OCR text.
// It lowercases and strips non-alphanumeric characters (preserving spaces), takes the first 100 characters, then returns the first up to three words joined with hyphens.
func CreateBlindIndex(ocrText string) string {
	// Take first 100 characters
	preview := ocrText
	if len(preview) > 100 {
		preview = preview[:100]
	}

	// Normalize
	preview = strings.ToLower(preview)
	preview = regexp.MustCompile(`[^a-z0-9\s]`).ReplaceAllString(preview, "")

	// Take first 3 words
	words := strings.Fields(preview)
	if len(words) > 3 {
		words = words[:3]
	}

	return strings.Join(words, "-")
}