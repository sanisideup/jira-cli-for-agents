package jira

import (
	"strings"
	"testing"
)

func TestADFToPlainText_NilInput(t *testing.T) {
	result := ADFToPlainText(nil)
	if result != "" {
		t.Errorf("Expected empty string for nil input, got: %q", result)
	}
}

func TestADFToPlainText_StringInput(t *testing.T) {
	input := "Already plain text"
	result := ADFToPlainText(input)
	if result != input {
		t.Errorf("Expected %q, got: %q", input, result)
	}
}

func TestADFToPlainText_SimpleParagraph(t *testing.T) {
	adf := map[string]interface{}{
		"type":    "doc",
		"version": 1,
		"content": []interface{}{
			map[string]interface{}{
				"type": "paragraph",
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "Hello world",
					},
				},
			},
		},
	}

	result := ADFToPlainText(adf)
	expected := "Hello world"
	if result != expected {
		t.Errorf("Expected %q, got: %q", expected, result)
	}
}

func TestADFToPlainText_MultipleParagraphs(t *testing.T) {
	adf := map[string]interface{}{
		"type":    "doc",
		"version": 1,
		"content": []interface{}{
			map[string]interface{}{
				"type": "paragraph",
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "First paragraph",
					},
				},
			},
			map[string]interface{}{
				"type": "paragraph",
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "Second paragraph",
					},
				},
			},
		},
	}

	result := ADFToPlainText(adf)
	if !strings.Contains(result, "First paragraph") || !strings.Contains(result, "Second paragraph") {
		t.Errorf("Expected both paragraphs, got: %q", result)
	}
}

func TestADFToPlainText_TextWithMarks(t *testing.T) {
	// Bold and italic marks are stripped, only text content is preserved
	adf := map[string]interface{}{
		"type":    "doc",
		"version": 1,
		"content": []interface{}{
			map[string]interface{}{
				"type": "paragraph",
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "Normal ",
					},
					map[string]interface{}{
						"type": "text",
						"text": "bold",
						"marks": []interface{}{
							map[string]interface{}{"type": "strong"},
						},
					},
					map[string]interface{}{
						"type": "text",
						"text": " text",
					},
				},
			},
		},
	}

	result := ADFToPlainText(adf)
	expected := "Normal bold text"
	if result != expected {
		t.Errorf("Expected %q, got: %q", expected, result)
	}
}

func TestADFToPlainText_CodeBlock(t *testing.T) {
	adf := map[string]interface{}{
		"type":    "doc",
		"version": 1,
		"content": []interface{}{
			map[string]interface{}{
				"type": "codeBlock",
				"attrs": map[string]interface{}{
					"language": "json",
				},
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "{ \"key\": \"value\" }",
					},
				},
			},
		},
	}

	result := ADFToPlainText(adf)
	if !strings.Contains(result, "[json]") {
		t.Errorf("Expected language indicator [json], got: %q", result)
	}
	if !strings.Contains(result, "{ \"key\": \"value\" }") {
		t.Errorf("Expected code content, got: %q", result)
	}
}

func TestADFToPlainText_BulletList(t *testing.T) {
	adf := map[string]interface{}{
		"type":    "doc",
		"version": 1,
		"content": []interface{}{
			map[string]interface{}{
				"type": "bulletList",
				"content": []interface{}{
					map[string]interface{}{
						"type": "listItem",
						"content": []interface{}{
							map[string]interface{}{
								"type": "paragraph",
								"content": []interface{}{
									map[string]interface{}{
										"type": "text",
										"text": "Item one",
									},
								},
							},
						},
					},
					map[string]interface{}{
						"type": "listItem",
						"content": []interface{}{
							map[string]interface{}{
								"type": "paragraph",
								"content": []interface{}{
									map[string]interface{}{
										"type": "text",
										"text": "Item two",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	result := ADFToPlainText(adf)
	if !strings.Contains(result, "•") {
		t.Errorf("Expected bullet marker, got: %q", result)
	}
	if !strings.Contains(result, "Item one") || !strings.Contains(result, "Item two") {
		t.Errorf("Expected list items, got: %q", result)
	}
}

func TestADFToPlainText_OrderedList(t *testing.T) {
	adf := map[string]interface{}{
		"type":    "doc",
		"version": 1,
		"content": []interface{}{
			map[string]interface{}{
				"type": "orderedList",
				"content": []interface{}{
					map[string]interface{}{
						"type": "listItem",
						"content": []interface{}{
							map[string]interface{}{
								"type": "paragraph",
								"content": []interface{}{
									map[string]interface{}{
										"type": "text",
										"text": "First",
									},
								},
							},
						},
					},
					map[string]interface{}{
						"type": "listItem",
						"content": []interface{}{
							map[string]interface{}{
								"type": "paragraph",
								"content": []interface{}{
									map[string]interface{}{
										"type": "text",
										"text": "Second",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	result := ADFToPlainText(adf)
	if !strings.Contains(result, "1.") || !strings.Contains(result, "2.") {
		t.Errorf("Expected numbered markers, got: %q", result)
	}
	if !strings.Contains(result, "First") || !strings.Contains(result, "Second") {
		t.Errorf("Expected list items, got: %q", result)
	}
}

func TestADFToPlainText_Heading(t *testing.T) {
	adf := map[string]interface{}{
		"type":    "doc",
		"version": 1,
		"content": []interface{}{
			map[string]interface{}{
				"type": "heading",
				"attrs": map[string]interface{}{
					"level": 1,
				},
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "My Heading",
					},
				},
			},
		},
	}

	result := ADFToPlainText(adf)
	expected := "My Heading"
	if result != expected {
		t.Errorf("Expected %q, got: %q", expected, result)
	}
}

func TestADFToPlainText_Blockquote(t *testing.T) {
	adf := map[string]interface{}{
		"type":    "doc",
		"version": 1,
		"content": []interface{}{
			map[string]interface{}{
				"type": "blockquote",
				"content": []interface{}{
					map[string]interface{}{
						"type": "paragraph",
						"content": []interface{}{
							map[string]interface{}{
								"type": "text",
								"text": "Quoted text",
							},
						},
					},
				},
			},
		},
	}

	result := ADFToPlainText(adf)
	if !strings.Contains(result, ">") {
		t.Errorf("Expected blockquote marker >, got: %q", result)
	}
	if !strings.Contains(result, "Quoted text") {
		t.Errorf("Expected quoted text, got: %q", result)
	}
}

func TestADFToPlainText_HardBreak(t *testing.T) {
	adf := map[string]interface{}{
		"type":    "doc",
		"version": 1,
		"content": []interface{}{
			map[string]interface{}{
				"type": "paragraph",
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "Line one",
					},
					map[string]interface{}{
						"type": "hardBreak",
					},
					map[string]interface{}{
						"type": "text",
						"text": "Line two",
					},
				},
			},
		},
	}

	result := ADFToPlainText(adf)
	if !strings.Contains(result, "Line one\nLine two") {
		t.Errorf("Expected line break between lines, got: %q", result)
	}
}

func TestADFToPlainText_Rule(t *testing.T) {
	adf := map[string]interface{}{
		"type":    "doc",
		"version": 1,
		"content": []interface{}{
			map[string]interface{}{
				"type": "paragraph",
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "Before",
					},
				},
			},
			map[string]interface{}{
				"type": "rule",
			},
			map[string]interface{}{
				"type": "paragraph",
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "After",
					},
				},
			},
		},
	}

	result := ADFToPlainText(adf)
	if !strings.Contains(result, "---") {
		t.Errorf("Expected horizontal rule ---, got: %q", result)
	}
}

func TestADFToPlainText_InlineCard(t *testing.T) {
	adf := map[string]interface{}{
		"type":    "doc",
		"version": 1,
		"content": []interface{}{
			map[string]interface{}{
				"type": "paragraph",
				"content": []interface{}{
					map[string]interface{}{
						"type": "inlineCard",
						"attrs": map[string]interface{}{
							"url": "https://example.com",
						},
					},
				},
			},
		},
	}

	result := ADFToPlainText(adf)
	if !strings.Contains(result, "https://example.com") {
		t.Errorf("Expected URL from inline card, got: %q", result)
	}
}

func TestADFToPlainText_Mention(t *testing.T) {
	adf := map[string]interface{}{
		"type":    "doc",
		"version": 1,
		"content": []interface{}{
			map[string]interface{}{
				"type": "paragraph",
				"content": []interface{}{
					map[string]interface{}{
						"type": "mention",
						"attrs": map[string]interface{}{
							"text": "@john.doe",
						},
					},
				},
			},
		},
	}

	result := ADFToPlainText(adf)
	if !strings.Contains(result, "@john.doe") {
		t.Errorf("Expected mention text, got: %q", result)
	}
}

func TestADFToPlainText_Emoji(t *testing.T) {
	adf := map[string]interface{}{
		"type":    "doc",
		"version": 1,
		"content": []interface{}{
			map[string]interface{}{
				"type": "paragraph",
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "Great job ",
					},
					map[string]interface{}{
						"type": "emoji",
						"attrs": map[string]interface{}{
							"shortName": ":thumbsup:",
						},
					},
				},
			},
		},
	}

	result := ADFToPlainText(adf)
	if !strings.Contains(result, ":thumbsup:") {
		t.Errorf("Expected emoji shortname, got: %q", result)
	}
}

func TestADFToPlainText_Table(t *testing.T) {
	adf := map[string]interface{}{
		"type":    "doc",
		"version": 1,
		"content": []interface{}{
			map[string]interface{}{
				"type": "table",
				"content": []interface{}{
					map[string]interface{}{
						"type": "tableRow",
						"content": []interface{}{
							map[string]interface{}{
								"type": "tableHeader",
								"content": []interface{}{
									map[string]interface{}{
										"type": "paragraph",
										"content": []interface{}{
											map[string]interface{}{
												"type": "text",
												"text": "Header 1",
											},
										},
									},
								},
							},
							map[string]interface{}{
								"type": "tableHeader",
								"content": []interface{}{
									map[string]interface{}{
										"type": "paragraph",
										"content": []interface{}{
											map[string]interface{}{
												"type": "text",
												"text": "Header 2",
											},
										},
									},
								},
							},
						},
					},
					map[string]interface{}{
						"type": "tableRow",
						"content": []interface{}{
							map[string]interface{}{
								"type": "tableCell",
								"content": []interface{}{
									map[string]interface{}{
										"type": "paragraph",
										"content": []interface{}{
											map[string]interface{}{
												"type": "text",
												"text": "Cell 1",
											},
										},
									},
								},
							},
							map[string]interface{}{
								"type": "tableCell",
								"content": []interface{}{
									map[string]interface{}{
										"type": "paragraph",
										"content": []interface{}{
											map[string]interface{}{
												"type": "text",
												"text": "Cell 2",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	result := ADFToPlainText(adf)
	if !strings.Contains(result, "|") {
		t.Errorf("Expected table with pipe separators, got: %q", result)
	}
	if !strings.Contains(result, "Header 1") || !strings.Contains(result, "Header 2") {
		t.Errorf("Expected table headers, got: %q", result)
	}
	if !strings.Contains(result, "Cell 1") || !strings.Contains(result, "Cell 2") {
		t.Errorf("Expected table cells, got: %q", result)
	}
}

func TestADFToPlainText_ComplexDocument(t *testing.T) {
	// Simulates a real Jira description with mixed content
	adf := map[string]interface{}{
		"type":    "doc",
		"version": 1,
		"content": []interface{}{
			map[string]interface{}{
				"type": "paragraph",
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "For now we're using the Reviews API with a 6 month timeframe with this format:",
					},
				},
			},
			map[string]interface{}{
				"type": "codeBlock",
				"attrs": map[string]interface{}{
					"language": "json",
				},
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "{\n  \"average\": 4.25,\n  \"total_reviews\": 128\n}",
					},
				},
			},
			map[string]interface{}{
				"type": "paragraph",
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "We need to update this to fetch lifetime data instead.",
					},
				},
			},
		},
	}

	result := ADFToPlainText(adf)

	// Check for key content
	if !strings.Contains(result, "Reviews API") {
		t.Errorf("Expected 'Reviews API', got: %q", result)
	}
	if !strings.Contains(result, "[json]") {
		t.Errorf("Expected '[json]' language marker, got: %q", result)
	}
	if !strings.Contains(result, "average") {
		t.Errorf("Expected code content, got: %q", result)
	}
	if !strings.Contains(result, "lifetime data") {
		t.Errorf("Expected final paragraph, got: %q", result)
	}
}

func TestADFToPlainText_MediaSingle(t *testing.T) {
	adf := map[string]interface{}{
		"type":    "doc",
		"version": 1,
		"content": []interface{}{
			map[string]interface{}{
				"type": "mediaSingle",
				"content": []interface{}{
					map[string]interface{}{
						"type": "media",
						"attrs": map[string]interface{}{
							"type": "file",
							"alt":  "screenshot.png",
						},
					},
				},
			},
		},
	}

	result := ADFToPlainText(adf)
	if !strings.Contains(result, "[File: screenshot.png]") {
		t.Errorf("Expected file placeholder, got: %q", result)
	}
}

func TestADFToPlainText_InvalidInput(t *testing.T) {
	// Test with various invalid inputs
	tests := []struct {
		name  string
		input interface{}
	}{
		{"integer", 42},
		{"float", 3.14},
		{"boolean", true},
		{"slice", []string{"a", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ADFToPlainText(tt.input)
			if result != "" {
				t.Errorf("Expected empty string for %s input, got: %q", tt.name, result)
			}
		})
	}
}
