# Jira CLI Examples

This directory contains example input files for testing the Jira CLI.

## Single Issue Creation

### Create a Story
```bash
jcfa create --template story --data examples/single-story.json
```

### Create an Epic
```bash
jcfa create --template epic --data examples/single-epic.json
```

### Dry-Run Validation
```bash
jcfa create --template story --data examples/single-story.json --dry-run
```

## Batch Creation

### Create Multiple Issues
The `batch-input.json` file demonstrates:
- Creating an epic with ID reference
- Creating multiple stories linked to the epic using `@epic1`
- Creating a bug
- Automatic epic-story linking

```bash
jcfa batch create examples/batch-input.json
```

### Batch with Dry-Run
```bash
jcfa batch create examples/batch-input.json --dry-run
```

### Batch with JSON Output
```bash
jcfa batch create examples/batch-input.json --json
```

## File Descriptions

### single-story.json
Example data for creating a single story issue with:
- Summary and description
- Priority and story points
- Labels

### single-epic.json
Example data for creating an epic with:
- Epic name field
- Comprehensive description
- Labels for tracking

### batch-input.json
Comprehensive example with 6 issues:
1. **Epic**: Q1 2024 Authentication & Security (id: epic1)
2. **Story**: JWT authentication (linked to @epic1)
3. **Story**: Password reset (linked to @epic1)
4. **Story**: Multi-factor authentication (linked to @epic1)
5. **Story**: OAuth social login (linked to @epic1)
6. **Bug**: Session timeout issue

Demonstrates:
- ID referencing system (`@epic1`)
- Epic-story linking
- Different issue types
- Varying story points and priorities
- Rich descriptions with markdown

## Tips

1. **Project Key**: Replace `"PROJ"` with your actual Jira project key
2. **Custom Fields**: Use field mappings for custom fields like Story Points
3. **Templates**: Ensure templates are initialized with `jcfa template init`
4. **Validation**: Always use `--dry-run` first to validate your data
