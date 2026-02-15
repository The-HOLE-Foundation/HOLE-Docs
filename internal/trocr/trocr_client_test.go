package trocr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestExtractPageNumber(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected int
		wantErr  bool
	}{
		{
			name:     "ghostscript format",
			filename: "document-page-001.png",
			expected: 1,
			wantErr:  false,
		},
		{
			name:     "ghostscript multi-digit",
			filename: "brief-page-042.png",
			expected: 42,
			wantErr:  false,
		},
		{
			name:     "underscore format",
			filename: "page_001.png",
			expected: 1,
			wantErr:  false,
		},
		{
			name:     "underscore multi-digit",
			filename: "page_123.png",
			expected: 123,
			wantErr:  false,
		},
		{
			name:     "invalid format",
			filename: "somefile.png",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "no extension",
			filename: "page-001",
			expected: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractPageNumber(tt.filename)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	client := NewClient("/path/to/script.py", "printed", logger)

	assert.NotNil(t, client)
	assert.Equal(t, "/path/to/script.py", client.pythonScript)
	assert.Equal(t, "printed", client.model)
	assert.NotNil(t, client.logger)
}

// Note: ProcessBatch requires actual Python service and images
// Integration tests are in tests/integration/trocr_test.go
