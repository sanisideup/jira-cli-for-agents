#!/bin/bash
# Phase 3 Demo Script
# Demonstrates template management and validation features

set -e

echo "======================================"
echo "Phase 3: Templates & Validation Demo"
echo "======================================"
echo ""

# Build the CLI
echo "1. Building jira-cli..."
go build -o jira-cli main.go
echo "✓ Build complete"
echo ""

# Show template help
echo "2. Template command help:"
./jira-cli template --help
echo ""

# Initialize templates
echo "3. Initializing templates..."
./jira-cli template init
echo ""

# List templates
echo "4. Listing available templates:"
./jira-cli template list
echo ""

# List templates in JSON
echo "5. Listing templates (JSON format):"
./jira-cli template list --json
echo ""

# Show epic template
echo "6. Showing epic template:"
./jira-cli template show epic
echo ""

# Show story template
echo "7. Showing story template:"
./jira-cli template show story
echo ""

# Run all tests
echo "8. Running Phase 3 tests..."
echo ""
echo "Template service tests:"
go test ./pkg/template -v
echo ""
echo "Metadata service tests:"
go test ./pkg/jira -v -run TestMetadata
echo ""
echo "Fields service tests:"
go test ./cmd -v -run TestFields
echo ""

echo "======================================"
echo "Phase 3 Demo Complete! ✅"
echo "======================================"
echo ""
echo "Summary:"
echo "  ✓ Metadata service with validation"
echo "  ✓ Template system with rendering"
echo "  ✓ 4 default templates (epic, story, bug, charter)"
echo "  ✓ Template management commands"
echo "  ✓ All tests passing"
echo ""
echo "Templates location: ~/.jira-cli/templates/"
echo "Ready for Phase 4: Issue Creation"
