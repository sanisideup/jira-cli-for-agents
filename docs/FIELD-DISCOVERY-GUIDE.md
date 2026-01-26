# Field Discovery Guide

Quick reference for using the field discovery and mapping features.

## Commands

### List All Fields

```bash
# Human-readable table output
jira-cli fields list

# JSON output for scripting
jira-cli fields list --json

# Filter by project (returns all fields for now)
jira-cli fields list --project MYPROJ
```

### Create Field Alias

```bash
# Map a custom field to a friendly alias
jira-cli fields map story_points customfield_10016
jira-cli fields map epic_link customfield_10014
jira-cli fields map sprint customfield_10020
```

## Understanding the Output

### Human-Readable Format

```
Standard Fields:
ID              NAME                TYPE
--------------------------------------------------------------------------------
summary         Summary             string
description     Description         string
issuetype       Issue Type          issuetype
...

Custom Fields:
ID                    NAME              TYPE
--------------------------------------------------------------------------------
customfield_10014     Epic Link         epic
customfield_10016     Story Points      float
customfield_10020     Sprint            sprint
...

Total: 150 fields (45 standard, 105 custom)

Current Field Mappings:
ALIAS           FIELD ID
----------------------------------------
story_points    customfield_10016
epic_link       customfield_10014
```

**Standard Fields:** Built-in Jira fields that exist in all instances (summary, description, assignee, etc.)

**Custom Fields:** Organization-specific fields created by admins (identified by `customfield_*` prefix)

**Field Mappings:** Your configured aliases for easier reference

### JSON Format

```json
[
  {
    "id": "summary",
    "name": "Summary",
    "custom": false,
    "orderable": true,
    "navigable": true,
    "searchable": true,
    "schema": {
      "type": "string",
      "system": "summary"
    }
  },
  {
    "id": "customfield_10016",
    "name": "Story Points",
    "custom": true,
    "orderable": true,
    "navigable": true,
    "searchable": true,
    "schema": {
      "type": "number",
      "custom": "com.atlassian.jira.plugin.system.customfieldtypes:float"
    }
  }
]
```

## Finding Custom Field IDs

### Method 1: Use the CLI (Recommended)

```bash
# List all fields and search for the one you need
jira-cli fields list | grep -i "story points"
# Output: customfield_10016     Story Points      float

# Or use JSON output and jq
jira-cli fields list --json | jq '.[] | select(.name | contains("Story Points"))'
```

### Method 2: Jira UI

1. Go to Jira Settings → Issues → Custom Fields
2. Click on the field you want
3. Look at the URL: `...customfield_10016`
4. That's your field ID!

### Method 3: Browser Developer Tools

1. Open a Jira issue
2. Right-click on the custom field → Inspect
3. Look for the `data-fieldkey` or similar attribute
4. Copy the `customfield_*` value

## Common Field Aliases

Here are common custom fields you might want to map:

```bash
# Agile/Scrum fields
jira-cli fields map story_points customfield_10016
jira-cli fields map epic_link customfield_10014
jira-cli fields map epic_name customfield_10011
jira-cli fields map sprint customfield_10020

# Time tracking (if custom)
jira-cli fields map original_estimate customfield_10001
jira-cli fields map remaining_estimate customfield_10002

# Development (if you use these)
jira-cli fields map development customfield_10100
jira-cli fields map pull_request customfield_10101
```

## Field Mapping Best Practices

### ✅ Good Alias Names

- `story_points` - lowercase, underscore separated
- `epic_link` - clear, descriptive
- `original_estimate` - follows Jira naming
- `team` - short but clear

### ❌ Bad Alias Names

- `story-points` - hyphens not allowed
- `Story Points` - spaces not allowed
- `sp` - too abbreviated, unclear
- `customfield_10016` - defeats the purpose

### Alias Naming Guidelines

1. **Use lowercase** - easier to type
2. **Use underscores** - not hyphens or spaces
3. **Be descriptive** - `epic_link` not `el`
4. **Match Jira names** - helps with recognition
5. **Stay consistent** - if you use `story_points`, use `epic_link` (not `epicLink`)

## Configuration File

Mappings are stored in `~/.jira-cli/config.yaml`:

```yaml
domain: yourcompany.atlassian.net
email: user@example.com
api_token: your_token_here
default_project: PROJ
field_mappings:
  story_points: customfield_10016
  epic_link: customfield_10014
  sprint: customfield_10020
  epic_name: customfield_10011
```

### Manual Editing

You can manually edit this file to:
- **Remove mappings:** Delete the line from `field_mappings`
- **Rename aliases:** Change the key (left side)
- **Update field IDs:** Change the value (right side) if IDs change

**Note:** Always validate with `jira-cli fields list` after manual edits.

## Error Messages

### "Field not found"

```
Error: cannot map alias 'my_field': field with ID 'customfield_99999' not found
```

**Solution:** Double-check the field ID with `jira-cli fields list`

### "Alias already mapped"

```
Error: alias 'story_points' already mapped to 'customfield_10016'. Remove the existing mapping first
```

**Solution:** Edit `~/.jira-cli/config.yaml` to remove or rename the existing mapping

### "Invalid alias"

```
Error: invalid alias 'story-points': alias must contain only letters, numbers, and underscores
```

**Solution:** Use only alphanumeric characters and underscores (e.g., `story_points`)

## Using Field Mappings (Future Phases)

Once you've created field mappings, you'll be able to use them in:

### Issue Creation (Phase 4)
```bash
# Instead of this:
jira-cli create --field customfield_10016=5

# You can do this:
jira-cli create --field story_points=5
```

### Issue Updates (Phase 5)
```bash
# Instead of this:
jira-cli update PROJ-123 --field customfield_10016=8

# You can do this:
jira-cli update PROJ-123 --field story_points=8
```

### Templates (Phase 3)
```yaml
# In your template files
fields:
  summary: "{{.Summary}}"
  story_points: {{.StoryPoints}}  # Uses your alias!
  epic_link: "{{.EpicKey}}"       # Uses your alias!
```

## Scripting Examples

### Export all custom fields to CSV

```bash
jira-cli fields list --json | \
  jq -r '.[] | select(.custom == true) | [.id, .name, .schema.type] | @csv' > custom_fields.csv
```

### Find all custom fields of a specific type

```bash
# Find all custom fields that are numbers
jira-cli fields list --json | \
  jq '.[] | select(.custom == true and .schema.type == "number")'
```

### Auto-generate common mappings

```bash
# Create mappings for common Scrum fields
for field in "Story Points:customfield_10016" "Epic Link:customfield_10014" "Sprint:customfield_10020"; do
  IFS=':' read -r name id <<< "$field"
  alias=$(echo "$name" | tr '[:upper:]' '[:lower:]' | tr ' ' '_')
  jira-cli fields map "$alias" "$id"
done
```

## Troubleshooting

### "Config file not found"

Run `jira-cli configure` first to set up your credentials.

### Field IDs changed after Jira migration

After migrating Jira instances, custom field IDs may change. You'll need to:

1. List fields in the new instance: `jira-cli fields list`
2. Find the new IDs for your custom fields
3. Update your config manually or re-run the map commands

### Can't find a field I know exists

Some fields may not appear if:
- You don't have permission to view them
- They're project-specific and you're looking at the wrong project
- They were deleted by an admin

Try:
```bash
# Search by name (case-insensitive)
jira-cli fields list --json | jq '.[] | select(.name | test("search term"; "i"))'
```

## Tips & Tricks

### Quick field ID lookup

```bash
# Add to your .bashrc / .zshrc
alias jira-field='jira-cli fields list --json | jq -r ".[] | select(.name | contains(\"$1\")) | .id"'

# Usage:
jira-field "Story Points"
# Output: customfield_10016
```

### Backup your mappings

```bash
# Backup config before changes
cp ~/.jira-cli/config.yaml ~/.jira-cli/config.yaml.backup

# Restore if needed
cp ~/.jira-cli/config.yaml.backup ~/.jira-cli/config.yaml
```

### Share mappings with team

```bash
# Export just the field mappings
jq '.field_mappings' ~/.jira-cli/config.yaml > team-field-mappings.json

# Team members can copy these to their config
```

---

**Next:** [Schema Validation Guide](./SCHEMA-VALIDATION-GUIDE.md) (Phase 3)
