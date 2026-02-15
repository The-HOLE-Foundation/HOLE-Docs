package docling

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go.uber.org/zap"
)

// Processor handles extracting data from Docling output for storage and processing
type Processor struct {
	client *Client
	logger *zap.Logger
}

// NewProcessor creates and returns a Processor configured with the provided Client and zap.Logger.
func NewProcessor(client *Client, logger *zap.Logger) *Processor {
	return &Processor{
		client: client,
		logger: logger,
	}
}

// DocumentIntelligence represents extracted intelligence from Docling
// Ready for storage in database and downstream processing
type DocumentIntelligence struct {
	// Full outputs (lossless)
	DoclingJSON    json.RawMessage
	DoclingMarkdown string

	// Structured extractions
	Metadata       DocumentMetadata
	Structure      DocumentStructure
	Tables         []Table

	// Normalized content
	TextContent    string  // Plain text for full-text search
	PageCount      int
	Language       string

	// Document classification (for legal work)
	DocumentType   string  // Will be enriched by Mistral if needed
	Keywords       []string

	// Processing metadata
	ProcessingTime int64   // milliseconds
	Confidence     float64 // Overall confidence in extraction (0-1)
}

// Extract processes a document with Docling and extracts all intelligence
func (p *Processor) Extract(ctx context.Context, filePath string) (*DocumentIntelligence, error) {
	p.logger.Info("Extracting document intelligence with Docling",
		zap.String("file_path", filePath))

	// Process document with Docling
	result, err := p.client.ProcessDocument(ctx, filePath)
	if err != nil {
		p.logger.Error("Failed to process document with Docling",
			zap.String("file_path", filePath),
			zap.Error(err))
		return nil, fmt.Errorf("docling processing failed: %w", err)
	}

	// Extract intelligence
	intelligence := &DocumentIntelligence{
		DoclingMarkdown: result.Markdown,
		Metadata:        result.Metadata,
		Structure:       result.Structure,
		Tables:          result.Tables,
		TextContent:     result.RawText,
		PageCount:       result.PageCount,
		Language:        result.Language,
		ProcessingTime:  result.ProcessingTime,
		Confidence:      0.95, // Docling is highly confident by default
		DoclingJSON:     result.RawJSON,
	}

	// Extract keywords from structure
	intelligence.Keywords = extractKeywordsFromStructure(result.Structure)

	// Try to auto-detect document type (can be overridden by Mistral later)
	intelligence.DocumentType = detectDocumentType(result)

	p.logger.Info("Document intelligence extracted successfully",
		zap.String("file_path", filePath),
		zap.String("language", intelligence.Language),
		zap.Int("pages", intelligence.PageCount),
		zap.Int("keywords", len(intelligence.Keywords)),
		zap.String("detected_type", intelligence.DocumentType))

	return intelligence, nil
}

// ExtractPDF is convenience method for PDF-specific processing
func (p *Processor) ExtractPDF(ctx context.Context, filePath string) (*DocumentIntelligence, error) {
	return p.Extract(ctx, filePath)
}

// GetStructuredMarkdown returns markdown representation for readable output
func (p *Processor) GetStructuredMarkdown(result *DocumentIntelligence) string {
	if result == nil {
		return ""
	}
	return result.DoclingMarkdown
}

// GetTableAsJSON returns specific table as JSON for structured storage
func (p *Processor) GetTableAsJSON(result *DocumentIntelligence, tableIndex int) (json.RawMessage, error) {
	if result == nil || tableIndex >= len(result.Tables) {
		return nil, fmt.Errorf("table index out of range")
	}

	data, err := json.Marshal(result.Tables[tableIndex])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal table: %w", err)
	}

	return data, nil
}

// GetAllTablesAsJSON returns all tables as JSON array
func (p *Processor) GetAllTablesAsJSON(result *DocumentIntelligence) (json.RawMessage, error) {
	if result == nil {
		return nil, fmt.Errorf("result is nil")
	}

	data, err := json.Marshal(result.Tables)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tables: %w", err)
	}

	return data, nil
}

// GetStructureAsJSON returns document structure/hierarchy as JSON
func (p *Processor) GetStructureAsJSON(result *DocumentIntelligence) (json.RawMessage, error) {
	if result == nil {
		return nil, fmt.Errorf("result is nil")
	}

	data, err := json.Marshal(result.Structure)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal structure: %w", err)
	}

	return data, nil
}

// GetMetadataAsJSON returns document metadata as JSON
func (p *Processor) GetMetadataAsJSON(result *DocumentIntelligence) (json.RawMessage, error) {
	if result == nil {
		return nil, fmt.Errorf("result is nil")
	}

	data, err := json.Marshal(result.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	return data, nil
}

// Helper functions

// extractKeywordsFromStructure extracts keywords from document structure
// extractKeywordsFromStructure extracts up to 50 unique keywords from the provided DocumentStructure by selecting heading elements' content.
// It trims whitespace, ignores empty values, enforces a minimum length of 3 and maximum of 199 characters, and preserves the original heading order.
func extractKeywordsFromStructure(structure DocumentStructure) []string {
	keywords := []string{}
	keywordSet := make(map[string]bool) // For deduplication

	// Extract from all headings
	for _, elem := range structure.Elements {
		if elem.Type == "heading" && elem.Content != "" {
			// Clean and add keyword
			keyword := strings.TrimSpace(elem.Content)
			if len(keyword) > 2 && len(keyword) < 200 && !keywordSet[keyword] {
				keywords = append(keywords, keyword)
				keywordSet[keyword] = true
			}
		}
	}

	// Limit to reasonable number
	if len(keywords) > 50 {
		keywords = keywords[:50]
	}

	return keywords
}

// detectDocumentType attempts to detect document type from content
// For legal documents, this looks for common patterns
// detectDocumentType infers a document type from the provided ProcessingResult using simple heuristics.
// It returns "unknown" if result is nil. The function examines raw text and the metadata title to classify
// common legal and generic document types (for example: "police_report", "arrest_record", "indictment",
// "court_order", "motion", "legal_brief", "contract", "subpoena", "deposition", "report", "memo") and
// returns "document" when no specific pattern is matched.
func detectDocumentType(result *ProcessingResult) string {
	if result == nil {
		return "unknown"
	}

	lowerText := strings.ToLower(result.RawText)

	// Legal document patterns
	if strings.Contains(lowerText, "police report") || strings.Contains(lowerText, "incident report") {
		return "police_report"
	}
	if strings.Contains(lowerText, "arrest record") || strings.Contains(lowerText, "booking") {
		return "arrest_record"
	}
	if strings.Contains(lowerText, "indictment") || strings.Contains(lowerText, "charged") {
		return "indictment"
	}
	if strings.Contains(lowerText, "court order") || strings.Contains(lowerText, "judgment") {
		return "court_order"
	}
	if strings.Contains(lowerText, "motion") && strings.Contains(lowerText, "court") {
		return "motion"
	}
	if strings.Contains(lowerText, "brief") && (strings.Contains(lowerText, "plaintiff") || strings.Contains(lowerText, "defendant")) {
		return "legal_brief"
	}
	if strings.Contains(lowerText, "contract") || strings.Contains(lowerText, "agreement") {
		return "contract"
	}
	if strings.Contains(lowerText, "subpoena") {
		return "subpoena"
	}
	if strings.Contains(lowerText, "deposition") {
		return "deposition"
	}

	// Generic types
	if result.Metadata.Title != "" {
		if strings.Contains(strings.ToLower(result.Metadata.Title), "report") {
			return "report"
		}
		if strings.Contains(strings.ToLower(result.Metadata.Title), "memo") {
			return "memo"
		}
	}

	return "document"
}

// CompareWithMistral prepares comparison data if document was also processed by Mistral
// Useful for audit and quality assessment
type ComparisonMetrics struct {
	TextLengthDocling  int
	TextLengthMistral  int
	PagesDocling       int
	PagesMistral       int
	LanguageDocling    string
	LanguageMistral    string
	KeywordsDocling    int
	KeywordsMistral    int
	Confidence         float64 // How similar the results are (0-1)
}

// DocumentComparison compares Docling results with Mistral results (if available)
// Returns metrics about differences and similarities
type DocumentComparison struct {
	Metrics      ComparisonMetrics
	Differences  []string
	Similarities []string
	Confidence   float64
	Recommendation string // Which to prefer: docling, mistral, or combine
}

// BuildComparison creates a comparison between Docling and Mistral results
// (Mistral comparison logic to be added when integrating with Mistral)
func BuildComparison(docling *DocumentIntelligence) *DocumentComparison {
	return &DocumentComparison{
		Metrics: ComparisonMetrics{
			TextLengthDocling: len(docling.TextContent),
			PagesDocling:      docling.PageCount,
			LanguageDocling:   docling.Language,
			KeywordsDocling:   len(docling.Keywords),
		},
		Confidence:   docling.Confidence,
		Recommendation: "use_docling", // Default recommendation
	}
}