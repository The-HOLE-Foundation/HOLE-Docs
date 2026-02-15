package neon

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/The-HOLE-Foundation/hole-docs/internal/mistral"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

// DocumentsClient handles document operations in Neon
type DocumentsClient struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewDocumentsClient returns a DocumentsClient configured with the provided database handle and logger.
func NewDocumentsClient(db *sql.DB, logger *zap.Logger) *DocumentsClient {
	return &DocumentsClient{
		db:     db,
		logger: logger,
	}
}

// DocumentRecord represents a document in the database
type DocumentRecord struct {
	ID                       int
	UserID                   string
	Filename                 string
	DocumentType             string
	DocumentCategory         string
	ClassificationConfidence float64
	OCRPageCount             int
	Keywords                 []string
	R2Path                   string
	Project                  string
	CreatedAt                time.Time
}

// InsertDocument inserts a new document with OCR results
func (dc *DocumentsClient) InsertDocument(
	ctx context.Context,
	userID string,
	filename string,
	originalFilename string,
	fileData []byte,
	ocrResult *mistral.OCRResponse,
	classification *mistral.DocumentClassification,
	entities *mistral.ExtractedEntities,
	keywords []string,
	searchIndex string,
	r2Path string,
	projectName string,
) (int, error) {
	dc.logger.Info("Inserting document into Neon",
		zap.String("filename", filename),
		zap.String("user_id", userID))

	// Validate required pointer parameters
	if ocrResult == nil {
		return 0, fmt.Errorf("ocrResult cannot be nil")
	}
	if classification == nil {
		return 0, fmt.Errorf("classification cannot be nil")
	}
	if entities == nil {
		return 0, fmt.Errorf("entities cannot be nil")
	}

	// Calculate file hash
	hash := sha256.Sum256(fileData)
	fileHash := hex.EncodeToString(hash[:])

	// Marshal JSONB fields
	ocrStructureJSON, err := json.Marshal(ocrResult.Structure)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal ocr structure: %w", err)
	}

	imagesBBoxesJSON, err := json.Marshal(ocrResult.ImagesBBoxes)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal images bboxes: %w", err)
	}

	entitiesJSON, err := json.Marshal(entities)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal entities: %w", err)
	}

	// Insert document
	// Note: PDF stored in Cloudflare R2 (Phase 2). Currently only storing metadata.
	query := `
		INSERT INTO docai.documents (
			user_id,
			filename,
			original_filename,
			file_hash,
			mime_type,
			file_size_bytes,
			document_type,
			document_category,
			classification_confidence,
			classification_model,
			ocr_text_content,
			ocr_header,
			ocr_footer,
			ocr_page_count,
			ocr_structure,
			ocr_images_bboxes,
			r2_path,
			search_index,
			keywords,
			extracted_entities,
			project_name,
			mistral_processed_at,
			uploaded_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)
		RETURNING id
	`

	var docID int
	err = dc.db.QueryRowContext(
		ctx,
		query,
		userID,
		filename,
		originalFilename,
		fileHash,
		"application/pdf",
		len(fileData),
		classification.DocumentType,
		classification.Category,
		classification.Confidence,
		"mistral-large-latest",
		ocrResult.TextContent,
		ocrResult.Header,
		ocrResult.Footer,
		ocrResult.GetPageCount(),
		ocrStructureJSON,
		imagesBBoxesJSON,
		r2Path,
		searchIndex,
		pq.Array(keywords),
		entitiesJSON,
		projectName,
		time.Now(),
		time.Now(),
	).Scan(&docID)

	if err != nil {
		dc.logger.Error("Failed to insert document",
			zap.Error(err))
		return 0, fmt.Errorf("failed to insert document: %w", err)
	}

	dc.logger.Info("Document inserted successfully",
		zap.Int("document_id", docID))

	// Insert extractions if we have detailed data
	if len(entities.CaseNumbers) > 0 || len(entities.People) > 0 {
		if err := dc.insertExtractions(ctx, docID, entities); err != nil {
			dc.logger.Warn("Failed to insert extractions (continuing anyway)",
				zap.Error(err))
		}
	}

	// Log to audit
	dc.logAudit(ctx, userID, "upload", &docID, map[string]interface{}{
		"filename":      filename,
		"document_type": classification.DocumentType,
		"pages":         ocrResult.GetPageCount(),
	}, "success", "")

	return docID, nil
}

// insertExtractions inserts detailed extraction data
func (dc *DocumentsClient) insertExtractions(
	ctx context.Context,
	documentID int,
	entities *mistral.ExtractedEntities,
) error {
	var err error
	datesJSON, err := json.Marshal(entities.Dates)
	if err != nil {
		return fmt.Errorf("failed to marshal dates: %w", err)
	}
	chargesJSON, err := json.Marshal(entities.Charges)
	if err != nil {
		return fmt.Errorf("failed to marshal charges: %w", err)
	}
	partiesJSON, err := json.Marshal(entities.Parties)
	if err != nil {
		return fmt.Errorf("failed to marshal parties: %w", err)
	}

	query := `
		INSERT INTO docai.document_extractions (
			document_id,
			people_mentioned,
			dates_mentioned,
			locations_mentioned,
			case_numbers,
			court_name,
			judge_name,
			charges,
			parties_involved,
			extraction_model,
			extracted_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (document_id) DO UPDATE SET
			people_mentioned = EXCLUDED.people_mentioned,
			dates_mentioned = EXCLUDED.dates_mentioned,
			locations_mentioned = EXCLUDED.locations_mentioned,
			case_numbers = EXCLUDED.case_numbers,
			court_name = EXCLUDED.court_name,
			judge_name = EXCLUDED.judge_name,
			charges = EXCLUDED.charges,
			parties_involved = EXCLUDED.parties_involved,
			extracted_at = EXCLUDED.extracted_at
	`

	_, err = dc.db.ExecContext(
		ctx,
		query,
		documentID,
		pq.Array(entities.People),
		datesJSON,
		pq.Array(entities.Locations),
		pq.Array(entities.CaseNumbers),
		entities.Court,
		entities.Judge,
		chargesJSON,
		partiesJSON,
		"mistral-large-latest",
		time.Now(),
	)

	return err
}

// SearchDocuments performs full-text search
func (dc *DocumentsClient) SearchDocuments(
	ctx context.Context,
	userID string,
	searchQuery string,
	projectName string,
	limit int,
) ([]*DocumentRecord, error) {
	dc.logger.Info("Searching documents",
		zap.String("user_id", userID),
		zap.String("query", searchQuery),
		zap.String("project", projectName),
		zap.Int("limit", limit))

	// Use the helper function we created
	query := `SELECT * FROM docai.search_documents($1, $2, $3)`

	rows, err := dc.db.QueryContext(ctx, query, userID, searchQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("search query failed: %w", err)
	}
	defer rows.Close()

	var results []*DocumentRecord
	for rows.Next() {
		var id int
		var filename, docType, snippet string
		var rank float64

		if err := rows.Scan(&id, &filename, &docType, &snippet, &rank); err != nil {
			dc.logger.Error("Failed to scan row", zap.Error(err))
			continue
		}

		// Apply project filter if specified
		if projectName != "" {
			// TODO: Filter by project - requires schema update to include project in search results
			// For now, all results are included but field is preserved for future use
		}

		results = append(results, &DocumentRecord{
			ID:           id,
			Filename:     filename,
			DocumentType: docType,
			Project:      projectName,
		})
	}

	dc.logger.Info("Search complete",
		zap.Int("results_count", len(results)))

	// Log search query
	dc.logSearchQuery(ctx, userID, searchQuery, "keyword", len(results))

	return results, nil
}

// ListDocuments lists documents for a user
func (dc *DocumentsClient) ListDocuments(
	ctx context.Context,
	userID string,
	documentType string,
	projectName string,
	limit int,
) ([]*DocumentRecord, error) {
	dc.logger.Info("Listing documents",
		zap.String("user_id", userID),
		zap.String("type", documentType),
		zap.String("project", projectName),
		zap.Int("limit", limit))

	query := `
		SELECT id, filename, document_type, document_category,
		       classification_confidence, ocr_page_count, keywords, r2_path, project_name, created_at
		FROM docai.documents
		WHERE user_id = $1
		  AND ($2 = '' OR document_type = $2)
		  AND ($3 = '' OR project_name = $3)
		ORDER BY created_at DESC
		LIMIT $4
	`

	rows, err := dc.db.QueryContext(ctx, query, userID, documentType, projectName, limit)
	if err != nil {
		return nil, fmt.Errorf("list query failed: %w", err)
	}
	defer rows.Close()

	var results []*DocumentRecord
	for rows.Next() {
		doc := &DocumentRecord{}
		var docType, docCategory sql.NullString
		if err := rows.Scan(
			&doc.ID,
			&doc.Filename,
			&docType,
			&docCategory,
			&doc.ClassificationConfidence,
			&doc.OCRPageCount,
			pq.Array(&doc.Keywords),
			&doc.R2Path,
			&doc.Project,
			&doc.CreatedAt,
		); err != nil {
			dc.logger.Error("Failed to scan row", zap.Error(err))
			continue
		}

		// Handle nullable fields
		if docType.Valid {
			doc.DocumentType = docType.String
		}
		if docCategory.Valid {
			doc.DocumentCategory = docCategory.String
		}

		results = append(results, doc)
	}

	dc.logger.Info("List complete",
		zap.Int("results_count", len(results)))

	return results, nil
}

// GetDocumentR2Path gets the R2 path for a document
func (dc *DocumentsClient) GetDocumentR2Path(
	ctx context.Context,
	userID string,
	documentID int,
) (string, error) {
	var r2Path string
	query := `SELECT r2_path FROM docai.documents WHERE id = $1 AND user_id = $2`

	err := dc.db.QueryRowContext(ctx, query, documentID, userID).Scan(&r2Path)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("document not found or access denied")
		}
		return "", fmt.Errorf("failed to get r2 path: %w", err)
	}

	// Log access
	dc.logDocumentAccess(ctx, userID, documentID, "view")

	return r2Path, nil
}

// logAudit logs an action to audit_logs
func (dc *DocumentsClient) logAudit(
	ctx context.Context,
	userID string,
	action string,
	documentID *int,
	details map[string]interface{},
	status string,
	errorMsg string,
) {
	detailsJSON, _ := json.Marshal(details)

	query := `
		INSERT INTO docai.audit_logs (user_id, action, document_id, details, status, error_message, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := dc.db.ExecContext(
		ctx,
		query,
		userID,
		action,
		documentID,
		detailsJSON,
		status,
		errorMsg,
		time.Now(),
	)

	if err != nil {
		dc.logger.Error("Failed to log audit", zap.Error(err))
	}
}

// logSearchQuery logs a search query
func (dc *DocumentsClient) logSearchQuery(
	ctx context.Context,
	userID string,
	queryText string,
	queryType string,
	resultsCount int,
) {
	query := `
		INSERT INTO docai.search_queries (user_id, query_text, query_type, results_count, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := dc.db.ExecContext(ctx, query, userID, queryText, queryType, resultsCount, time.Now())
	if err != nil {
		dc.logger.Error("Failed to log search query", zap.Error(err))
	}
}

// logDocumentAccess logs document access
func (dc *DocumentsClient) logDocumentAccess(
	ctx context.Context,
	userID string,
	documentID int,
	accessType string,
) {
	query := `
		INSERT INTO docai.document_access (user_id, document_id, access_type, access_source, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := dc.db.ExecContext(ctx, query, userID, documentID, accessType, "cli", time.Now())
	if err != nil {
		dc.logger.Error("Failed to log document access", zap.Error(err))
	}
}

// DocumentExport represents a document ready for download
type DocumentExport struct {
	ID       int
	Filename string
	FileData []byte
	FileSize int64
}

// GetDocumentByID retrieves a document by ID for download/export
func (dc *DocumentsClient) GetDocumentByID(
	ctx context.Context,
	userID string,
	documentID int,
) (*DocumentExport, error) {
	dc.logger.Info("Retrieving document for download",
		zap.String("user_id", userID),
		zap.Int("document_id", documentID))

	// Check if document exists and belongs to user
	// Note: file_data storage needs to be verified in schema
	query := `
		SELECT id, filename, file_size_bytes
		FROM docai.documents
		WHERE id = $1 AND user_id = $2
	`

	var docID int
	var filename string
	var fileSize int64

	err := dc.db.QueryRowContext(ctx, query, documentID, userID).Scan(
		&docID,
		&filename,
		&fileSize,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("document not found or access denied")
		}
		return nil, fmt.Errorf("failed to retrieve document: %w", err)
	}

	// Log access
	dc.logDocumentAccess(ctx, userID, documentID, "download")

	return &DocumentExport{
		ID:       docID,
		Filename: filename,
		FileSize: fileSize,
		FileData: nil, // Phase 2: File data retrieval from Cloudflare R2
	}, nil
}

// GetUserStats gets statistics for a user
func (dc *DocumentsClient) GetUserStats(ctx context.Context, userID string) (map[string]interface{}, error) {
	query := `
		SELECT
			COUNT(*) as total_documents,
			COALESCE(SUM(ocr_page_count), 0) as total_pages,
			COALESCE(SUM(file_size_bytes), 0) as storage_bytes,
			COUNT(*) FILTER (WHERE vectorize_status = 'complete') as vectorized_documents
		FROM docai.documents
		WHERE user_id = $1
	`

	var totalDocs, totalPages, vectorizedDocs int
	var storageBytes int64

	err := dc.db.QueryRowContext(ctx, query, userID).Scan(
		&totalDocs,
		&totalPages,
		&storageBytes,
		&vectorizedDocs,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return map[string]interface{}{
		"total_documents":       totalDocs,
		"total_pages":           totalPages,
		"storage_bytes":         storageBytes,
		"storage_mb":            float64(storageBytes) / (1024 * 1024),
		"vectorized_documents":  vectorizedDocs,
	}, nil
}

// InsertDocumentWithDocling inserts a document processed with Docling pipeline
func (dc *DocumentsClient) InsertDocumentWithDocling(
	ctx context.Context,
	userID string,
	filename string,
	originalFilename string,
	fileData []byte,
	doclingIntelligence interface{}, // Will be *docling.DocumentIntelligence, avoiding circular import
	keywords []string,
	searchIndex string,
	r2Path string,
	projectName string,
) (int, error) {
	dc.logger.Info("Inserting document with Docling pipeline",
		zap.String("filename", filename),
		zap.String("user_id", userID))

	// Calculate file hash
	hash := sha256.Sum256(fileData)
	fileHash := hex.EncodeToString(hash[:])

	// Convert docling intelligence to map for flexible access
	var doclingMap map[string]interface{}
	doclingJSON, err := json.Marshal(doclingIntelligence)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal docling intelligence: %w", err)
	}
	if err := json.Unmarshal(doclingJSON, &doclingMap); err != nil {
		return 0, fmt.Errorf("failed to parse docling intelligence: %w", err)
	}

	// Extract fields from docling intelligence
	textContent := ""
	if v, ok := doclingMap["TextContent"].(string); ok {
		textContent = v
	}

	pageCount := 0
	if v, ok := doclingMap["PageCount"].(float64); ok {
		pageCount = int(v)
	}

	language := "en"
	if v, ok := doclingMap["Language"].(string); ok && v != "" {
		language = v
	}

	// Get document type (may be auto-detected by Docling)
	documentType := "document"
	if v, ok := doclingMap["DocumentType"].(string); ok && v != "" {
		documentType = v
	}

	// Insert document with Docling pipeline
	// Note: docling_json stores the full lossless output for legal document fidelity
	query := `
		INSERT INTO docai.documents (
			user_id,
			filename,
			original_filename,
			file_hash,
			mime_type,
			file_size_bytes,
			document_type,
			document_category,
			classification_confidence,
			classification_model,
			ocr_text_content,
			ocr_page_count,
			r2_path,
			search_index,
			keywords,
			project_name,
			processing_pipeline,
			processing_version,
			processing_timestamp,
			docling_json,
			docling_metadata,
			docling_tables,
			docling_structure,
			docling_markdown,
			docling_processing_status,
			uploaded_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26)
		RETURNING id
	`

	// Marshal JSONB fields
	var metadataJSON, tablesJSON, structureJSON, markdownJSON []byte
	if metadata, ok := doclingMap["Metadata"]; ok && metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(metadata)
		if err != nil {
			return 0, fmt.Errorf("failed to marshal docling metadata: %w", err)
		}
	}
	if tables, ok := doclingMap["Tables"]; ok && tables != nil {
		var err error
		tablesJSON, err = json.Marshal(tables)
		if err != nil {
			return 0, fmt.Errorf("failed to marshal docling tables: %w", err)
		}
	}
	if structure, ok := doclingMap["Structure"]; ok && structure != nil {
		var err error
		structureJSON, err = json.Marshal(structure)
		if err != nil {
			return 0, fmt.Errorf("failed to marshal docling structure: %w", err)
		}
	}
	if markdown, ok := doclingMap["DoclingMarkdown"]; ok && markdown != nil {
		var err error
		markdownJSON, err = json.Marshal(markdown)
		if err != nil {
			return 0, fmt.Errorf("failed to marshal docling markdown: %w", err)
		}
	}

	var docID int
	err = dc.db.QueryRowContext(
		ctx,
		query,
		userID,                              // $1
		filename,                            // $2
		originalFilename,                    // $3
		fileHash,                            // $4
		"application/pdf",                   // $5
		len(fileData),                       // $6
		documentType,                        // $7
		"administrative",                    // $8
		0.9,                                 // $9 - Docling confidence
		"docling",                           // $10
		textContent,                         // $11
		pageCount,                           // $12
		r2Path,                              // $13
		searchIndex,                         // $14
		pq.Array(keywords),                  // $15
		projectName,                         // $16
		"docling",                           // $17 - processing_pipeline
		"1.0.0",                             // $18 - processing_version (Docling version)
		time.Now(),                          // $19 - processing_timestamp
		doclingJSON,                         // $20 - docling_json (full lossless output)
		metadataJSON,                        // $21 - docling_metadata
		tablesJSON,                          // $22 - docling_tables
		structureJSON,                       // $23 - docling_structure
		markdownJSON,                        // $24 - docling_markdown
		"complete",                          // $25 - docling_processing_status
		time.Now(),                          // $26 - uploaded_at
	).Scan(&docID)

	if err != nil {
		dc.logger.Error("Failed to insert document with Docling",
			zap.Error(err))
		return 0, fmt.Errorf("failed to insert document: %w", err)
	}

	dc.logger.Info("Document inserted successfully with Docling pipeline",
		zap.Int("document_id", docID),
		zap.String("language", language),
		zap.Int("pages", pageCount))

	// Log to audit
	dc.logAudit(ctx, userID, "upload", &docID, map[string]interface{}{
		"filename":    filename,
		"pipeline":    "docling",
		"pages":       pageCount,
		"language":    language,
	}, "success", "")

	return docID, nil
}

// UpdateDocumentProcessing updates the processing status for a document
func (dc *DocumentsClient) UpdateDocumentProcessing(
	ctx context.Context,
	documentID int,
	pipeline string,
	status string,
	errorMsg string,
) error {
	var query string
	var args []interface{}

	// Build query and args based on pipeline
	if pipeline == "docling" {
		query = `
		UPDATE docai.documents
		SET processing_pipeline = $1,
		    processing_timestamp = CURRENT_TIMESTAMP,
		    docling_processing_status = $3,
		    docling_error_message = $4
		WHERE id = $2
		RETURNING id
		`
		args = []interface{}{pipeline, documentID, status, errorMsg}
	} else if pipeline == "mistral" {
		query = `
		UPDATE docai.documents
		SET processing_pipeline = $1,
		    processing_timestamp = CURRENT_TIMESTAMP,
		    vectorize_status = $3
		WHERE id = $2
		RETURNING id
		`
		args = []interface{}{pipeline, documentID, status}
	} else {
		query = `
		UPDATE docai.documents
		SET processing_pipeline = $1,
		    processing_timestamp = CURRENT_TIMESTAMP
		WHERE id = $2
		RETURNING id
		`
		args = []interface{}{pipeline, documentID}
	}

	var returnedID int
	err := dc.db.QueryRowContext(ctx, query, args...).Scan(&returnedID)
	if err != nil {
		dc.logger.Error("Failed to update document processing",
			zap.Int("document_id", documentID),
			zap.String("pipeline", pipeline),
			zap.Error(err))
		return fmt.Errorf("failed to update document: %w", err)
	}

	return nil
}

// UpdateDocumentStatus updates the processing status of a document
func (dc *DocumentsClient) UpdateDocumentStatus(
	ctx context.Context,
	docID int,
	status string,
) error {
	query := `
		UPDATE docai.documents
		SET processing_status = $1,
		    updated_at = NOW()
		WHERE id = $2
	`

	_, err := dc.db.ExecContext(ctx, query, status, docID)
	if err != nil {
		dc.logger.Error("Failed to update document status",
			zap.Int("document_id", docID),
			zap.String("status", status),
			zap.Error(err))
		return fmt.Errorf("failed to update document status: %w", err)
	}

	dc.logger.Info("Document status updated",
		zap.Int("document_id", docID),
		zap.String("status", status))

	return nil
}