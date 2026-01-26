#!/bin/bash
# Example usage of jira-cli attachment commands
# This demonstrates comprehensive file attachment management operations

# Exit on error
set -e

ISSUE_KEY="PROJ-123"

echo "=== Jira CLI Attachment Management Examples ==="
echo

# Create sample files for testing
echo "Creating sample test files..."
mkdir -p /tmp/jira-cli-test
echo "This is a test document" > /tmp/jira-cli-test/test-document.txt
echo "Sample README content" > /tmp/jira-cli-test/README.md
dd if=/dev/zero of=/tmp/jira-cli-test/large-file.bin bs=1M count=2 2>/dev/null
echo "Created test files in /tmp/jira-cli-test/"
echo

# Example 1: Upload a single file
echo "1. Uploading a single file..."
jira-cli attachment upload "$ISSUE_KEY" /tmp/jira-cli-test/test-document.txt
echo

# Example 2: Upload multiple files
echo "2. Uploading multiple files..."
jira-cli attachment upload "$ISSUE_KEY" \
  /tmp/jira-cli-test/test-document.txt \
  /tmp/jira-cli-test/README.md
echo

# Example 3: Upload a large file with progress bar
echo "3. Uploading a large file (with progress)..."
jira-cli attachment upload "$ISSUE_KEY" /tmp/jira-cli-test/large-file.bin
echo

# Example 4: Upload without progress bar
echo "4. Uploading without progress bar..."
jira-cli attachment upload "$ISSUE_KEY" /tmp/jira-cli-test/test-document.txt --no-progress
echo

# Example 5: List all attachments
echo "5. Listing all attachments..."
jira-cli attachment list "$ISSUE_KEY"
echo

# Example 6: List attachments as JSON
echo "6. Listing attachments as JSON..."
jira-cli attachment list "$ISSUE_KEY" --json
echo

# Example 7: Download attachment by filename
echo "7. Downloading attachment by filename..."
jira-cli attachment download "$ISSUE_KEY" test-document.txt
echo "Downloaded to current directory"
echo

# Example 8: Download attachment by ID
echo "8. Downloading attachment by ID..."
# Replace 10001 with an actual attachment ID from your issue
ATTACHMENT_ID="10001"
jira-cli attachment download "$ISSUE_KEY" "$ATTACHMENT_ID"
echo

# Example 9: Download to specific directory
echo "9. Downloading to specific directory..."
mkdir -p /tmp/jira-cli-downloads
jira-cli attachment download "$ISSUE_KEY" test-document.txt --output /tmp/jira-cli-downloads/
echo "Downloaded to /tmp/jira-cli-downloads/"
echo

# Example 10: Download with custom filename
echo "10. Downloading with custom filename..."
jira-cli attachment download "$ISSUE_KEY" test-document.txt --output /tmp/custom-name.txt
echo "Downloaded as /tmp/custom-name.txt"
echo

# Example 11: Download large file with progress
echo "11. Downloading large file (with progress)..."
jira-cli attachment download "$ISSUE_KEY" large-file.bin --output /tmp/downloaded-large-file.bin
echo

# Example 12: Download without progress bar
echo "12. Downloading without progress bar..."
jira-cli attachment download "$ISSUE_KEY" README.md --output /tmp/readme.md --no-progress
echo

# Example 13: Delete attachment (requires confirmation)
echo "13. Deleting an attachment..."
# Uncomment the following line to actually delete (be careful!)
# jira-cli attachment delete "$ATTACHMENT_ID" --confirm
echo "(Commented out for safety - uncomment to test deletion)"
echo

# Example 14: Delete with JSON output
echo "14. Deleting with JSON output..."
# Uncomment to test
# jira-cli attachment delete "$ATTACHMENT_ID" --confirm --json
echo "(Commented out for safety)"
echo

# Example 15: Scripting workflow - upload and verify
echo "15. Scripting workflow - upload and verify..."
UPLOAD_RESPONSE=$(jira-cli attachment upload "$ISSUE_KEY" /tmp/jira-cli-test/test-document.txt --json 2>/dev/null || echo '{}')
echo "Upload response: $UPLOAD_RESPONSE"

# List attachments to verify upload
echo "Verifying upload..."
jira-cli attachment list "$ISSUE_KEY"
echo

# Example 16: Batch upload workflow
echo "16. Batch upload workflow..."
for file in /tmp/jira-cli-test/*.txt; do
  echo "Uploading $(basename "$file")..."
  jira-cli attachment upload "$ISSUE_KEY" "$file" --no-progress
done
echo

# Example 17: Find and download specific attachment
echo "17. Find and download specific attachment..."
# Get list of attachments as JSON, find specific file, download it
ATTACHMENTS=$(jira-cli attachment list "$ISSUE_KEY" --json)
echo "Found attachments:"
echo "$ATTACHMENTS" | grep -o '"filename":"[^"]*"' || echo "No attachments found"
echo

# Example 18: Download all attachments from an issue
echo "18. Download all attachments from an issue..."
mkdir -p /tmp/all-attachments
FILENAMES=$(jira-cli attachment list "$ISSUE_KEY" --json | grep -o '"filename":"[^"]*"' | cut -d'"' -f4 || true)
for filename in $FILENAMES; do
  echo "Downloading $filename..."
  jira-cli attachment download "$ISSUE_KEY" "$filename" --output /tmp/all-attachments/ --no-progress || true
done
echo "All attachments downloaded to /tmp/all-attachments/"
echo

# Cleanup
echo "Cleaning up test files..."
rm -rf /tmp/jira-cli-test
# Optionally clean up downloads
# rm -rf /tmp/jira-cli-downloads /tmp/custom-name.txt /tmp/downloaded-large-file.bin /tmp/readme.md /tmp/all-attachments

echo "=== Examples Complete ==="
echo "Note: Some commands are commented out for safety. Uncomment carefully when testing."
