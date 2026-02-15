package docling

import (
	"encoding/json"
	"time"
)

// DocumentMetadata represents extracted document metadata from Docling
type DocumentMetadata struct {
	Title           string    `json:"title,omitempty"`
	Author          string    `json:"author,omitempty"`
	Subject         string    `json:"subject,omitempty"`
	Creator         string    `json:"creator,omitempty"`
	Producer        string    `json:"producer,omitempty"`
	CreatedDate     *time.Time `json:"created_date,omitempty"`
	ModifiedDate    *time.Time `json:"modified_date,omitempty"`
	Language        string    `json:"language,omitempty"`
	PageCount       int       `json:"page_count,omitempty"`
	FileSize        int64     `json:"file_size,omitempty"`
	ContentType     string    `json:"content_type,omitempty"`
}

// TableCell represents a single cell in a table
type TableCell struct {
	Content string `json:"content"`
	Colspan int    `json:"colspan,omitempty"`
	Rowspan int    `json:"rowspan,omitempty"`
}

// TableRow represents a row in a table
type TableRow struct {
	Cells []TableCell `json:"cells"`
}

// Table represents structured table data extracted from document
type Table struct {
	ID       string     `json:"id,omitempty"`
	Caption  string     `json:"caption,omitempty"`
	Rows     []TableRow `json:"rows"`
	BBox     *BoundingBox `json:"bbox,omitempty"`
}

// BoundingBox represents coordinate information for document elements
type BoundingBox struct {
	X0 float64 `json:"x0"`
	Y0 float64 `json:"y0"`
	X1 float64 `json:"x1"`
	Y1 float64 `json:"y1"`
	PageNumber int `json:"page,omitempty"`
}

// DocumentElement represents a single element in document structure
type DocumentElement struct {
	Type       string          `json:"type"` // heading, paragraph, table, image, list, code, etc.
	Level      int             `json:"level,omitempty"` // For headings: 1-6
	Content    string          `json:"content,omitempty"`
	Children   []DocumentElement `json:"children,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	BBox       *BoundingBox    `json:"bbox,omitempty"`
}

// DocumentStructure represents hierarchical document structure
type DocumentStructure struct {
	Elements []DocumentElement `json:"elements"`
	TOC      []DocumentElement `json:"toc,omitempty"` // Table of contents
}

// ProcessingResult represents the complete Docling processing output
type ProcessingResult struct {
	// Document identification
	DocumentID string `json:"document_id,omitempty"`
	FileName   string `json:"file_name"`

	// Extracted content
	RawText    string `json:"raw_text"` // Plain text extraction
	Markdown   string `json:"markdown"` // Markdown representation

	// Structured data
	Metadata   DocumentMetadata `json:"metadata"`
	Structure  DocumentStructure `json:"structure"`
	Tables     []Table `json:"tables,omitempty"`

	// Document insights
	Language   string `json:"language,omitempty"`
	PageCount  int    `json:"page_count"`

	// Processing metadata
	ProcessedAt    time.Time `json:"processed_at"`
	ProcessingTime int64     `json:"processing_time_ms"` // milliseconds
	Version        string    `json:"version"` // Docling version

	// Full response for archival
	RawJSON json.RawMessage `json:"raw_json,omitempty"`
}

// ProcessRequest is sent to Docling API
type ProcessRequest struct {
	FilePath string `json:"file_path"`
	FileName string `json:"file_name"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

// ProcessResponse is returned from Docling API
type ProcessResponse struct {
	Success bool                   `json:"success"`
	Data    ProcessingResult       `json:"data,omitempty"`
	Error   string                 `json:"error,omitempty"`
	ErrorDetails map[string]interface{} `json:"error_details,omitempty"`
}

// HealthCheckResponse represents health check status
type HealthCheckResponse struct {
	Status  string `json:"status"` // ok, degraded, error
	Version string `json:"version"`
	Models  []string `json:"models_loaded"`
	Message string `json:"message,omitempty"`
}
