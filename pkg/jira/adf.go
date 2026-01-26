package jira

import (
	"fmt"
	"strings"
)

// ADFToPlainText converts Atlassian Document Format (ADF) to plain text.
// ADF is Jira's rich text format returned by API v3 for description, comments, etc.
//
// Supported node types:
//   - doc: Root document node
//   - paragraph: Text paragraphs
//   - heading: h1-h6 headers
//   - codeBlock: Code blocks with optional language
//   - blockquote: Quoted text
//   - bulletList / orderedList: Lists with listItem children
//   - listItem: Individual list items
//   - text: Text content with optional marks (bold, italic, etc.)
//   - hardBreak: Line breaks
//   - mediaSingle / mediaGroup: Images and media (displayed as placeholders)
//   - rule: Horizontal rule
//   - table, tableRow, tableHeader, tableCell: Table structures
//
// Example ADF input:
//
//	{
//	  "type": "doc",
//	  "version": 1,
//	  "content": [
//	    {"type": "paragraph", "content": [{"type": "text", "text": "Hello"}]}
//	  ]
//	}
//
// Output: "Hello"
func ADFToPlainText(adf interface{}) string {
	if adf == nil {
		return ""
	}

	// Handle string input (already plain text)
	if str, ok := adf.(string); ok {
		return str
	}

	// Handle ADF document structure
	adfMap, ok := adf.(map[string]interface{})
	if !ok {
		return ""
	}

	var result strings.Builder
	processADFNode(&result, adfMap, 0, 0)
	return strings.TrimRight(result.String(), "\n")
}

// processADFNode recursively processes an ADF node and writes to the result
// Parameters:
//   - result: StringBuilder to write output to
//   - node: The ADF node to process
//   - depth: Nesting depth (for lists)
//   - listIndex: Current item index for ordered lists (0 for unordered)
func processADFNode(result *strings.Builder, node map[string]interface{}, depth int, listIndex int) {
	nodeType, _ := node["type"].(string)

	switch nodeType {
	case "doc":
		processChildren(result, node, depth)

	case "paragraph":
		processChildren(result, node, depth)
		result.WriteString("\n")

	case "heading":
		// Extract heading level for potential future formatting
		processChildren(result, node, depth)
		result.WriteString("\n")

	case "text":
		text, _ := node["text"].(string)
		result.WriteString(text)

	case "hardBreak":
		result.WriteString("\n")

	case "codeBlock":
		// Add code block with optional language indicator
		if lang, ok := node["attrs"].(map[string]interface{})["language"].(string); ok && lang != "" {
			result.WriteString(fmt.Sprintf("[%s]\n", lang))
		}
		processChildren(result, node, depth)
		result.WriteString("\n")

	case "blockquote":
		// Process blockquote content with ">" prefix
		var quoteBuf strings.Builder
		processChildren(&quoteBuf, node, depth)
		lines := strings.Split(strings.TrimRight(quoteBuf.String(), "\n"), "\n")
		for _, line := range lines {
			result.WriteString("> ")
			result.WriteString(line)
			result.WriteString("\n")
		}

	case "bulletList":
		processListItems(result, node, depth, false)

	case "orderedList":
		processListItems(result, node, depth, true)

	case "listItem":
		processChildren(result, node, depth)

	case "mediaSingle", "mediaGroup":
		// Handle images and media - show as placeholder
		processMediaNode(result, node)

	case "rule":
		result.WriteString("---\n")

	case "table":
		processTableNode(result, node)

	case "inlineCard":
		// Handle smart links (URLs embedded as cards)
		if attrs, ok := node["attrs"].(map[string]interface{}); ok {
			if url, ok := attrs["url"].(string); ok {
				result.WriteString(url)
			}
		}

	case "mention":
		// Handle @mentions
		if attrs, ok := node["attrs"].(map[string]interface{}); ok {
			if text, ok := attrs["text"].(string); ok {
				result.WriteString(text)
			}
		}

	case "emoji":
		// Handle emoji shortnames
		if attrs, ok := node["attrs"].(map[string]interface{}); ok {
			if shortName, ok := attrs["shortName"].(string); ok {
				result.WriteString(shortName)
			}
		}

	default:
		// For unknown types, try to process children if they exist
		processChildren(result, node, depth)
	}
}

// processChildren processes all child nodes in the content array
func processChildren(result *strings.Builder, node map[string]interface{}, depth int) {
	content, ok := node["content"].([]interface{})
	if !ok {
		return
	}

	for _, item := range content {
		if itemMap, ok := item.(map[string]interface{}); ok {
			processADFNode(result, itemMap, depth, 0)
		}
	}
}

// processListItems handles bullet and ordered lists
func processListItems(result *strings.Builder, node map[string]interface{}, depth int, ordered bool) {
	content, ok := node["content"].([]interface{})
	if !ok {
		return
	}

	indent := strings.Repeat("  ", depth)

	for i, item := range content {
		if itemMap, ok := item.(map[string]interface{}); ok {
			// Create list marker
			var marker string
			if ordered {
				marker = fmt.Sprintf("%d. ", i+1)
			} else {
				marker = "• "
			}

			result.WriteString(indent)
			result.WriteString(marker)

			// Process list item content
			var itemBuf strings.Builder
			if itemContent, ok := itemMap["content"].([]interface{}); ok {
				for _, child := range itemContent {
					if childMap, ok := child.(map[string]interface{}); ok {
						childType, _ := childMap["type"].(string)

						// Handle nested lists
						if childType == "bulletList" || childType == "orderedList" {
							// First, add any text content accumulated
							if itemBuf.Len() > 0 {
								result.WriteString(strings.TrimRight(itemBuf.String(), "\n"))
								result.WriteString("\n")
								itemBuf.Reset()
							}
							// Process nested list with increased depth
							processADFNode(result, childMap, depth+1, 0)
						} else {
							processADFNode(&itemBuf, childMap, depth+1, 0)
						}
					}
				}
			}

			// Write remaining item content
			if itemBuf.Len() > 0 {
				text := strings.TrimRight(itemBuf.String(), "\n")
				// Replace internal newlines with proper indentation
				text = strings.ReplaceAll(text, "\n", "\n"+indent+"  ")
				result.WriteString(text)
				result.WriteString("\n")
			}
		}
	}
}

// processMediaNode handles media elements (images, files)
func processMediaNode(result *strings.Builder, node map[string]interface{}) {
	content, ok := node["content"].([]interface{})
	if !ok {
		result.WriteString("[Media]\n")
		return
	}

	for _, item := range content {
		if itemMap, ok := item.(map[string]interface{}); ok {
			nodeType, _ := itemMap["type"].(string)
			if nodeType == "media" {
				attrs, _ := itemMap["attrs"].(map[string]interface{})
				mediaType, _ := attrs["type"].(string)

				switch mediaType {
				case "file":
					// For files, try to get filename from alt or fallback
					if alt, ok := attrs["alt"].(string); ok && alt != "" {
						result.WriteString(fmt.Sprintf("[File: %s]\n", alt))
					} else {
						result.WriteString("[File]\n")
					}
				case "external":
					if url, ok := attrs["url"].(string); ok {
						result.WriteString(fmt.Sprintf("[Image: %s]\n", url))
					} else {
						result.WriteString("[Image]\n")
					}
				default:
					result.WriteString("[Media]\n")
				}
			}
		}
	}
}

// processTableNode handles table structures
func processTableNode(result *strings.Builder, node map[string]interface{}) {
	content, ok := node["content"].([]interface{})
	if !ok {
		return
	}

	for _, row := range content {
		if rowMap, ok := row.(map[string]interface{}); ok {
			processTableRow(result, rowMap)
		}
	}
	result.WriteString("\n")
}

// processTableRow processes a single table row
func processTableRow(result *strings.Builder, node map[string]interface{}) {
	content, ok := node["content"].([]interface{})
	if !ok {
		return
	}

	cells := make([]string, 0)
	for _, cell := range content {
		if cellMap, ok := cell.(map[string]interface{}); ok {
			var cellBuf strings.Builder
			processChildren(&cellBuf, cellMap, 0)
			cellText := strings.TrimSpace(cellBuf.String())
			cellText = strings.ReplaceAll(cellText, "\n", " ")
			cells = append(cells, cellText)
		}
	}

	result.WriteString("| ")
	result.WriteString(strings.Join(cells, " | "))
	result.WriteString(" |\n")
}
