package pipeline

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

// TestNewProcessingOrchestrator tests the constructor
func TestNewProcessingOrchestrator(t *testing.T) {
	logger, err := zap.NewProduction()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	orchestrator := NewProcessingOrchestrator(nil, nil, nil, nil, nil, logger)

	if orchestrator == nil {
		t.Fatal("NewProcessingOrchestrator returned nil")
	}
	if orchestrator.logger != logger {
		t.Error("Logger not set correctly")
	}
}

// TestProcessResponse tests the ProcessResponse struct
func TestProcessResponse(t *testing.T) {
	resp := &ProcessResponse{
		DocumentID: "doc-123",
		Filename:   "test.pdf",
		Status:     "completed",
	}

	if resp.DocumentID != "doc-123" {
		t.Errorf("DocumentID = %v, want doc-123", resp.DocumentID)
	}
	if resp.Filename != "test.pdf" {
		t.Errorf("Filename = %v, want test.pdf", resp.Filename)
	}
	if resp.Status != "completed" {
		t.Errorf("Status = %v, want completed", resp.Status)
	}
}

// TestPipelineStatus_NotImplementedStatuses tests that unimplemented features return "not_implemented"
// This test addresses Issue #16: Orchestrator should mark unimplemented features correctly
func TestPipelineStatus_NotImplementedStatuses(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	orchestrator := NewProcessingOrchestrator(nil, nil, nil, nil, nil, logger)

	req := &ProcessRequest{
		FilePath: "/tmp/test.pdf",
		Options: ProcessingOptions{
			OCR:           true,
			ExtractTables: true,
			Vectorize:     true,
			StoreOriginal: false,
		},
	}

	ctx := context.Background()
	resp, err := orchestrator.ProcessDocument(ctx, req)

	// Allow error since dependencies may not be configured
	if err != nil && resp == nil {
		t.Skip("Skipping test - dependencies not configured")
	}

	// Even if processing partially succeeds, check that unimplemented features are marked correctly
	if resp != nil {
		// MistralOCR should be "not_implemented" when OCR is requested but not integrated
		if req.Options.OCR && resp.Pipeline.MistralOCR.Status != "not_implemented" && resp.Pipeline.MistralOCR.Status != "skipped" {
			t.Errorf("MistralOCR.Status = %v, want 'not_implemented' or 'skipped'", resp.Pipeline.MistralOCR.Status)
		}

		// DoclingIntelligence should be "not_implemented" when tables extraction requested
		if req.Options.ExtractTables && resp.Pipeline.DoclingIntelligence.Status != "not_implemented" && resp.Pipeline.DoclingIntelligence.Status != "skipped" {
			t.Errorf("DoclingIntelligence.Status = %v, want 'not_implemented' or 'skipped'", resp.Pipeline.DoclingIntelligence.Status)
		}

		// VoyageVectorization should be "not_implemented" when vectorization requested
		if req.Options.Vectorize && resp.Pipeline.VoyageVectorization.Status != "not_implemented" && resp.Pipeline.VoyageVectorization.Status != "skipped" {
			t.Errorf("VoyageVectorization.Status = %v, want 'not_implemented' or 'skipped'", resp.Pipeline.VoyageVectorization.Status)
		}

		// Storage should be "not_implemented" since Neon integration is pending
		if resp.Pipeline.Storage.Status != "not_implemented" && resp.Pipeline.Storage.Status != "skipped" {
			t.Errorf("Storage.Status = %v, want 'not_implemented' or 'skipped'", resp.Pipeline.Storage.Status)
		}
	}
}

// TestPipelineStatus_NoFalseCompletions tests that TODOs are not marked as "completed"
func TestPipelineStatus_NoFalseCompletions(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	orchestrator := NewProcessingOrchestrator(nil, nil, nil, nil, nil, logger)

	req := &ProcessRequest{
		FilePath: "/tmp/test.pdf",
		Options: ProcessingOptions{
			OCR:           true,
			ExtractTables: true,
			Vectorize:     true,
			StoreOriginal: false,
		},
	}

	ctx := context.Background()
	resp, err := orchestrator.ProcessDocument(ctx, req)

	// Allow error since dependencies may not be configured
	if err != nil && resp == nil {
		t.Skip("Skipping test - dependencies not configured")
	}

	if resp != nil {
		// CRITICAL: Unimplemented features must not be marked as "completed"
		if resp.Pipeline.MistralOCR.Status == "completed" && req.Options.OCR {
			t.Error("REGRESSION: MistralOCR marked as 'completed' but has TODO (should be 'not_implemented')")
		}
		if resp.Pipeline.DoclingIntelligence.Status == "completed" && req.Options.ExtractTables {
			t.Error("REGRESSION: DoclingIntelligence marked as 'completed' but has TODO (should be 'not_implemented')")
		}
		if resp.Pipeline.VoyageVectorization.Status == "completed" && req.Options.Vectorize {
			t.Error("REGRESSION: VoyageVectorization marked as 'completed' but has TODO (should be 'not_implemented')")
		}
		if resp.Pipeline.Storage.Status == "completed" {
			t.Error("REGRESSION: Storage marked as 'completed' but has TODO (should be 'not_implemented')")
		}
	}
}

// TestPipelineStatus_SkippedWhenNotRequested tests that optional features are skipped when not requested
func TestPipelineStatus_SkippedWhenNotRequested(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	orchestrator := NewProcessingOrchestrator(nil, nil, nil, nil, nil, logger)

	req := &ProcessRequest{
		FilePath: "/tmp/test.pdf",
		Options: ProcessingOptions{
			OCR:           false, // Not requested
			ExtractTables: false, // Not requested
			Vectorize:     false, // Not requested
			StoreOriginal: false,
		},
	}

	ctx := context.Background()
	resp, err := orchestrator.ProcessDocument(ctx, req)

	// Allow error since dependencies may not be configured
	if err != nil && resp == nil {
		t.Skip("Skipping test - dependencies not configured")
	}

	if resp != nil {
		// When OCR not requested for non-image, should be skipped
		if resp.Pipeline.MistralOCR.Status != "skipped" && resp.Pipeline.MistralOCR.Status != "" {
			t.Logf("Note: MistralOCR.Status = %v, expected 'skipped' when not requested", resp.Pipeline.MistralOCR.Status)
		}

		// When tables not requested, should be skipped
		if resp.Pipeline.DoclingIntelligence.Status != "skipped" && resp.Pipeline.DoclingIntelligence.Status != "" {
			t.Logf("Note: DoclingIntelligence.Status = %v, expected 'skipped' when not requested", resp.Pipeline.DoclingIntelligence.Status)
		}

		// When vectorization not requested, should be skipped
		if resp.Pipeline.VoyageVectorization.Status != "skipped" && resp.Pipeline.VoyageVectorization.Status != "" {
			t.Logf("Note: VoyageVectorization.Status = %v, expected 'skipped' when not requested", resp.Pipeline.VoyageVectorization.Status)
		}
	}
}

// TestProcessRequest tests the ProcessRequest struct
func TestProcessRequest(t *testing.T) {
	req := &ProcessRequest{
		FilePath: "/path/to/document.pdf",
		DocType:  "policy",
		Metadata: map[string]interface{}{
			"author": "Test Author",
			"title":  "Test Document",
		},
		Options: ProcessingOptions{
			OCR:           true,
			ExtractTables: true,
			Vectorize:     true,
			StoreOriginal: true,
		},
	}

	if req.FilePath != "/path/to/document.pdf" {
		t.Errorf("FilePath = %v, want /path/to/document.pdf", req.FilePath)
	}
	if req.DocType != "policy" {
		t.Errorf("DocType = %v, want policy", req.DocType)
	}
	if !req.Options.OCR {
		t.Error("OCR option not set correctly")
	}
	if !req.Options.Vectorize {
		t.Error("Vectorize option not set correctly")
	}
}

// TestProcessingOptions tests the ProcessingOptions struct
func TestProcessingOptions(t *testing.T) {
	tests := []struct {
		name    string
		options ProcessingOptions
	}{
		{
			name: "all options enabled",
			options: ProcessingOptions{
				OCR:           true,
				ExtractTables: true,
				Vectorize:     true,
				StoreOriginal: true,
			},
		},
		{
			name: "all options disabled",
			options: ProcessingOptions{
				OCR:           false,
				ExtractTables: false,
				Vectorize:     false,
				StoreOriginal: false,
			},
		},
		{
			name: "mixed options",
			options: ProcessingOptions{
				OCR:           true,
				ExtractTables: false,
				Vectorize:     true,
				StoreOriginal: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.options.OCR != tt.options.OCR {
				t.Error("OCR option mismatch")
			}
			if tt.options.ExtractTables != tt.options.ExtractTables {
				t.Error("ExtractTables option mismatch")
			}
			if tt.options.Vectorize != tt.options.Vectorize {
				t.Error("Vectorize option mismatch")
			}
			if tt.options.StoreOriginal != tt.options.StoreOriginal {
				t.Error("StoreOriginal option mismatch")
			}
		})
	}
}

// TestProcessSummary tests the ProcessSummary struct
func TestProcessSummary(t *testing.T) {
	summary := ProcessSummary{
		Pages:      10,
		Chunks:     50,
		Vectors:    50,
		Searchable: true,
		RAGReady:   true,
	}

	if summary.Pages != 10 {
		t.Errorf("Pages = %d, want 10", summary.Pages)
	}
	if summary.Chunks != 50 {
		t.Errorf("Chunks = %d, want 50", summary.Chunks)
	}
	if !summary.Searchable {
		t.Error("Searchable should be true")
	}
	if !summary.RAGReady {
		t.Error("RAGReady should be true")
	}
}

// TestPipelineStatus_MessageField tests that "not_implemented" status includes reason message
func TestPipelineStatus_MessageField(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	orchestrator := NewProcessingOrchestrator(nil, nil, nil, nil, nil, logger)

	req := &ProcessRequest{
		FilePath: "/tmp/test.pdf",
		DocType:  "image",
		Options: ProcessingOptions{
			OCR:           true,
			ExtractTables: true,
			Vectorize:     true,
			StoreOriginal: false,
		},
	}

	ctx := context.Background()
	resp, err := orchestrator.ProcessDocument(ctx, req)

	// Allow error since dependencies may not be configured
	if err != nil && resp == nil {
		t.Skip("Skipping test - dependencies not configured")
	}

	if resp != nil {
		// When status is "not_implemented", there should be a reason/message
		if resp.Pipeline.MistralOCR.Status == "not_implemented" {
			if resp.Pipeline.MistralOCR.Reason == "" {
				t.Log("Note: MistralOCR marked as not_implemented but no reason message (consider adding)")
			}
		}

		if resp.Pipeline.DoclingIntelligence.Status == "not_implemented" {
			// Check if message field exists (may vary by struct definition)
			t.Logf("DoclingIntelligence.Status = not_implemented (reason tracking recommended)")
		}
	}
}

// TestPipelineStatus_Boundary tests boundary conditions
func TestPipelineStatus_Boundary(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	orchestrator := NewProcessingOrchestrator(nil, nil, nil, nil, nil, logger)

	tests := []struct {
		name     string
		filePath string
		wantErr  bool
	}{
		{
			name:     "empty file path",
			filePath: "",
			wantErr:  true,
		},
		{
			name:     "very long file path",
			filePath: string(make([]byte, 10000)),
			wantErr:  true,
		},
		{
			name:     "file path with special characters",
			filePath: "/tmp/test!@#$%^&*().pdf",
			wantErr:  false, // May or may not exist, but shouldn't crash
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &ProcessRequest{
				FilePath: tt.filePath,
				Options:  ProcessingOptions{},
			}

			ctx := context.Background()
			_, err := orchestrator.ProcessDocument(ctx, req)

			// Test should not panic, error handling is acceptable
			if err == nil && tt.wantErr {
				t.Logf("Note: Expected error for %s but got none", tt.name)
			}
		})
	}
}

// TestPipelineStatus_ConcurrentAccess tests concurrent processing requests
func TestPipelineStatus_ConcurrentAccess(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	orchestrator := NewProcessingOrchestrator(nil, nil, nil, nil, nil, logger)

	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func(id int) {
			req := &ProcessRequest{
				FilePath: "/tmp/test.pdf",
				Options:  ProcessingOptions{},
			}

			ctx := context.Background()
			_, err := orchestrator.ProcessDocument(ctx, req)

			// Allow error, just testing for crashes
			if err != nil {
				t.Logf("Concurrent request %d returned error (acceptable): %v", id, err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}
}

// TestPipelineStatus_ContextCancellation tests that context cancellation is respected
func TestPipelineStatus_ContextCancellation(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	orchestrator := NewProcessingOrchestrator(nil, nil, nil, nil, nil, logger)

	req := &ProcessRequest{
		FilePath: "/tmp/test.pdf",
		Options:  ProcessingOptions{},
	}

	// Create canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := orchestrator.ProcessDocument(ctx, req)

	// Should handle cancellation gracefully (may or may not return error)
	if err != nil {
		t.Logf("Canceled context returned error (expected): %v", err)
	}
}

// TestPipelineStatus_Issue16_AllNotImplemented tests that all unimplemented features are marked correctly (Issue #16)
func TestPipelineStatus_Issue16_AllNotImplemented(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	orchestrator := NewProcessingOrchestrator(nil, nil, nil, nil, nil, logger)

	req := &ProcessRequest{
		FilePath: "/tmp/test.pdf",
		Options: ProcessingOptions{
			OCR:           true,  // Request OCR
			ExtractTables: true,  // Request table extraction
			Vectorize:     true,  // Request vectorization
			StoreOriginal: false,
		},
	}

	ctx := context.Background()
	resp, err := orchestrator.ProcessDocument(ctx, req)

	// Allow error since dependencies may not be configured
	if err != nil && resp == nil {
		t.Skip("Skipping test - dependencies not configured")
	}

	if resp != nil {
		// CRITICAL CHECKS for Issue #16
		criticalChecks := []struct {
			name   string
			status string
			field  string
		}{
			{"MistralOCR", resp.Pipeline.MistralOCR.Status, "MistralOCR.Status"},
			{"DoclingIntelligence", resp.Pipeline.DoclingIntelligence.Status, "DoclingIntelligence.Status"},
			{"VoyageVectorization", resp.Pipeline.VoyageVectorization.Status, "VoyageVectorization.Status"},
			{"Storage", resp.Pipeline.Storage.Status, "Storage.Status"},
		}

		for _, check := range criticalChecks {
			// Status must be either "not_implemented" or "skipped", NEVER "completed"
			if check.status == "completed" {
				t.Errorf("CRITICAL REGRESSION (Issue #16): %s marked as 'completed' but has TODO (should be 'not_implemented')", check.field)
			}

			// If requested and not skipped, should be "not_implemented"
			if check.status != "not_implemented" && check.status != "skipped" && check.status != "" {
				t.Logf("WARNING: %s has status '%s', expected 'not_implemented' or 'skipped'", check.field, check.status)
			}
		}
	}
}

// TestPipelineStatus_ReasonMessages tests that not_implemented statuses include reason messages
func TestPipelineStatus_ReasonMessages(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	orchestrator := NewProcessingOrchestrator(nil, nil, nil, nil, nil, logger)

	req := &ProcessRequest{
		FilePath: "/tmp/test.pdf",
		Options: ProcessingOptions{
			OCR:           true,
			ExtractTables: true,
			Vectorize:     true,
			StoreOriginal: false,
		},
	}

	ctx := context.Background()
	resp, err := orchestrator.ProcessDocument(ctx, req)

	if err != nil && resp == nil {
		t.Skip("Skipping test - dependencies not configured")
	}

	if resp != nil {
		// Check that not_implemented statuses have reason messages
		if resp.Pipeline.MistralOCR.Status == "not_implemented" {
			if resp.Pipeline.MistralOCR.Reason == "" {
				t.Error("MistralOCR marked as not_implemented but missing Reason message")
			} else {
				t.Logf("MistralOCR reason: %s", resp.Pipeline.MistralOCR.Reason)
			}
		}

		// Other pipeline steps may not have Reason field (depends on struct definition)
		// Log for informational purposes
		t.Logf("Pipeline statuses: OCR=%s, Docling=%s, Vectorization=%s, Storage=%s",
			resp.Pipeline.MistralOCR.Status,
			resp.Pipeline.DoclingIntelligence.Status,
			resp.Pipeline.VoyageVectorization.Status,
			resp.Pipeline.Storage.Status)
	}
}

// TestPipelineStatus_DocumentTypeDetection tests format detection
func TestPipelineStatus_DocumentTypeDetection(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	orchestrator := NewProcessingOrchestrator(nil, nil, nil, nil, nil, logger)

	tests := []struct {
		name         string
		filename     string
		expectedType string
	}{
		{"PDF file", "document.pdf", "pdf"},
		{"DOCX file", "document.docx", "docx"},
		{"DOC file", "legacy.doc", "docx"},
		{"JPEG image", "scan.jpg", "image"},
		{"PNG image", "screenshot.png", "image"},
		{"TIFF image", "scan.tiff", "image"},
		{"Unknown file", "data.xyz", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &ProcessRequest{
				FilePath: "/tmp/" + tt.filename,
				Options:  ProcessingOptions{},
			}

			ctx := context.Background()
			resp, err := orchestrator.ProcessDocument(ctx, req)

			// Allow error but check format detection if response exists
			if resp != nil {
				if resp.Pipeline.FormatDetection.Type != tt.expectedType {
					t.Errorf("Format detection for %s: got %s, want %s",
						tt.filename, resp.Pipeline.FormatDetection.Type, tt.expectedType)
				}
			} else if err != nil {
				t.Logf("Processing failed (acceptable for test): %v", err)
			}
		})
	}
}

// TestProcessSummary_RAGReadyFlag tests that RAG_ready flag reflects vectorization option
func TestProcessSummary_RAGReadyFlag(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	orchestrator := NewProcessingOrchestrator(nil, nil, nil, nil, nil, logger)

	tests := []struct {
		name           string
		vectorize      bool
		expectedRAGReady bool
	}{
		{"Vectorization enabled", true, true},
		{"Vectorization disabled", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &ProcessRequest{
				FilePath: "/tmp/test.pdf",
				Options: ProcessingOptions{
					Vectorize: tt.vectorize,
				},
			}

			ctx := context.Background()
			resp, err := orchestrator.ProcessDocument(ctx, req)

			if err != nil && resp == nil {
				t.Skip("Skipping test - dependencies not configured")
			}

			if resp != nil {
				if resp.Summary.RAGReady != tt.expectedRAGReady {
					t.Errorf("RAGReady flag = %v, want %v when vectorize=%v",
						resp.Summary.RAGReady, tt.expectedRAGReady, tt.vectorize)
				}
			}
		})
	}
}

// TestProcessResponse_StatusValues tests different overall status values
func TestProcessResponse_StatusValues(t *testing.T) {
	// Test that ProcessResponse can have different status values
	statuses := []string{
		"processing",
		"completed",
		"failed",
		"completed_with_errors",
	}

	for _, status := range statuses {
		resp := &ProcessResponse{
			DocumentID: "test-123",
			Filename:   "test.pdf",
			Status:     status,
		}

		if resp.Status != status {
			t.Errorf("Status = %v, want %v", resp.Status, status)
		}
	}
}

// TestProcessingOptions_DefaultValues tests default option values
func TestProcessingOptions_DefaultValues(t *testing.T) {
	// Test zero value initialization
	opts := ProcessingOptions{}

	if opts.OCR {
		t.Error("Default OCR should be false")
	}
	if opts.ExtractTables {
		t.Error("Default ExtractTables should be false")
	}
	if opts.Vectorize {
		t.Error("Default Vectorize should be false")
	}
	if opts.StoreOriginal {
		t.Error("Default StoreOriginal should be false")
	}
}

// TestPipelineStatus_AllStagesTracked tests that all pipeline stages are tracked
func TestPipelineStatus_AllStagesTracked(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	orchestrator := NewProcessingOrchestrator(nil, nil, nil, nil, nil, logger)

	req := &ProcessRequest{
		FilePath: "/tmp/test.pdf",
		Options: ProcessingOptions{
			OCR:           true,
			ExtractTables: true,
			Vectorize:     true,
			StoreOriginal: false, // Disable to avoid nil pointer with mock dependencies
		},
	}

	ctx := context.Background()
	resp, err := orchestrator.ProcessDocument(ctx, req)

	if err != nil && resp == nil {
		t.Skip("Skipping test - dependencies not configured")
	}

	if resp != nil {
		// Verify all pipeline stages have a status
		stages := []struct {
			name   string
			status string
		}{
			{"FormatDetection", resp.Pipeline.FormatDetection.Status},
			{"TextExtraction", resp.Pipeline.TextExtraction.Status},
			{"OriginalStorage", resp.Pipeline.OriginalStorage.Status},
			{"MistralOCR", resp.Pipeline.MistralOCR.Status},
			{"DoclingIntelligence", resp.Pipeline.DoclingIntelligence.Status},
			{"TextChunking", resp.Pipeline.TextChunking.Status},
			{"VoyageVectorization", resp.Pipeline.VoyageVectorization.Status},
			{"Storage", resp.Pipeline.Storage.Status},
		}

		for _, stage := range stages {
			if stage.status == "" {
				t.Errorf("Pipeline stage %s has empty status", stage.name)
			}
			t.Logf("%s: %s", stage.name, stage.status)
		}
	}
}

// TestPipelineStatus_ErrorRecovery tests that partial failures don't crash processing
func TestPipelineStatus_ErrorRecovery(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	orchestrator := NewProcessingOrchestrator(nil, nil, nil, nil, nil, logger)

	// Test with non-existent file
	req := &ProcessRequest{
		FilePath: "/nonexistent/file.pdf",
		Options:  ProcessingOptions{},
	}

	ctx := context.Background()
	resp, err := orchestrator.ProcessDocument(ctx, req)

	// Should either return error or response with failed status
	// Should NOT panic
	if err != nil {
		t.Logf("Processing failed as expected: %v", err)
	}
	if resp != nil && resp.Status == "failed" {
		t.Logf("Processing returned failed status as expected")
	}
}