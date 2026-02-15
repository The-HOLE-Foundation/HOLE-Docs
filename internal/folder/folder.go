package folder

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/The-HOLE-Foundation/hole-docs/internal/naturalsort"
)

// CollectOptions controls how files are collected from folders
type CollectOptions struct {
	Recursive         bool      // Scan subdirectories
	NaturalSort       bool      // Use human-friendly sorting
	Pattern           string    // Glob pattern (e.g., "*.pdf", "*")
	ExcludePatterns   []string  // Patterns to exclude
	PreferredExts     []string  // Preferred file extensions (optional filter)
	AllowedExts       []string  // Only allow these extensions (optional filter)
	DryRun            bool      // Don't actually collect, just plan
}

// CollectedFiles holds the results of folder collection
type CollectedFiles struct {
	Count           int
	TotalSize       int64
	Files           []string // Sorted file paths
	ExcludedCount   int
	ExcludedSize    int64
	DryRun          bool
}

// CollectFilesFromFolder collects files from a folder with optional filtering
func CollectFilesFromFolder(folderPath string, opts *CollectOptions) (*CollectedFiles, error) {
	if opts == nil {
		opts = &CollectOptions{
			Recursive:   false,
			NaturalSort: true,
			Pattern:     "*",
		}
	}

	// Verify folder exists
	info, err := os.Stat(folderPath)
	if err != nil {
		return nil, fmt.Errorf("folder not found: %v", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", folderPath)
	}

	files := []string{}
	excludedCount := 0
	excludedSize := int64(0)
	totalSize := int64(0)

	// Compile exclude patterns
	var excludeRegexes []*regexp.Regexp
	for _, pattern := range opts.ExcludePatterns {
		if regex, err := regexp.Compile(pattern); err == nil {
			excludeRegexes = append(excludeRegexes, regex)
		}
	}

	// Walk the directory
	walkFunc := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Get relative path from folder
		relPath, _ := filepath.Rel(folderPath, path)

		// Check recursion depth
		if !opts.Recursive && strings.Count(relPath, string(os.PathSeparator)) > 0 {
			return nil
		}

		// Check if matches pattern
		matches, _ := filepath.Match(opts.Pattern, filepath.Base(path))
		if !matches {
			return nil
		}

		// Check exclude patterns
		excluded := false
		for _, regex := range excludeRegexes {
			if regex.MatchString(relPath) {
				excluded = true
				break
			}
		}
		if excluded {
			if fileInfo, err := os.Stat(path); err == nil {
				excludedCount++
				excludedSize += fileInfo.Size()
			}
			return nil
		}

		// Check file extension filters
		if len(opts.AllowedExts) > 0 {
			ext := strings.ToLower(filepath.Ext(path))
			allowed := false
			for _, allowed_ext := range opts.AllowedExts {
				if ext == allowed_ext || ext == "."+allowed_ext {
					allowed = true
					break
				}
			}
			if !allowed {
				return nil
			}
		}

		files = append(files, path)
		if fileInfo, err := os.Stat(path); err == nil {
			totalSize += fileInfo.Size()
		}

		return nil
	}

	// Use WalkDir for efficient directory traversal
	if opts.Recursive {
		err := filepath.WalkDir(folderPath, walkFunc)
		if err != nil {
			return nil, fmt.Errorf("error walking directory: %v", err)
		}
	} else {
		// Non-recursive: only read immediate directory
		entries, err := os.ReadDir(folderPath)
		if err != nil {
			return nil, fmt.Errorf("error reading directory: %v", err)
		}
		for _, entry := range entries {
			path := filepath.Join(folderPath, entry.Name())
			if err := walkFunc(path, entry, nil); err != nil {
				return nil, fmt.Errorf("error processing file %s: %v", path, err)
			}
		}
	}

	// Sort files
	if opts.NaturalSort {
		files = naturalsort.SortPathsWithBasename(files)
	} else {
		// Standard lexicographic sort
		sort.Strings(files)
	}

	return &CollectedFiles{
		Count:         len(files),
		TotalSize:     totalSize,
		Files:         files,
		ExcludedCount: excludedCount,
		ExcludedSize:  excludedSize,
		DryRun:        opts.DryRun,
	}, nil
}

// GetPDFFilesFromFolder is a convenience function for collecting PDF files
func GetPDFFilesFromFolder(folderPath string, recursive bool) (*CollectedFiles, error) {
	return CollectFilesFromFolder(folderPath, &CollectOptions{
		Recursive:   recursive,
		NaturalSort: true,
		Pattern:     "*.pdf",
		AllowedExts: []string{".pdf"},
	})
}

// GetAllFilesFromFolder is a convenience function for collecting all files
func GetAllFilesFromFolder(folderPath string, recursive bool) (*CollectedFiles, error) {
	return CollectFilesFromFolder(folderPath, &CollectOptions{
		Recursive:   recursive,
		NaturalSort: true,
		Pattern:     "*",
	})
}

// IsGlobPattern checks if a string looks like a glob pattern
func IsGlobPattern(path string) bool {
	return strings.Contains(path, "*") || strings.Contains(path, "?") || strings.Contains(path, "[")
}

// IsFolderPath checks if a path is a directory
func IsFolderPath(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// ExpandPath handles glob patterns and folder paths
// Returns either a glob match or folder contents
func ExpandPath(pathArg string, opts *CollectOptions) ([]string, error) {
	// If it's a glob pattern
	if IsGlobPattern(pathArg) {
		matches, err := filepath.Glob(pathArg)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern: %v", err)
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("no files matched pattern: %s", pathArg)
		}
		// Apply natural sort if requested
		if opts != nil && opts.NaturalSort {
			matches = naturalsort.SortPaths(matches)
		}
		return matches, nil
	}

	// If it's a folder
	if IsFolderPath(pathArg) {
		collected, err := CollectFilesFromFolder(pathArg, opts)
		if err != nil {
			return nil, err
		}
		return collected.Files, nil
	}

	// Single file
	if _, err := os.Stat(pathArg); err == nil {
		return []string{pathArg}, nil
	}

	return nil, fmt.Errorf("path not found or not accessible: %s", pathArg)
}
