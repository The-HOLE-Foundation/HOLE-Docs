package voyage

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// Vectorizer coordinates document chunking and embedding
type Vectorizer struct {
	client *Client
	logger *zap.Logger
	config ChunkConfig
}

// NewVectorizer returns a Vectorizer configured with the provided Client and zap.Logger and initialized with the default chunking configuration.
func NewVectorizer(client *Client, logger *zap.Logger) *Vectorizer {
	return &Vectorizer{
		client: client,
		logger: logger,
		config: DefaultChunkConfig(),
	}
}

// VectorizeResult contains the output of vectorization
type VectorizeResult struct {
	DocumentID      int
	ChunkCount      int
	Chunks          []*ChunkEmbedding
	TotalCharacters int
	ModelUsed       string
	Dimensions      int
}

// VectorizeDocument chunks text and generates embeddings
func (v *Vectorizer) VectorizeDocument(ctx context.Context, documentID int, text string) (*VectorizeResult, error) {
	v.logger.Info("Starting document vectorization",
		zap.Int("document_id", documentID),
		zap.Int("text_length", len(text)))

	// Step 1: Chunk the document
	chunks := ChunkDocument(text, v.config, v.logger)
	if len(chunks) == 0 {
		return nil, fmt.Errorf("failed to chunk document: no chunks created")
	}

	v.logger.Info("Document chunked",
		zap.Int("chunk_count", len(chunks)),
		zap.Int("first_chunk_size", len(chunks[0].Text)))

	// Step 2: Extract chunk texts for embedding
	chunkTexts := ExtractChunkTexts(chunks)

	// Step 3: Generate embeddings with Voyage AI
	embeddings, err := v.client.Embed(ctx, chunkTexts)
	if err != nil {
		return nil, fmt.Errorf("failed to embed chunks: %w", err)
	}

	if len(embeddings) != len(chunks) {
		return nil, fmt.Errorf("embedding count mismatch: expected %d, got %d", len(chunks), len(embeddings))
	}

	// Step 4: Pair chunks with embeddings
	chunkEmbeddings := CreateChunkEmbeddings(chunks, embeddings)

	v.logger.Info("Document vectorization complete",
		zap.Int("chunk_count", len(chunkEmbeddings)),
		zap.Int("embedding_dimensions", len(embeddings[0])),
		zap.String("model", "voyage-4-large"))

	return &VectorizeResult{
		DocumentID:      documentID,
		ChunkCount:      len(chunkEmbeddings),
		Chunks:          chunkEmbeddings,
		TotalCharacters: len(text),
		ModelUsed:       "voyage-large-4",
		Dimensions:      len(embeddings[0]),
	}, nil
}

// IsConfigured returns true if vectorization is available
func (v *Vectorizer) IsConfigured() bool {
	return v.client.IsConfigured()
}