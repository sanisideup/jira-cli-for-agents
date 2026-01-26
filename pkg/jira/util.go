package jira

import (
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FormatFileSize converts bytes to human-readable format
// Examples: 1.2 MB, 456 KB, 3.4 GB
func FormatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KB", "MB", "GB", "TB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

// FormatDate converts ISO8601 to readable format
// Example: 2024-01-15T10:30:00.000+0000 -> 2024-01-15 10:30
func FormatDate(iso8601 string) string {
	// Try multiple common Jira date formats
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05.000-0700",
		"2006-01-02T15:04:05.000+0000",
		"2006-01-02T15:04:05.999Z",
	}

	var t time.Time
	var err error
	for _, format := range formats {
		t, err = time.Parse(format, iso8601)
		if err == nil {
			break
		}
	}

	if err != nil {
		// If parsing fails, return original string
		return iso8601
	}

	return t.Format("2006-01-02 15:04")
}

// ExtractPlainText converts ADF (Atlassian Document Format) to plain text for preview
// This is a simplified implementation that extracts text content
func ExtractPlainText(adf interface{}) string {
	if adf == nil {
		return ""
	}

	// Handle ADF document structure
	adfMap, ok := adf.(map[string]interface{})
	if !ok {
		// If it's already a string, return it
		if str, ok := adf.(string); ok {
			return str
		}
		return ""
	}

	var result strings.Builder

	// Recursively extract text from content array
	if content, ok := adfMap["content"].([]interface{}); ok {
		for _, item := range content {
			if itemMap, ok := item.(map[string]interface{}); ok {
				result.WriteString(extractTextFromNode(itemMap))
			}
		}
	}

	return strings.TrimSpace(result.String())
}

// extractTextFromNode recursively extracts text from an ADF node
func extractTextFromNode(node map[string]interface{}) string {
	var result strings.Builder

	// Check node type
	nodeType, _ := node["type"].(string)

	// If it's a text node, return its text
	if nodeType == "text" {
		if text, ok := node["text"].(string); ok {
			result.WriteString(text)
		}
	}

	// Recursively process content
	if content, ok := node["content"].([]interface{}); ok {
		for _, item := range content {
			if itemMap, ok := item.(map[string]interface{}); ok {
				result.WriteString(extractTextFromNode(itemMap))
			}
		}
	}

	// Add newline after paragraphs
	if nodeType == "paragraph" {
		result.WriteString("\n")
	}

	return result.String()
}

// ValidateFilePath checks if file exists and is readable
func ValidateFilePath(path string) error {
	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file '%s' not found", path)
		}
		return fmt.Errorf("failed to access file '%s': %w", path, err)
	}

	// Check if it's a regular file
	if info.IsDir() {
		return fmt.Errorf("'%s' is a directory, not a file", path)
	}

	// Check if file is readable
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("file '%s' is not readable: %w", path, err)
	}
	file.Close()

	return nil
}

// GetMimeType determines MIME type from file extension
func GetMimeType(filename string) string {
	ext := filepath.Ext(filename)
	if ext == "" {
		return "application/octet-stream"
	}

	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		return "application/octet-stream"
	}

	return mimeType
}
