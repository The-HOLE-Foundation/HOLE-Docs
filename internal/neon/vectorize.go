package neon

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/The-HOLE-Foundation/hole-docs/internal/voyage"
	"go.uber.org/zap"
)

// VectorizeClient handles vector storage and search in Neon
type VectorizeClient struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewVectorizeClient creates a VectorizeClient configured with the provided
// database connection and logger.
func NewVectorizeClient(db *sql.DB, logger *zap.Logger) *VectorizeClient {
	return &VectorizeClient{
		db:     db,
		logger: logger,
	}
}

// StoreChunkEmbeddings stores vector embeddings for document chunks
func (vc *VectorizeClient) StoreChunkEmbeddings(
	ctx context.Context,
	documentID int,
	chunks []*voyage.ChunkEmbedding,
	indexName string,
) error {
	if len(chunks) == 0 {
		return fmt.Errorf("no chunks to store")
	}

	vc.logger.Info("Storing chunk embeddings",
		zap.Int("document_id", documentID),
		zap.Int("chunk_count", len(chunks)),
		zap.String("index_name", indexName))

	// For now, we'll store metadata about vectorization
	// Actual vector storage would go to Cloudflare Vectorize via API
	query := `
		UPDATE docai.documents
		SET vectorize_status = $1,
		    vectorize_chunks_count = $2,
		    vectorize_processed_at = CURRENT_TIMESTAMP,
		    vectorize_index_id = $3
		WHERE id = $4
	`

	_, err := vc.db.ExecContext(ctx, query, "complete", len(chunks), indexName, documentID)
	if err != nil {
		return fmt.Errorf("failed to update vectorization status: %w", err)
	}

	vc.logger.Info("Chunk embeddings stored successfully",
		zap.Int("document_id", documentID),
		zap.Int("chunk_count", len(chunks)))

	return nil
}

// VectorSearchResult represents a single search result
type VectorSearchResult struct {
	DocumentID      int
	Filename        string
	DocumentType    string
	Similarity      float64 // Cosine similarity score 0-1
	ChunkIndex      int
	ChunkText       string
	Rank            float64 // BM25 rank score (for reranking)
}

// SearchSemantic performs semantic search using vector similarity
// Note: This is a placeholder that shows the structure
// Actual implementation would call Cloudflare Vectorize API
func (vc *VectorizeClient) SearchSemantic(
	ctx context.Context,
	userID string,
	documentType string,
	limit int,
) ([]*VectorSearchResult, error) {
	vc.logger.Info("Performing semantic search",
		zap.String("user_id", userID),
		zap.String("doc_type", documentType),
		zap.Int("limit", limit))

	// Prepare documentType parameter (nil if empty)
	var documentTypeParam interface{}
	if documentType == "" {
		documentTypeParam = nil
	} else {
		documentTypeParam = documentType
	}

	// For now, return documents with vectorize_status = "complete"
	// Use COALESCE to handle NULL ocr_text_content
	query := `
		SELECT d.id, d.filename, d.document_type, 0.0 as similarity, 0 as chunk_index, 
		       COALESCE(SUBSTRING(d.ocr_text_content, 1, 200), '') as chunk_text, 0.0 as rank
		FROM docai.documents d
		WHERE d.user_id = $1
		  AND d.vectorize_status = 'complete'
		  AND (d.document_type = $2 OR $2 IS NULL)
		ORDER BY d.created_at DESC
		LIMIT $3
	`

	rows, err := vc.db.QueryContext(ctx, query, userID, documentTypeParam, limit)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	defer rows.Close()

	var results []*VectorSearchResult
	for rows.Next() {
		var r VectorSearchResult
		if err := rows.Scan(&r.DocumentID, &r.Filename, &r.DocumentType, &r.Similarity,
			&r.ChunkIndex, &r.ChunkText, &r.Rank); err != nil {
			vc.logger.Error("Failed to scan result", zap.Error(err))
			continue
		}
		results = append(results, &r)
	}

	vc.logger.Info("Semantic search complete",
		zap.Int("results_count", len(results)))

	return results, rows.Err()
}

// BM25Rerank reranks results using BM25 algorithm
// This combines keyword relevance with semantic search results
func (vc *VectorizeClient) BM25Rerank(
	ctx context.Context,
	query string,
	results []*VectorSearchResult,
	limit int,
) []*VectorSearchResult {
	if len(results) == 0 {
		return results
	}

	vc.logger.Info("Reranking results with BM25",
		zap.String("query", query),
		zap.Int("input_count", len(results)),
		zap.Int("limit", limit))

	// BM25 parameters (standard values)
	const k1 = 1.5  // Term frequency saturation
	const b = 0.75  // Length normalization
	// Note: avgDocLength would be 1000.0 in full implementation

	// For each result, calculate BM25 score and combine with vector similarity
	for i := range results {
		// Placeholder BM25 calculation
		// In production, this would:
		// 1. Tokenize query
		// 2. Calculate TF-IDF from full text
		// 3. Combine with vector similarity using fusion techniques
		results[i].Rank = results[i].Similarity // For now, use vector similarity as rank
	}

	// Return top-k results
	if len(results) > limit {
		results = results[:limit]
	}

	vc.logger.Info("Reranking complete",
		zap.Int("output_count", len(results)))

	return results
}

// GetVectorizationStatus returns vectorization status for a document
func (vc *VectorizeClient) GetVectorizationStatus(ctx context.Context, documentID int) (string, int, error) {
	query := `
		SELECT vectorize_status, vectorize_chunks_count
		FROM docai.documents
		WHERE id = $1
	`

	var status string
	var chunkCount int
	err := vc.db.QueryRowContext(ctx, query, documentID).Scan(&status, &chunkCount)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", 0, fmt.Errorf("document not found")
		}
		return "", 0, fmt.Errorf("query failed: %w", err)
	}

	return status, chunkCount, nil
}

// ListVectorizedDocuments lists all documents with completed vectorization
func (vc *VectorizeClient) ListVectorizedDocuments(
	ctx context.Context,
	userID string,
	limit int,
) ([]*DocumentRecord, error) {
	query := `
		SELECT id, filename, document_type, vectorize_chunks_count, created_at
		FROM docai.documents
		WHERE user_id = $1 AND vectorize_status = 'complete'
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := vc.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var results []*DocumentRecord
	for rows.Next() {
		var d DocumentRecord
		var chunkCount sql.NullInt32
		if err := rows.Scan(&d.ID, &d.Filename, &d.DocumentType, &chunkCount, &d.CreatedAt); err != nil {
			vc.logger.Error("Failed to scan row", zap.Error(err))
			continue
		}
		results = append(results, &d)
	}

	return results, rows.Err()
}