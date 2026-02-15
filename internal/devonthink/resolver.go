package devonthink

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"go.uber.org/zap"
)

// Resolver handles DEVONthink link resolution
type Resolver struct {
	logger *zap.Logger
}

// NewResolver creates a new DEVONthink resolver
func NewResolver(logger *zap.Logger) *Resolver {
	return &Resolver{
		logger: logger,
	}
}

// ResolveLinkToPath takes a DEVONthink link and returns the file path
// Link format: x-devonthink-item://UUID
// Works with any document type (PDFs, Word docs, images, etc.)
func (r *Resolver) ResolveLinkToPath(dtLink string) (string, error) {
	r.logger.Info("Resolving DEVONthink link", zap.String("link", dtLink))

	// Check if this is actually a file path (not a DT link)
	if !strings.HasPrefix(dtLink, "x-devonthink-item://") {
		// It's already a file path, return as-is
		r.logger.Info("Input is already a file path", zap.String("path", dtLink))
		return dtLink, nil
	}

	// Extract UUID from the link
	uuid := strings.TrimPrefix(dtLink, "x-devonthink-item://")
	uuid = strings.TrimSpace(uuid)

	if uuid == "" {
		return "", fmt.Errorf("invalid DEVONthink link format: %s", dtLink)
	}

	// AppleScript to query DEVONthink and get the file path
	applescript := fmt.Sprintf(`
tell application "DEVONthink 3"
	try
		set theRecord to get record with uuid "%s"
		if theRecord is not missing value then
			set filePath to path of theRecord
			return filePath
		else
			error "Record not found"
		end if
	on error errMsg
		error errMsg
	end try
end tell
	`, uuid)

	// Execute the AppleScript
	cmd := exec.Command("osascript", "-e", applescript)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		r.logger.Error("Failed to resolve DEVONthink link",
			zap.String("uuid", uuid),
			zap.Error(err),
			zap.String("stderr", stderr.String()))
		return "", fmt.Errorf("failed to resolve DEVONthink link: %v - %s", err, stderr.String())
	}

	filePath := strings.TrimSpace(stdout.String())
	if filePath == "" {
		return "", fmt.Errorf("no file path returned from DEVONthink for UUID: %s", uuid)
	}

	r.logger.Info("DEVONthink link resolved successfully",
		zap.String("uuid", uuid),
		zap.String("path", filePath))

	return filePath, nil
}

// ResolveLinksToPaths takes multiple DEVONthink links and returns file paths
// Supports both DEVONthink links and regular file paths (mixed input)
func (r *Resolver) ResolveLinksToPaths(dtLinks []string) ([]string, error) {
	r.logger.Info("Resolving multiple DEVONthink links", zap.Int("count", len(dtLinks)))

	var paths []string

	for i, link := range dtLinks {
		path, err := r.ResolveLinkToPath(link)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve link at index %d (%s): %v", i, link, err)
		}
		paths = append(paths, path)
	}

	r.logger.Info("All links resolved successfully", zap.Int("count", len(paths)))
	return paths, nil
}

// IsDEVONthinkLink checks if a string is a DEVONthink link
func IsDEVONthinkLink(str string) bool {
	return strings.HasPrefix(str, "x-devonthink-item://")
}

// ExtractUUID extracts the UUID from a DEVONthink link
func ExtractUUID(dtLink string) string {
	uuid := strings.TrimPrefix(dtLink, "x-devonthink-item://")
	return strings.TrimSpace(uuid)
}
