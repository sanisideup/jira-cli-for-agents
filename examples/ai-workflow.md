# Using jira-cli with AI Assistants

This guide demonstrates how to use jira-cli with AI assistants (Claude, ChatGPT, Gemini, Copilot, etc.) for AI-assisted project management.

## Workflow Overview

1. **Discussion/Meeting** → AI assistant processes transcript
2. **Generate Issues** → AI assistant creates JSON file
3. **Batch Create** → jira-cli creates issues in Jira
4. **Review & Refine** → Team reviews in Jira

## Example: Processing Meeting Notes

### Step 1: Meeting Transcript

```
Meeting Notes - Q1 Platform Planning
=====================================

Attendees: Alice, Bob, Charlie
Date: 2024-01-15

Discussion Points:
- Need to add authentication to our API
  - JWT-based auth with email/password login
  - Story Points: 5
  - Priority: High

- Rate limiting is critical to prevent abuse
  - Implement per-user limits
  - Story Points: 3
  - Should be part of security epic

- We're seeing 500 errors when users send bad JSON
  - Should return 400 instead
  - Priority: High
  - Needs immediate fix

Decisions:
- Create "Q1 Platform Improvements" epic
- Target completion by end of Q1
```

### Step 2: Ask Your AI Assistant to Generate Issues

**User:**
```
Create a jira-cli batch input file from these meeting notes.
Use the epic template for the main initiative and story/bug templates
for individual items.
```

**AI Assistant Response:**

```json
[
  {
    "template": "epic",
    "data": {
      "Project": "PLATFORM",
      "Summary": "Q1 2024 Platform Improvements",
      "Description": "Track all platform improvements planned for Q1 2024.\n\n## Scope\n- API authentication\n- Rate limiting\n- Error handling improvements\n\n## Timeline\nTarget: End of Q1 2024",
      "EpicName": "Q1 Platform",
      "Labels": ["Q1-2024", "platform"]
    }
  },
  {
    "template": "story",
    "data": {
      "Project": "PLATFORM",
      "Summary": "Implement JWT-based authentication",
      "Description": "Add authentication to API endpoints.\n\n## Requirements\n- Email/password login\n- JWT token generation\n- Token validation on protected endpoints\n\n## Notes from Meeting\n- Discussed in Q1 planning meeting\n- Priority: High",
      "Priority": "High",
      "StoryPoints": 5,
      "Labels": ["auth", "security", "Q1-2024"],
      "EpicKey": "PLATFORM-1"
    }
  },
  {
    "template": "story",
    "data": {
      "Project": "PLATFORM",
      "Summary": "Implement rate limiting",
      "Description": "Add rate limiting to prevent API abuse.\n\n## Requirements\n- Per-user limits\n- Configurable thresholds\n- Proper error responses (429)\n\n## Notes\n- Part of security improvements\n- Related to authentication work",
      "StoryPoints": 3,
      "Labels": ["security", "api", "Q1-2024"],
      "EpicKey": "PLATFORM-1"
    }
  },
  {
    "template": "bug",
    "data": {
      "Project": "PLATFORM",
      "Summary": "API returns 500 on malformed JSON",
      "Description": "When clients send malformed JSON, API returns 500 Internal Server Error instead of 400 Bad Request.\n\n## Steps to Reproduce\n1. Send POST request with invalid JSON\n2. Observe 500 response\n\n## Expected Behavior\nShould return 400 Bad Request with helpful error message\n\n## Priority\nHigh - Immediate fix needed",
      "Priority": "High",
      "Labels": ["bug", "api", "error-handling"]
    }
  }
]
```

### Step 3: Save and Create Issues

```bash
# Save the JSON to a file
# (AI assistant might save it directly)

# Validate with dry-run
jira-cli batch create issues.json --dry-run

# Create the issues
jira-cli batch create issues.json

# Output:
# Creating issues... [████████████████████] 100% (4/4)
#
# ✅ Successfully created 4 issues:
# PLATFORM-1: Epic - Q1 2024 Platform Improvements
# PLATFORM-2: Story - Implement JWT-based authentication
# PLATFORM-3: Story - Implement rate limiting
# PLATFORM-4: Bug - API returns 500 on malformed JSON
```

### Step 4: Get JSON Output for Further Processing

```bash
# Get results in JSON format
jira-cli batch create issues.json --json > results.json

# Extract issue keys
cat results.json | jq -r '.created[].key'

# Send summary to team
cat results.json | jq '{
  total: .success,
  epic: .created[0].key,
  stories: [.created[1].key, .created[2].key],
  bugs: [.created[3].key]
}'
```

## Advanced Workflows

### 1. Update Existing Issues

```bash
# Search for issues to update
jira-cli search "project = PLATFORM AND status = 'In Progress'" --json > in-progress.json

# Use AI assistant to analyze and suggest updates
# Then apply updates:
jira-cli update PLATFORM-2 --field story_points=8
jira-cli comment PLATFORM-2 "Updated estimate based on technical investigation"
```

### 2. Generate Reports

```bash
# Get all issues for a sprint
jira-cli search "project = PLATFORM AND sprint = 'Sprint 1'" --json > sprint-1.json

# Ask AI assistant to analyze sprint-1.json and generate report
# Example report: completion rate, blocked items, recommendations
```

### 3. Link Related Issues

```bash
# After creating issues, link them
jira-cli link PLATFORM-2 PLATFORM-3 --type "Relates"
jira-cli link PLATFORM-4 PLATFORM-2 --type "Blocks"
```

## Tips for Using with AI assistant

### 1. Be Specific About Templates

When asking AI assistant to generate issues:

**Good:**
```
Create issues using these templates:
- epic.yaml for the main initiative
- story.yaml for user stories (include StoryPoints)
- bug.yaml for bugs (set Priority to High)
```

**Less Specific:**
```
Create some Jira issues from this
```

### 2. Include Field Mappings

Tell AI assistant about your custom fields:

```
Our Jira instance uses these custom fields:
- story_points = customfield_10016
- epic_link = customfield_10014
- sprint = customfield_10020

Generate a batch file using these aliases.
```

### 3. Request Validation

```
Generate the batch file and validate it against our templates.
Make sure all required fields are present.
```

### 4. Iterative Refinement

```
User: Create issues from meeting notes
Claude: [generates JSON]
User: Add acceptance criteria to stories
Claude: [updates JSON with AC]
User: Looks good, save to issues.json
```

## Example Scripts

### Automated Issue Creation from Text

```bash
#!/bin/bash
# create-issues-from-text.sh

# 1. Ask AI assistant to process text
echo "Processing meeting notes..."

# 2. AI assistant generates issues.json

# 3. Validate
echo "Validating issues..."
jira-cli batch create issues.json --dry-run

# 4. Get user confirmation
read -p "Create these issues? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
  # 5. Create issues
  jira-cli batch create issues.json --json > results.json
  
  # 6. Report results
  echo "Created $(cat results.json | jq -r '.success') issues"
  cat results.json | jq -r '.created[] | "- \(.key): \(.summary)"'
fi
```

### Daily Standup Report

```bash
#!/bin/bash
# standup-report.sh

# Get your recent activity
jira-cli list --assignee currentUser() --limit 10 --json > my-issues.json

# Ask AI assistant to generate standup update
# Input: my-issues.json
# Output: Formatted standup notes with:
#   - Yesterday: Completed tasks
#   - Today: In progress tasks
#   - Blockers: Any blocked issues
```

## Common Patterns

### Pattern 1: Epic with Stories

```json
[
  {"template": "epic", "data": {"Summary": "Epic name", ...}},
  {"template": "story", "data": {"EpicKey": "PROJ-1", ...}},
  {"template": "story", "data": {"EpicKey": "PROJ-1", ...}}
]
```

**Note:** The CLI will create the epic first, then link stories automatically.

### Pattern 2: Bug with Related Stories

```json
[
  {"template": "bug", "data": {"Summary": "Production bug", ...}},
  {"template": "story", "data": {"Summary": "Fix for bug", ...}}
]
```

Then manually link:
```bash
jira-cli link PROJ-2 PROJ-1 --type "Blocks"
```

### Pattern 3: Charter (Planning Doc)

```json
[
  {
    "template": "charter",
    "data": {
      "Project": "PROJ",
      "Summary": "Authentication System Design",
      "Problem": "Need secure API access",
      "Goals": "JWT auth with role-based access",
      "SuccessCriteria": "100% endpoints protected, <50ms overhead"
    }
  }
]
```

## Troubleshooting

### Issue: AI assistant generates invalid JSON

**Solution:** Ask AI assistant to validate JSON:
```
Please validate this JSON and fix any syntax errors
```

### Issue: Custom fields not recognized

**Solution:** Run field discovery first:
```bash
jira-cli fields list --json > fields.json
```

Then share fields.json with AI assistant for reference.

### Issue: Template not found

**Solution:** Initialize templates:
```bash
jira-cli template init
ls ~/.jira-cli/templates/
```

## Best Practices

1. **Always use --dry-run first** to validate before creating
2. **Save generated JSON** for review and version control
3. **Use descriptive Labels** to make searching easier
4. **Link related issues** after creation for better tracking
5. **Include acceptance criteria** in story descriptions
6. **Set realistic story points** based on team velocity
7. **Use consistent templates** across the team

## Resources

- [jira-cli Documentation](../README.md)
- [Template Reference](../templates/)
- [JQL Query Guide](https://support.atlassian.com/jira-software-cloud/docs/use-advanced-search-with-jira-query-language-jql/)
