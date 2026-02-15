package folder

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsGlobPattern(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
		name     string
	}{
		{"/path/to/*.pdf", true, "Glob with asterisk"},
		{"/path/to/file[0-9].pdf", true, "Glob with bracket"},
		{"/path/to/file?.pdf", true, "Glob with question mark"},
		{"/path/to/file.pdf", false, "Regular file path"},
		{"file.pdf", false, "Single filename"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsGlobPattern(tt.path)
			if result != tt.expected {
				t.Errorf("IsGlobPattern(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestIsFolderPath(t *testing.T) {
	// Create temporary test directory
	tempDir := t.TempDir()

	tests := []struct {
		path     string
		expected bool
		name     string
	}{
		{tempDir, true, "Existing directory"},
		{filepath.Join(tempDir, "nonexistent"), false, "Non-existent directory"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsFolderPath(tt.path)
			if result != tt.expected {
				t.Errorf("IsFolderPath(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestCollectFilesFromFolder(t *testing.T) {
	// Create temporary directory with test files
	tempDir := t.TempDir()

	// Create test PDF files with various names
	testFiles := []string{
		"a3.1.pdf",
		"a3.2.pdf",
		"a3.10.pdf",
		"archive.pdf",
		"beehive.pdf",
	}

	for _, f := range testFiles {
		filepath := filepath.Join(tempDir, f)
		if err := os.WriteFile(filepath, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	t.Run("Natural sort collection", func(t *testing.T) {
		collected, err := CollectFilesFromFolder(tempDir, &CollectOptions{
			Recursive:   false,
			NaturalSort: true,
			Pattern:     "*.pdf",
		})

		if err != nil {
			t.Fatalf("CollectFilesFromFolder failed: %v", err)
		}

		if collected.Count != len(testFiles) {
			t.Errorf("Expected %d files, got %d", len(testFiles), collected.Count)
		}

		// Verify natural sort order
		expected := []string{"a3.1", "a3.2", "a3.10", "archive", "beehive"}
		for i, file := range collected.Files {
			basename := filepath.Base(file)
			basename = basename[:len(basename)-4] // Remove .pdf extension
			if basename != expected[i] {
				t.Errorf("Sort order mismatch at position %d: got %q, want %q", i, basename, expected[i])
			}
		}
	})

	t.Run("Lexicographic sort collection", func(t *testing.T) {
		collected, err := CollectFilesFromFolder(tempDir, &CollectOptions{
			Recursive:   false,
			NaturalSort: false,
			Pattern:     "*.pdf",
		})

		if err != nil {
			t.Fatalf("CollectFilesFromFolder failed: %v", err)
		}

		if collected.Count != len(testFiles) {
			t.Errorf("Expected %d files, got %d", len(testFiles), collected.Count)
		}

		// Lexicographic order would be different (a3.1, a3.10, a3.2, archive, beehive)
		// Just verify they're sorted consistently
		for i := 0; i < len(collected.Files)-1; i++ {
			if collected.Files[i] > collected.Files[i+1] {
				t.Errorf("Files not in lexicographic order: %v", collected.Files)
				break
			}
		}
	})

	t.Run("PDF extension filter", func(t *testing.T) {
		// Create a non-PDF file
		nonPDFPath := filepath.Join(tempDir, "readme.txt")
		if err := os.WriteFile(nonPDFPath, []byte("text file"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		collected, err := CollectFilesFromFolder(tempDir, &CollectOptions{
			Recursive:   false,
			NaturalSort: true,
			Pattern:     "*",
			AllowedExts: []string{".pdf"},
		})

		if err != nil {
			t.Fatalf("CollectFilesFromFolder failed: %v", err)
		}

		if collected.Count != len(testFiles) {
			t.Errorf("Expected %d PDF files (excluding .txt), got %d", len(testFiles), collected.Count)
		}

		// Verify no .txt files in results
		for _, f := range collected.Files {
			if filepath.Ext(f) == ".txt" {
				t.Errorf("Non-PDF file found in results: %v", f)
			}
		}
	})
}

func TestGetPDFFilesFromFolder(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	testFiles := []string{"doc1.pdf", "doc2.pdf", "readme.txt"}
	for _, f := range testFiles {
		filepath := filepath.Join(tempDir, f)
		if err := os.WriteFile(filepath, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	collected, err := GetPDFFilesFromFolder(tempDir, false)
	if err != nil {
		t.Fatalf("GetPDFFilesFromFolder failed: %v", err)
	}

	if collected.Count != 2 {
		t.Errorf("Expected 2 PDF files, got %d", collected.Count)
	}
}
