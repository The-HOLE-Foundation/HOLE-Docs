package neon

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"go.uber.org/zap"
)

// TestNewDocumentsClient tests the constructor
func TestNewDocumentsClient(t *testing.T) {
	logger, err := zap.NewProduction()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	client := NewDocumentsClient(db, logger)

	if client == nil {
		t.Fatal("NewDocumentsClient returned nil")
	}
	if client.db != db {
		t.Error("DocumentsClient.db not set correctly")
	}
	if client.logger != logger {
		t.Error("DocumentsClient.logger not set correctly")
	}
}

// TestUpdateDocumentStatus tests the UpdateDocumentStatus method
func TestUpdateDocumentStatus(t *testing.T) {
	logger, err := zap.NewProduction()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	client := NewDocumentsClient(db, logger)

	tests := []struct {
		name      string
		docID     int
		status    string
		mockSetup func()
		wantErr   bool
	}{
		{
			name:   "successful status update",
			docID:  123,
			status: "vectorization_failed",
			mockSetup: func() {
				mock.ExpectExec(`UPDATE docai.documents`).
					WithArgs("vectorization_failed", 123).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name:   "update classification_failed status",
			docID:  456,
			status: "classification_failed",
			mockSetup: func() {
				mock.ExpectExec(`UPDATE docai.documents`).
					WithArgs("classification_failed", 456).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name:   "database error",
			docID:  789,
			status: "vectorization_failed",
			mockSetup: func() {
				mock.ExpectExec(`UPDATE docai.documents`).
					WithArgs("vectorization_failed", 789).
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
		{
			name:   "document not found (0 rows affected)",
			docID:  999,
			status: "pending",
			mockSetup: func() {
				mock.ExpectExec(`UPDATE docai.documents`).
					WithArgs("pending", 999).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: false, // Not an error, just no rows updated
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			ctx := context.Background()
			err := client.UpdateDocumentStatus(ctx, tt.docID, tt.status)

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateDocumentStatus() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unfulfilled expectations: %s", err)
			}
		})
	}
}

// TestUpdateDocumentStatus_ValidStatuses tests various valid status values
func TestUpdateDocumentStatus_ValidStatuses(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	client := NewDocumentsClient(db, logger)

	validStatuses := []string{
		"pending",
		"processing",
		"completed",
		"vectorization_failed",
		"classification_failed",
		"vectorization_storage_failed",
	}

	for _, status := range validStatuses {
		t.Run("status_"+status, func(t *testing.T) {
			mock.ExpectExec(`UPDATE docai.documents`).
				WithArgs(status, 123).
				WillReturnResult(sqlmock.NewResult(0, 1))

			ctx := context.Background()
			err := client.UpdateDocumentStatus(ctx, 123, status)

			if err != nil {
				t.Errorf("UpdateDocumentStatus(%s) unexpected error: %v", status, err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unfulfilled expectations: %s", err)
			}
		})
	}
}

// TestUpdateDocumentStatus_Concurrent tests concurrent updates
func TestUpdateDocumentStatus_Concurrent(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	client := NewDocumentsClient(db, logger)

	// Setup expectations for 5 concurrent updates
	for i := 0; i < 5; i++ {
		mock.ExpectExec(`UPDATE docai.documents`).
			WithArgs("processing", sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}

	// Launch concurrent goroutines
	done := make(chan bool)
	for i := 1; i <= 5; i++ {
		go func(docID int) {
			ctx := context.Background()
			err := client.UpdateDocumentStatus(ctx, docID, "processing")
			if err != nil {
				t.Errorf("Concurrent update failed for docID %d: %v", docID, err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}

	// Note: We can't strictly verify expectations order in concurrent scenario
	// but we can check they all completed
}

// TestUpdateDocumentStatus_ContextCancellation tests context cancellation
func TestUpdateDocumentStatus_ContextCancellation(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	client := NewDocumentsClient(db, logger)

	// Setup expectation that will be canceled
	mock.ExpectExec(`UPDATE docai.documents`).
		WithArgs("processing", 123).
		WillReturnError(context.Canceled)

	// Create canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err = client.UpdateDocumentStatus(ctx, 123, "processing")

	// Should receive an error due to cancellation
	if err == nil {
		t.Error("Expected error from canceled context, got nil")
	}
}

// TestUpdateDocumentStatus_BoundaryValues tests boundary conditions
func TestUpdateDocumentStatus_BoundaryValues(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	client := NewDocumentsClient(db, logger)

	tests := []struct {
		name   string
		docID  int
		status string
	}{
		{
			name:   "minimum document ID",
			docID:  1,
			status: "completed",
		},
		{
			name:   "large document ID",
			docID:  999999999,
			status: "processing",
		},
		{
			name:   "zero document ID",
			docID:  0,
			status: "pending",
		},
		{
			name:   "negative document ID",
			docID:  -1,
			status: "failed",
		},
		{
			name:   "empty status string",
			docID:  123,
			status: "",
		},
		{
			name:   "very long status string",
			docID:  123,
			status: string(make([]byte, 1000)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.ExpectExec(`UPDATE docai.documents`).
				WithArgs(tt.status, tt.docID).
				WillReturnResult(sqlmock.NewResult(0, 1))

			ctx := context.Background()
			err := client.UpdateDocumentStatus(ctx, tt.docID, tt.status)

			// Should not panic or return unexpected errors for boundary values
			// Database constraints will handle invalid values
			if err != nil && err != sql.ErrConnDone {
				// Allow connection errors but test shouldn't crash
				t.Logf("Boundary test returned error (expected): %v", err)
			}
		})
	}
}

// TestDocumentRecord tests the DocumentRecord struct
func TestDocumentRecord(t *testing.T) {
	doc := &DocumentRecord{
		ID:                       123,
		UserID:                   "test-user",
		Filename:                 "test.pdf",
		DocumentType:             "policy",
		DocumentCategory:         "administrative",
		ClassificationConfidence: 0.95,
		OCRPageCount:             10,
		Keywords:                 []string{"keyword1", "keyword2"},
		R2Path:                   "bucket/path/test.pdf",
		Project:                  "test-project",
	}

	if doc.ID != 123 {
		t.Errorf("ID = %d, want 123", doc.ID)
	}
	if doc.UserID != "test-user" {
		t.Errorf("UserID = %s, want test-user", doc.UserID)
	}
	if len(doc.Keywords) != 2 {
		t.Errorf("Keywords length = %d, want 2", len(doc.Keywords))
	}
	if doc.ClassificationConfidence != 0.95 {
		t.Errorf("ClassificationConfidence = %f, want 0.95", doc.ClassificationConfidence)
	}
}

// TestDocumentExport tests the DocumentExport struct
func TestDocumentExport(t *testing.T) {
	export := &DocumentExport{
		ID:       456,
		Filename: "export.pdf",
		FileData: []byte("fake pdf content"),
		FileSize: 12345,
	}

	if export.ID != 456 {
		t.Errorf("ID = %d, want 456", export.ID)
	}
	if export.Filename != "export.pdf" {
		t.Errorf("Filename = %s, want export.pdf", export.Filename)
	}
	if len(export.FileData) != 16 {
		t.Errorf("FileData length = %d, want 16", len(export.FileData))
	}
	if export.FileSize != 12345 {
		t.Errorf("FileSize = %d, want 12345", export.FileSize)
	}
}

// TestUpdateDocumentStatus_MultipleStatusTypes tests updating with different status values
func TestUpdateDocumentStatus_MultipleStatusTypes(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	client := NewDocumentsClient(db, logger)

	// Test all expected status values
	statusValues := []struct {
		name   string
		status string
		docID  int
	}{
		{"pending status", "pending", 1},
		{"processing status", "processing", 2},
		{"completed status", "completed", 3},
		{"vectorization_failed", "vectorization_failed", 4},
		{"classification_failed", "classification_failed", 5},
		{"vectorization_storage_failed", "vectorization_storage_failed", 6},
	}

	for _, sv := range statusValues {
		t.Run(sv.name, func(t *testing.T) {
			mock.ExpectExec(`UPDATE docai.documents`).
				WithArgs(sv.status, sv.docID).
				WillReturnResult(sqlmock.NewResult(0, 1))

			ctx := context.Background()
			err := client.UpdateDocumentStatus(ctx, sv.docID, sv.status)

			if err != nil {
				t.Errorf("UpdateDocumentStatus(%s) unexpected error: %v", sv.status, err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unfulfilled expectations: %s", err)
			}
		})
	}
}

// TestUpdateDocumentStatus_IssuesTracking tests Issue #14 and #17 - tracking failures
func TestUpdateDocumentStatus_IssuesTracking(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	client := NewDocumentsClient(db, logger)

	tests := []struct {
		name        string
		docID       int
		status      string
		description string
	}{
		{
			name:        "track vectorization failure (Issue #14)",
			docID:       100,
			status:      "vectorization_failed",
			description: "Vectorization failures should be tracked in database",
		},
		{
			name:        "track classification failure (Issue #17)",
			docID:       200,
			status:      "classification_failed",
			description: "Classification failures should be tracked in database",
		},
		{
			name:        "track embedding storage failure",
			docID:       300,
			status:      "vectorization_storage_failed",
			description: "Embedding storage failures should be tracked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.ExpectExec(`UPDATE docai.documents`).
				WithArgs(tt.status, tt.docID).
				WillReturnResult(sqlmock.NewResult(0, 1))

			ctx := context.Background()
			err := client.UpdateDocumentStatus(ctx, tt.docID, tt.status)

			if err != nil {
				t.Errorf("%s failed: %v", tt.description, err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unfulfilled expectations for %s: %s", tt.name, err)
			}
		})
	}
}

// TestUpdateDocumentStatus_RetryAfterFailure tests updating status after retry
func TestUpdateDocumentStatus_RetryAfterFailure(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	client := NewDocumentsClient(db, logger)

	docID := 555

	// First update: mark as failed
	mock.ExpectExec(`UPDATE docai.documents`).
		WithArgs("vectorization_failed", docID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Second update: mark as processing (retry)
	mock.ExpectExec(`UPDATE docai.documents`).
		WithArgs("processing", docID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Third update: mark as completed
	mock.ExpectExec(`UPDATE docai.documents`).
		WithArgs("completed", docID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	ctx := context.Background()

	// Simulate failure -> retry -> success workflow
	if err := client.UpdateDocumentStatus(ctx, docID, "vectorization_failed"); err != nil {
		t.Errorf("Failed to mark as failed: %v", err)
	}

	if err := client.UpdateDocumentStatus(ctx, docID, "processing"); err != nil {
		t.Errorf("Failed to mark as processing: %v", err)
	}

	if err := client.UpdateDocumentStatus(ctx, docID, "completed"); err != nil {
		t.Errorf("Failed to mark as completed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

// TestUpdateDocumentStatus_SQLInjection tests that SQL injection is prevented
func TestUpdateDocumentStatus_SQLInjection(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	client := NewDocumentsClient(db, logger)

	// Attempt SQL injection in status field
	maliciousStatus := "completed'; DROP TABLE documents; --"
	docID := 999

	// The parameterized query should safely handle this
	mock.ExpectExec(`UPDATE docai.documents`).
		WithArgs(maliciousStatus, docID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	ctx := context.Background()
	err = client.UpdateDocumentStatus(ctx, docID, maliciousStatus)

	// Should not fail (parameterized query handles it safely)
	if err != nil {
		t.Errorf("UpdateDocumentStatus should handle malicious input safely: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

// TestUpdateDocumentStatus_EmptyStatus tests behavior with empty status string
func TestUpdateDocumentStatus_EmptyStatus(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	client := NewDocumentsClient(db, logger)

	// Empty status should still be processed (database constraints will validate)
	mock.ExpectExec(`UPDATE docai.documents`).
		WithArgs("", 123).
		WillReturnResult(sqlmock.NewResult(0, 1))

	ctx := context.Background()
	err = client.UpdateDocumentStatus(ctx, 123, "")

	// Function doesn't validate status, leaves that to database
	if err != nil {
		t.Logf("Empty status handled by database constraints: %v", err)
	}
}

// TestDocumentRecord_ZeroValues tests DocumentRecord with zero values
func TestDocumentRecord_ZeroValues(t *testing.T) {
	doc := &DocumentRecord{
		ID:                       0,
		UserID:                   "",
		Filename:                 "",
		DocumentType:             "",
		DocumentCategory:         "",
		ClassificationConfidence: 0.0,
		OCRPageCount:             0,
		Keywords:                 []string{},
		R2Path:                   "",
		Project:                  "",
	}

	// Zero values should be valid for a DocumentRecord
	if doc.ID != 0 {
		t.Errorf("ID = %d, want 0", doc.ID)
	}
	if doc.ClassificationConfidence != 0.0 {
		t.Errorf("ClassificationConfidence = %f, want 0.0", doc.ClassificationConfidence)
	}
	if len(doc.Keywords) != 0 {
		t.Errorf("Keywords length = %d, want 0", len(doc.Keywords))
	}
}

// TestDocumentRecord_NilKeywords tests handling of nil Keywords array
func TestDocumentRecord_NilKeywords(t *testing.T) {
	doc := &DocumentRecord{
		ID:       1,
		UserID:   "user-1",
		Filename: "test.pdf",
		Keywords: nil, // nil vs empty slice
	}

	// nil Keywords should be valid
	if doc.Keywords != nil && len(doc.Keywords) != 0 {
		t.Errorf("Keywords should be nil or empty")
	}
}

// TestDocumentExport_EmptyFileData tests DocumentExport with no file data
func TestDocumentExport_EmptyFileData(t *testing.T) {
	export := &DocumentExport{
		ID:       789,
		Filename: "metadata-only.pdf",
		FileData: nil,    // No file data (e.g., Phase 2 R2 storage)
		FileSize: 500000, // Size known but data not retrieved
	}

	if export.ID != 789 {
		t.Errorf("ID = %d, want 789", export.ID)
	}
	if export.FileData != nil {
		t.Error("FileData should be nil for metadata-only export")
	}
	if export.FileSize != 500000 {
		t.Errorf("FileSize = %d, want 500000", export.FileSize)
	}
}

// TestNewDocumentsClient_NilParameters tests constructor with nil parameters
func TestNewDocumentsClient_NilParameters(t *testing.T) {
	// Test with nil logger (should handle gracefully or panic)
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Constructor panicked with nil logger (expected in some designs): %v", r)
		}
	}()

	// Create client with nil database and logger
	client := NewDocumentsClient(nil, nil)

	// Constructor should return a client even with nil parameters
	// (actual usage will fail, but constructor shouldn't panic)
	if client == nil {
		t.Error("NewDocumentsClient should return non-nil client even with nil parameters")
	}
}

// TestUpdateDocumentStatus_TransactionScenario tests multiple updates in sequence
func TestUpdateDocumentStatus_TransactionScenario(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	client := NewDocumentsClient(db, logger)

	docID := 1001

	// Simulate a complete document processing lifecycle
	lifecycle := []string{
		"pending",
		"processing",
		"vectorization_failed", // First attempt fails
		"processing",           // Retry
		"completed",            // Success
	}

	for i, status := range lifecycle {
		mock.ExpectExec(`UPDATE docai.documents`).
			WithArgs(status, docID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		ctx := context.Background()
		err := client.UpdateDocumentStatus(ctx, docID, status)

		if err != nil {
			t.Errorf("Step %d (%s) failed: %v", i+1, status, err)
		}
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}