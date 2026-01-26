#!/bin/bash
# Example usage of jira-cli comment commands
# This demonstrates comprehensive comment management operations

# Exit on error
set -e

ISSUE_KEY="PROJ-123"

echo "=== Jira CLI Comment Management Examples ==="
echo

# Example 1: Add a comment (legacy syntax - backward compatible)
echo "1. Adding a comment (legacy syntax)..."
jira-cli comment "$ISSUE_KEY" "This is a comment using the legacy command"
echo

# Example 2: Add a comment (new syntax)
echo "2. Adding a comment (new syntax)..."
jira-cli comments add "$ISSUE_KEY" "This is a comment using the new subcommand"
echo

# Example 3: Add a comment with JSON output
echo "3. Adding a comment with JSON output..."
jira-cli comments add "$ISSUE_KEY" "Another comment for testing" --json
echo

# Example 4: List all comments
echo "4. Listing all comments..."
jira-cli comments list "$ISSUE_KEY"
echo

# Example 5: List comments with limit
echo "5. Listing last 5 comments..."
jira-cli comments list "$ISSUE_KEY" --limit 5
echo

# Example 6: List comments in reverse order (newest first)
echo "6. Listing comments in reverse order..."
jira-cli comments list "$ISSUE_KEY" --order -created
echo

# Example 7: Get a specific comment
echo "7. Getting a specific comment..."
# Replace 10001 with an actual comment ID from your issue
COMMENT_ID="10001"
jira-cli comments get "$ISSUE_KEY" "$COMMENT_ID"
echo

# Example 8: Get a specific comment as JSON
echo "8. Getting a specific comment as JSON..."
jira-cli comments get "$ISSUE_KEY" "$COMMENT_ID" --json
echo

# Example 9: Update a comment
echo "9. Updating a comment..."
jira-cli comments update "$ISSUE_KEY" "$COMMENT_ID" "Updated comment text with corrections"
echo

# Example 10: Update a comment with JSON output
echo "10. Updating a comment with JSON output..."
jira-cli comments update "$ISSUE_KEY" "$COMMENT_ID" "Final version of the comment" --json
echo

# Example 11: Delete a comment (requires confirmation)
echo "11. Deleting a comment..."
# Uncomment the following line to actually delete (be careful!)
# jira-cli comments delete "$ISSUE_KEY" "$COMMENT_ID" --confirm
echo "(Commented out for safety - uncomment to test deletion)"
echo

# Example 12: Multi-line comment (using heredoc)
echo "12. Adding a multi-line comment..."
COMMENT_TEXT=$(cat <<EOF
This is a multi-line comment.

It can contain:
- Bullet points
- Multiple paragraphs
- Code snippets

And more!
EOF
)
jira-cli comments add "$ISSUE_KEY" "$COMMENT_TEXT"
echo

# Example 13: Scripting workflow - add comment and verify
echo "13. Scripting workflow - add and verify..."
RESPONSE=$(jira-cli comments add "$ISSUE_KEY" "Automated comment from script" --json)
NEW_COMMENT_ID=$(echo "$RESPONSE" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "Created comment ID: $NEW_COMMENT_ID"
jira-cli comments get "$ISSUE_KEY" "$NEW_COMMENT_ID"
echo

echo "=== Examples Complete ==="
