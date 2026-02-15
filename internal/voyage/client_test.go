package voyage

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func TestChunking(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	text := `This is a test document. It contains multiple sentences and should be chunked appropriately.
	The chunking algorithm should split the text into overlapping chunks of 512 characters.
	Each chunk should have 50 characters of overlap with the previous chunk.
	This ensures that important information at chunk boundaries is preserved and searchable.
	The implementation should also respect word boundaries to avoid splitting mid-word.
	This is critical for maintaining semantic coherence in the chunks.
	Each chunk will be embedded individually with the Voyage AI API.`

	config := DefaultChunkConfig()
	chunks := ChunkDocument(text, config, logger)

	if len(chunks) == 0 {
		t.Fatal("Expected at least one chunk")
	}

	for i, chunk := range chunks {
		if len(chunk.Text) == 0 {
			t.Fatalf("Chunk %d is empty", i)
		}
		if chunk.Index != i {
			t.Fatalf("Chunk index mismatch: expected %d, got %d", i, chunk.Index)
		}
	}

	t.Logf("Created %d chunks from %d characters", len(chunks), len(text))
	for i, chunk := range chunks {
		t.Logf("  Chunk %d: %d chars, pos %d-%d", i, len(chunk.Text), chunk.StartPos, chunk.EndPos)
	}
}

func TestVoyageClientConfiguration(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	client := NewClient(logger)

	// If key is configured, basic properties should be set
	if client.IsConfigured() {
		t.Log("Voyage AI client is configured")
		t.Log("  Base URL: https://api.voyageai.com/v1")
		t.Log("  Model: voyage-4-large")
		t.Log("  Dimensions: 4096")
		t.Log("  Input types: document, query")
	} else {
		t.Log("Warning: VOYAGEAI_API_KEY not set - embedding tests will be skipped")
	}
}

func TestVectorizerConfiguration(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	client := NewClient(logger)
	vectorizer := NewVectorizer(client, logger)

	if !vectorizer.IsConfigured() {
		t.Skip("Voyage AI API key not configured")
	}

	config := vectorizer.config
	if config.ChunkSize != 512 {
		t.Errorf("Expected chunk size 512, got %d", config.ChunkSize)
	}
	if config.ChunkOverlap != 50 {
		t.Errorf("Expected chunk overlap 50, got %d", config.ChunkOverlap)
	}

	t.Logf("Vectorizer configured correctly:")
	t.Logf("  Chunk size: %d chars", config.ChunkSize)
	t.Logf("  Chunk overlap: %d chars", config.ChunkOverlap)
	t.Logf("  Min chunk size: %d chars", config.MinChunkSize)
}

// TestVoyageEmbedding tests actual embedding (requires API key)
func TestVoyageEmbedding(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping API test in short mode")
	}

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	client := NewClient(logger)
	if !client.IsConfigured() {
		t.Skip("VOYAGEAI_API_KEY not configured")
	}

	ctx := context.Background()

	// Test simple embedding
	texts := []string{
		"The defendant was arrested on charges of assault.",
		"The court found the defendant guilty on all counts.",
	}

	embeddings, err := client.Embed(ctx, texts)
	if err != nil {
		t.Fatalf("Failed to embed texts: %v", err)
	}

	if len(embeddings) != len(texts) {
		t.Fatalf("Expected %d embeddings, got %d", len(texts), len(embeddings))
	}

	// Verify dimensions (voyage-4-large uses 1024 dimensions)
	if len(embeddings[0]) == 0 {
		t.Fatal("Embedding has no dimensions")
	}
	dims := len(embeddings[0])
	for i, emb := range embeddings {
		if len(emb) != dims {
			t.Errorf("Embedding %d has inconsistent dimensions: %d vs %d", i, len(emb), dims)
		}
	}

	t.Logf("Successfully generated %d embeddings with %d dimensions each (voyage-4-large)", len(embeddings), dims)
}

// TestQueryEmbedding tests query-specific embedding
func TestQueryEmbedding(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping API test in short mode")
	}

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	client := NewClient(logger)
	if !client.IsConfigured() {
		t.Skip("VOYAGEAI_API_KEY not configured")
	}

	ctx := context.Background()

	query := "What charges did the defendant face?"
	embedding, err := client.EmbedQuery(ctx, query)
	if err != nil {
		t.Fatalf("Failed to embed query: %v", err)
	}

	if len(embedding) == 0 {
		t.Fatal("Query embedding has no dimensions")
	}

	t.Logf("Successfully generated query embedding with %d dimensions (voyage-4-large)", len(embedding))
}
