package voyage

import (
	"strings"

	"go.uber.org/zap"
)

// ChunkConfig defines how documents are chunked
type ChunkConfig struct {
	ChunkSize     int // Characters per chunk
	ChunkOverlap  int // Overlap between chunks in characters
	MinChunkSize  int // Minimum chunk size to avoid tiny chunks
}

// DefaultChunkConfig returns a ChunkConfig populated with recommended defaults for document chunking.
// The defaults set ChunkSize to 512, ChunkOverlap to 50, and MinChunkSize to 100.
func DefaultChunkConfig() ChunkConfig {
	return ChunkConfig{
		ChunkSize:    512,  // Per DOCUMENT_INTELLIGENCE_SCHEMA.sql
		ChunkOverlap: 50,   // Per DOCUMENT_INTELLIGENCE_SCHEMA.sql
		MinChunkSize: 100,  // Don't create chunks smaller than this
	}
}

// Chunk represents a document chunk ready for embedding
type Chunk struct {
	Index     int    // Chunk number (0-indexed)
	Text      string // Chunk text
	StartPos  int    // Starting position in original text
	EndPos    int    // Ending position in original text
	Metadata  map[string]interface{}
}

// ChunkDocument splits text into overlapping chunks optimized for semantic search.
// 
// For an empty input it returns an empty slice. If the entire text length is less
// than or equal to config.ChunkSize a single chunk containing the full text is
// returned. Otherwise the function advances by stepSize = ChunkSize - ChunkOverlap
// to produce overlapping chunks. When possible it moves chunk end boundaries left
// up to 50 characters to align to a word boundary while ensuring each chunk is
// at least config.MinChunkSize characters. Each produced chunk is trimmed of
// surrounding whitespace and populated with Index, StartPos, EndPos and a
// Metadata entry "chunk_number". The function logs a summary of the chunking
// operation using the provided logger.
func ChunkDocument(text string, config ChunkConfig, logger *zap.Logger) []*Chunk {
	if text == "" {
		return []*Chunk{}
	}

	// Validate chunk configuration to prevent infinite loops
	if config.ChunkSize <= 0 {
		logger.Error("Invalid chunk size",
			zap.Int("chunk_size", config.ChunkSize))
		return []*Chunk{}
	}
	if config.ChunkOverlap < 0 {
		logger.Error("Invalid chunk overlap - must be non-negative",
			zap.Int("chunk_overlap", config.ChunkOverlap))
		return []*Chunk{}
	}
	if config.ChunkOverlap >= config.ChunkSize {
		logger.Error("Invalid chunk overlap - must be less than chunk size",
			zap.Int("chunk_overlap", config.ChunkOverlap),
			zap.Int("chunk_size", config.ChunkSize))
		return []*Chunk{}
	}

	var chunks []*Chunk
	textLen := len(text)

	// If text fits in one chunk, don't split
	if textLen <= config.ChunkSize {
		chunks = append(chunks, &Chunk{
			Index:    0,
			Text:     text,
			StartPos: 0,
			EndPos:   textLen,
			Metadata: map[string]interface{}{},
		})
		logger.Info("Document chunking complete",
			zap.Int("chunks", len(chunks)),
			zap.Int("total_chars", textLen))
		return chunks
	}

	// Calculate step size (non-overlapping portion per chunk)
	stepSize := config.ChunkSize - config.ChunkOverlap

	chunkIndex := 0
	pos := 0

	for pos < textLen {
		// Calculate chunk boundaries
		chunkStart := pos
		chunkEnd := pos + config.ChunkSize

		// Don't exceed text length
		if chunkEnd > textLen {
			chunkEnd = textLen
		}

		// Try to split at word boundary to avoid cutting mid-word
		if chunkEnd < textLen && chunkEnd < len(text) {
			// Look back up to 50 chars for a space
			for i := 0; i < 50 && chunkEnd > chunkStart+config.MinChunkSize; i++ {
				if chunkEnd-1 >= 0 && text[chunkEnd-1] == ' ' {
					break
				}
				chunkEnd--
			}
		}

		// Ensure chunk is large enough
		if chunkEnd-chunkStart >= config.MinChunkSize {
			chunkText := strings.TrimSpace(text[chunkStart:chunkEnd])

			chunks = append(chunks, &Chunk{
				Index:    chunkIndex,
				Text:     chunkText,
				StartPos: chunkStart,
				EndPos:   chunkEnd,
				Metadata: map[string]interface{}{
					"chunk_number": chunkIndex,
				},
			})

			chunkIndex++
		}

		// Move position for next chunk (with overlap)
		pos += stepSize
	}

	logger.Info("Document chunking complete",
		zap.Int("chunks", len(chunks)),
		zap.Int("total_chars", textLen),
		zap.Int("chunk_size", config.ChunkSize),
		zap.Int("chunk_overlap", config.ChunkOverlap))

	return chunks
}

// ExtractChunkTexts extracts the Text field from each Chunk in the input slice.
// The returned slice preserves the input order and has the same length; an empty input yields an empty slice.
func ExtractChunkTexts(chunks []*Chunk) []string {
	texts := make([]string, len(chunks))
	for i, chunk := range chunks {
		texts[i] = chunk.Text
	}
	return texts
}

// CreateChunkEmbeddings pairs each Chunk with its corresponding embedding vector and returns a slice of ChunkEmbedding pointers in the same order.
// If the number of chunks and embeddings differ, it returns nil.
func CreateChunkEmbeddings(chunks []*Chunk, embeddings [][]float32) []*ChunkEmbedding {
	if len(chunks) != len(embeddings) {
		return nil
	}

	result := make([]*ChunkEmbedding, len(chunks))
	for i, chunk := range chunks {
		result[i] = &ChunkEmbedding{
			Chunk:     chunk,
			Embedding: embeddings[i],
		}
	}
	return result
}

// ChunkEmbedding pairs a chunk with its vector embedding
type ChunkEmbedding struct {
	Chunk     *Chunk
	Embedding []float32
}

// Dimensions returns the embedding dimension
func (ce *ChunkEmbedding) Dimensions() int {
	return len(ce.Embedding)
}