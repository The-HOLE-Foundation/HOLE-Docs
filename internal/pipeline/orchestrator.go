package pipeline

import (
	"context"
	"database/sql"
	"path/filepath"
	"time"

	"github.com/The-HOLE-Foundation/hole-docs/internal/docling"
	"github.com/The-HOLE-Foundation/hole-docs/internal/mistral"
	"github.com/The-HOLE-Foundation/hole-docs/internal/storage"
	"github.com/The-HOLE-Foundation/hole-docs/internal/voyage"
	"go.uber.org/zap"
)

// ProcessingOrchestrator coordinates the document processing pipeline
type ProcessingOrchestrator struct {
	r2Uploader     *storage.R2Uploader
	neonDB         *sql.DB
	doclingClient  *docling.Client
	mistralClient  *mistral.Client
	voyageClient   *voyage.Client
	logger         *zap.Logger
}

// NewProcessingOrchestrator creates a new orchestrator
func NewProcessingOrchestrator(
	r2Uploader *storage.R2Uploader,
	neonDB *sql.DB,
	doclingClient *docling.Client,
	mistralClient *mistral.Client,
	voyageClient *voyage.Client,
	logger *zap.Logger,
) *ProcessingOrchestrator {
	return &ProcessingOrchestrator{
		r2Uploader:    r2Uploader,
		neonDB:        neonDB,
		doclingClient: doclingClient,
		mistralClient: mistralClient,
		voyageClient:  voyageClient,
		logger:        logger,
	}
}

// ProcessResponse represents the result of document processing
type ProcessResponse struct {
	DocumentID   string                 `json:"document_id"`
	Filename     string                 `json:"filename"`
	Status       string                 `json:"status"`
	Pipeline     PipelineStatus         `json:"processing_pipeline"`
	Summary      ProcessSummary         `json:"summary"`
	CreatedAt    time.Time              `json:"created_at"`
	CompletedAt  time.Time              `json:"completed_at,omitempty"`
}

// PipelineStatus tracks each step in the processing pipeline
type PipelineStatus struct {
	FormatDetection struct {
		Status string `json:"status"`
		Type   string `json:"type,omitempty"`
	} `json:"format_detection"`
	TextExtraction struct {
		Status          string `json:"status"`
		CharacterCount  int    `json:"character_count,omitempty"`
		Language        string `json:"language,omitempty"`
	} `json:"text_extraction"`
	OriginalStorage struct {
		Status string `json:"status"`
		R2Path string `json:"r2_path,omitempty"`
	} `json:"original_storage"`
	MistralOCR struct {
		Status string `json:"status"`
		Reason string `json:"reason,omitempty"`
	} `json:"mistral_ocr"`
	DoclingIntelligence struct {
		Status          string `json:"status"`
		StructureCount  int    `json:"structure_count,omitempty"`
		TablesFound     int    `json:"tables_found,omitempty"`
		R2Path          string `json:"r2_path,omitempty"`
	} `json:"docling_intelligence"`
	TextChunking struct {
		Status       string `json:"status"`
		ChunkCount   int    `json:"chunk_count,omitempty"`
		AvgTokens    int    `json:"avg_tokens,omitempty"`
	} `json:"text_chunking"`
	VoyageVectorization struct {
		Status           string `json:"status"`
		EmbeddingsCount  int    `json:"embeddings_count,omitempty"`
		Model            string `json:"model,omitempty"`
	} `json:"voyage_vectorization"`
	Storage struct {
		Status  string   `json:"status"`
		NeonID  string   `json:"neon_id,omitempty"`
		R2Paths []string `json:"r2_paths,omitempty"`
	} `json:"storage"`
}

// ProcessSummary provides high-level processing results
type ProcessSummary struct {
	Pages      int    `json:"pages"`
	Chunks     int    `json:"chunks"`
	Vectors    int    `json:"vectors"`
	Searchable bool   `json:"searchable"`
	RAGReady   bool   `json:"rag_ready"`
}

// ProcessRequest represents a document processing request
type ProcessRequest struct {
	FilePath string                 `json:"file_path"`
	DocType  string                 `json:"document_type,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Options  ProcessingOptions      `json:"processing_options,omitempty"`
}

// ProcessingOptions controls which pipeline steps to execute
type ProcessingOptions struct {
	OCR            bool `json:"ocr"`
	ExtractTables  bool `json:"extract_tables"`
	Vectorize      bool `json:"vectorize"`
	StoreOriginal  bool `json:"store_original"`
}

// ProcessDocument orchestrates the complete document processing pipeline
func (p *ProcessingOrchestrator) ProcessDocument(ctx context.Context, req *ProcessRequest) (*ProcessResponse, error) {
	docID := storage.GenerateDocumentID()
	filename := filepath.Base(req.FilePath)

	resp := &ProcessResponse{
		DocumentID: docID,
		Filename:   filename,
		Status:     "processing",
		CreatedAt:  time.Now(),
	}

	p.logger.Info("Starting document processing",
		zap.String("doc_id", docID),
		zap.String("filename", filename))

	// Step 1: Detect format
	resp.Pipeline.FormatDetection.Status = "in_progress"
	fileType := p.detectFormat(filename)
	resp.Pipeline.FormatDetection.Status = "completed"
	resp.Pipeline.FormatDetection.Type = fileType

	p.logger.Info("Format detected",
		zap.String("doc_id", docID),
		zap.String("type", fileType))

	// Step 2: Extract text (based on format)
	resp.Pipeline.TextExtraction.Status = "in_progress"
	var rawText string
	var err error

	switch fileType {
	case "pdf":
		rawText, err = p.extractPDFText(ctx, req.FilePath)
	case "docx":
		rawText, err = p.extractDocxText(ctx, req.FilePath)
	case "image":
		if req.Options.OCR {
			rawText, err = p.ocrImage(ctx, req.FilePath)
		} else {
			rawText = "[Image - OCR skipped]"
		}
	default:
		rawText = ""
	}

	if err != nil {
		p.logger.Error("Text extraction failed", zap.String("doc_id", docID), zap.Error(err))
		resp.Pipeline.TextExtraction.Status = "failed"
		resp.Status = "failed"
		return resp, err
	}

	resp.Pipeline.TextExtraction.Status = "completed"
	resp.Pipeline.TextExtraction.CharacterCount = len(rawText)

	// Step 3: Upload original to R2 (if enabled)
	if req.Options.StoreOriginal {
		resp.Pipeline.OriginalStorage.Status = "in_progress"
		r2Path, err := p.r2Uploader.UploadOriginal(ctx, req.FilePath, docID)
		if err != nil {
			p.logger.Error("Failed to upload original", zap.String("doc_id", docID), zap.Error(err))
			resp.Pipeline.OriginalStorage.Status = "failed"
		} else {
			resp.Pipeline.OriginalStorage.Status = "completed"
			resp.Pipeline.OriginalStorage.R2Path = r2Path
		}
	} else {
		resp.Pipeline.OriginalStorage.Status = "skipped"
	}

	// Step 4: Mistral OCR (if image/scan)
	if fileType == "image" && req.Options.OCR {
		resp.Pipeline.MistralOCR.Status = "not_implemented"
		resp.Pipeline.MistralOCR.Reason = "OCR integration pending"
		// TODO: Integrate with mistral client for OCR
	} else {
		resp.Pipeline.MistralOCR.Status = "skipped"
		resp.Pipeline.MistralOCR.Reason = "not_required"
	}

	// Step 5: Docling Intelligence
	if req.Options.ExtractTables {
		resp.Pipeline.DoclingIntelligence.Status = "not_implemented"
		resp.Pipeline.DoclingIntelligence.StructureCount = 0
		resp.Pipeline.DoclingIntelligence.TablesFound = 0
		// TODO: Integrate with docling client
	} else {
		resp.Pipeline.DoclingIntelligence.Status = "skipped"
	}

	// Step 6: Text Chunking
	resp.Pipeline.TextChunking.Status = "in_progress"
	chunks := p.chunkText(rawText)
	resp.Pipeline.TextChunking.Status = "completed"
	resp.Pipeline.TextChunking.ChunkCount = len(chunks)

	// Step 7: Voyage Vectorization
	if req.Options.Vectorize {
		resp.Pipeline.VoyageVectorization.Status = "not_implemented"
		resp.Pipeline.VoyageVectorization.EmbeddingsCount = 0
		resp.Pipeline.VoyageVectorization.Model = "voyage-4-large"
		// TODO: Integrate with voyage client
	} else {
		resp.Pipeline.VoyageVectorization.Status = "skipped"
	}

	// Step 8: Store metadata in Neon
	resp.Pipeline.Storage.Status = "not_implemented"
	resp.Pipeline.Storage.NeonID = docID
	// TODO: Create document in Neon database with metadata from processing results

	// Finalize response
	resp.Status = "completed"
	resp.CompletedAt = time.Now()
	resp.Summary.Pages = 1          // TODO: Calculate from document
	resp.Summary.Chunks = len(chunks)
	resp.Summary.Vectors = len(chunks)
	resp.Summary.Searchable = true
	resp.Summary.RAGReady = req.Options.Vectorize

	p.logger.Info("Document processing completed",
		zap.String("doc_id", docID),
		zap.String("status", resp.Status),
		zap.Duration("duration", resp.CompletedAt.Sub(resp.CreatedAt)))

	return resp, nil
}

// Helper methods

func (p *ProcessingOrchestrator) detectFormat(filename string) string {
	ext := filepath.Ext(filename)
	switch ext {
	case ".pdf":
		return "pdf"
	case ".docx", ".doc":
		return "docx"
	case ".jpg", ".jpeg", ".png", ".tiff", ".tif", ".gif":
		return "image"
	default:
		return "unknown"
	}
}

func (p *ProcessingOrchestrator) extractPDFText(ctx context.Context, filePath string) (string, error) {
	// TODO: Implement PDF text extraction
	p.logger.Info("Extracting text from PDF", zap.String("path", filePath))
	return "[PDF text extraction not yet implemented]", nil
}

func (p *ProcessingOrchestrator) extractDocxText(ctx context.Context, filePath string) (string, error) {
	// TODO: Implement DOCX text extraction
	p.logger.Info("Extracting text from DOCX", zap.String("path", filePath))
	return "[DOCX text extraction not yet implemented]", nil
}

func (p *ProcessingOrchestrator) ocrImage(ctx context.Context, filePath string) (string, error) {
	// TODO: Integrate with mistral/client.go
	p.logger.Info("OCRing image", zap.String("path", filePath))
	return "[OCR not yet implemented]", nil
}

func (p *ProcessingOrchestrator) chunkText(text string) []string {
	// Simple chunking: split by ~256 tokens (roughly 1000 chars)
	const chunkSize = 1000
	var chunks []string

	for i := 0; i < len(text); i += chunkSize {
		end := i + chunkSize
		if end > len(text) {
			end = len(text)
		}
		chunks = append(chunks, text[i:end])
	}

	return chunks
}