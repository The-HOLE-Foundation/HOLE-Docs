package devonthink

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"go.uber.org/zap"
)

// SearchResult represents a DEVONthink search result
type SearchResult struct {
	UUID string
	Name string
	Path string
	Link string
}

// SearchDocuments searches DEVONthink and returns matching documents with their links
func (r *Resolver) SearchDocuments(query string, limit int) ([]SearchResult, error) {
	r.logger.Info("Searching DEVONthink", zap.String("query", query), zap.Int("limit", limit))

	if limit <= 0 {
		limit = 10
	}

	applescript := fmt.Sprintf(`
tell application "DEVONthink 3"
	set searchResults to search "%s" in current database
	set resultList to {}
	set counter to 0
	repeat with theRecord in searchResults
		if counter >= %d then exit repeat
		set counter to counter + 1
		set theUUID to uuid of theRecord
		set theName to name of theRecord
		set thePath to path of theRecord
		set end of resultList to theUUID & "|" & theName & "|" & thePath
	end repeat
	return resultList as text
end tell
	`, query, limit)

	cmd := exec.Command("osascript", "-e", applescript)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		r.logger.Error("Failed to search DEVONthink",
			zap.String("query", query),
			zap.Error(err),
			zap.String("stderr", stderr.String()))
		return nil, fmt.Errorf("failed to search DEVONthink: %v - %s", err, stderr.String())
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return []SearchResult{}, nil
	}

	// Parse results
	lines := strings.Split(output, ", ")
	var results []SearchResult

	for _, line := range lines {
		parts := strings.Split(line, "|")
		if len(parts) != 3 {
			continue
		}

		result := SearchResult{
			UUID: strings.TrimSpace(parts[0]),
			Name: strings.TrimSpace(parts[1]),
			Path: strings.TrimSpace(parts[2]),
			Link: "x-devonthink-item://" + strings.TrimSpace(parts[0]),
		}
		results = append(results, result)
	}

	r.logger.Info("Search completed", zap.Int("results", len(results)))
	return results, nil
}
