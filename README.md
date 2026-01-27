# jira-cli-for-agents

A fast, scriptable CLI for Jira Cloud designed for AI-assisted workflows and developer productivity.

## Features

- **Batch Operations**: Create multiple issues from JSON with progress tracking
- **Custom Fields**: Full support for custom fields with user-defined aliases
- **Template System**: Reusable templates for common issue types (Epic, Story, Bug, etc.)
- **Schema Validation**: Validate issue data before creation to catch errors early
- **Auto-Linking**: Automatically link stories to epics during batch creation
- **JQL Search**: Powerful search capabilities using Jira Query Language
- **JSON Output**: Machine-readable output for scripting and automation
- **Issue Management**: Complete CRUD operations for issues
- **Comment Management**: Add, list, update, and delete comments on issues
- **Attachment Support**: Upload, download, list, and delete file attachments
- **Allowlist control**: Specify read only or write access of commands via CLI

## Installation

### Prerequisites

- Go 1.21 or higher
- A Jira Cloud account
- API Token (create at https://id.atlassian.com/manage/api-tokens)

### From Source

```bash
git clone https://github.com/sanisideup/jira-cli-for-agents.git
cd jira-cli-for-agents
go build -o jcfa
```

### Install Globally

```bash
go install github.com/sanisideup/jira-cli-for-agents@latest
```

## Quick Start

### 1. Configure Credentials

```bash
jcfa configure
```

You'll be prompted for:
- Jira domain (e.g., `yourcompany.atlassian.net`)
- Email address
- API token (create at https://id.atlassian.com/manage/api-tokens)
- Default project (optional)

### 2. Discover Custom Fields

```bash
# List all fields
jcfa fields list

# List fields for a specific project
jcfa fields list --project PROJ
```

### 3. Create Field Aliases

```bash
jcfa fields map story_points customfield_10016
jcfa fields map epic_link customfield_10014
```

### 4. Initialize Templates

```bash
jcfa template init
```

Templates are saved to `~/.jcfa/templates/` for customization.

## Commands

### Issue Operations

#### Get Issue

```bash
# Default output (description + attachments)
jcfa get PROJ-123

# Show linked issues
jcfa get PROJ-123 --links
jcfa get PROJ-123 -l

# Show subtasks
jcfa get PROJ-123 --subtasks
jcfa get PROJ-123 -s

# Show comments
jcfa get PROJ-123 --comments
jcfa get PROJ-123 -c

# Combine multiple options
jcfa get PROJ-123 --links --comments
jcfa get PROJ-123 -lc

# Show everything (links + subtasks + comments)
jcfa get PROJ-123 --full
jcfa get PROJ-123 -f

# JSON output
jcfa get PROJ-123 --json
```

Output:
```
PROJ-123: User authentication story
================================================================================
Type: Story                              Status: In Progress
Priority: Medium                         Assignee: John Doe
Created: 2024-01-15                      Updated: 2024-01-20
Epic Link: PROJ-100
Labels: auth, security

Description:
--------------------------------------------------------------------------------
Implement JWT-based authentication for the platform.

Requirements:
• Token expiry handling
• Refresh token support
• Secure storage

Attachments (2):
--------------------------------------------------------------------------------
  design-spec.pdf                        2.3 MB    John Doe        2024-01-15
  auth-flow.png                        156.0 KB    Jane Smith      2024-01-16
```

With `--links` flag:
```
Linked Issues (2):
--------------------------------------------------------------------------------
  → blocks       PROJ-124     [To Do       ] Update dashboard component
  ← blocked by   PROJ-122     [Done        ] Set up database schema
```

With `--subtasks` flag:
```
Subtasks (3):
--------------------------------------------------------------------------------
  PROJ-123-1      [Done        ] Research JWT libraries
  PROJ-123-2      [In Progress ] Implement token service
  PROJ-123-3      [To Do       ] Add unit tests
```

With `--comments` flag:
```
Comments (2):
--------------------------------------------------------------------------------
[2024-01-16 14:30] John Doe:
Started implementation. Using go-jwt library.

[2024-01-17 09:15] Jane Smith:
Looks good! Make sure to add refresh token support.
```

#### Search Issues (JQL)

```bash
# Search with JQL
jcfa search "project = PROJ AND status = Open"

# Limit results
jcfa search "assignee = currentUser() ORDER BY updated DESC" --limit 20

# JSON output for scripting
jcfa search "type = Bug" --json
```

#### List Issues

```bash
# List your recent issues
jcfa list

# Filter by project
jcfa list --project PROJ

# Filter by assignee and status
jcfa list --assignee john@example.com --status "In Progress"

# Limit results
jcfa list --limit 10
```

#### Create Issue

```bash
# Create from template
jcfa create --template story --data story.json

# Dry-run mode (validation only)
jcfa create --template epic --data epic.json --dry-run
```

Example `story.json`:
```json
{
  "Project": "PROJ",
  "Summary": "User authentication",
  "Description": "Implement JWT-based authentication",
  "StoryPoints": 5,
  "Labels": ["auth", "security"]
}
```

#### Update Issue

```bash
# Update single field
jcfa update PROJ-123 --field summary="New title"

# Update multiple fields
jcfa update PROJ-123 \
  --field summary="Updated title" \
  --field story_points=8

# Update with field aliases
jcfa update PROJ-123 --field status="In Progress"
```

#### Add Comment

```bash
jcfa comment PROJ-123 "This is a comment"
jcfa comment PROJ-123 "Updated the implementation" --json
```

#### Transition Issue

```bash
# Move to different status
jcfa transition PROJ-123 "In Progress"
jcfa transition PROJ-123 "Done"
```

The CLI will automatically find the correct transition based on the status name (case-insensitive).

#### Link Issues

```bash
# Create a link (new syntax)
jcfa link create PROJ-123 PROJ-124 --type Blocks

# Create a link (legacy syntax - backward compatible)
jcfa link PROJ-123 PROJ-124 --type Blocks

# List available link types
jcfa link types
jcfa link types --json

# List all links on an issue
jcfa link list PROJ-123
jcfa link list PROJ-123 --json

# Delete a link by ID (requires --confirm for safety)
jcfa link delete 10234 --confirm
```

Output of `link types`:
```
Available Link Types:
────────────────────────────────────────────────────────────────────────────────
  Name              Inward                  Outward
  ────────────────  ──────────────────────  ──────────────────────
  Blocks            is blocked by           blocks
  Cloners           is cloned by            clones
  Duplicate         is duplicated by        duplicates
  Relates           relates to              relates to
```

Output of `link list PROJ-123`:
```
Links for PROJ-123 (2 total):
───────────────────────────────────────────────────────────────────────────────
  ID       Direction  Type        Issue       Status       Summary
  ──────   ────────   ──────────  ──────────  ───────────  ─────────────────
  10234    →          blocks      PROJ-124    [To Do]      Update dashboard...
  10235    ←          blocked by  PROJ-122    [Done]       Setup database...
```

Common link types:
- **Blocks**: The first issue blocks the second
- **Relates**: The issues are related
- **Duplicate**: The first issue duplicates the second
- **Epic**: Link a story to an epic

#### Create Subtasks

```bash
# Create a subtask under a parent issue
jcfa create --template task --data task.json --parent PROJ-123
jcfa create -t task -d task.json -p PROJ-123

# Dry-run to validate subtask creation
jcfa create --template task --data task.json --parent PROJ-123 --dry-run

# Interactive subtask creation
jcfa create --template task --interactive --parent PROJ-123
```

Notes:
- The parent issue must exist and cannot itself be a subtask
- Works with any template (task, story, bug, etc.)
- The issue type should typically be "Sub-task" or similar for proper Jira hierarchy

### Comment Operations

#### Add Comment

```bash
# Add a comment (legacy syntax - backward compatible)
jcfa comment PROJ-123 "This is a comment"

# Add a comment (new syntax with subcommands)
jcfa comments add PROJ-123 "This is a comment"

# Add comment with JSON output
jcfa comments add PROJ-123 "Implementation complete" --json
```

#### List Comments

```bash
# List all comments on an issue
jcfa comments list PROJ-123

# List with limit
jcfa comments list PROJ-123 --limit 10

# List in reverse order (newest first)
jcfa comments list PROJ-123 --order -created

# JSON output
jcfa comments list PROJ-123 --json
```

#### Get Comment

```bash
# Get a specific comment by ID
jcfa comments get PROJ-123 10001

# JSON output
jcfa comments get PROJ-123 10001 --json
```

#### Update Comment

```bash
# Update an existing comment
jcfa comments update PROJ-123 10001 "Updated comment text"

# Note: You can only update comments you created or if you have admin permissions
```

#### Delete Comment

```bash
# Delete a comment (requires confirmation for safety)
jcfa comments delete PROJ-123 10001 --confirm

# Note: You can only delete comments you created or if you have admin permissions
```

### Attachment Operations

#### List Attachments

```bash
# List all attachments on an issue
jcfa attachment list PROJ-123

# JSON output
jcfa attachment list PROJ-123 --json
```

Output:
```
Attachments for PROJ-123 (3 total):

ID       Filename           Size    Author      Date
----------------------------------------------------------------------------------
10001    design.pdf         2.3 MB  John Doe    2024-01-15 10:30
10002    screenshot.png     156 KB  Jane Smith  2024-01-16 14:20
10003    requirements.docx  45 KB   John Doe    2024-01-15 09:15
```

#### Upload Attachment

```bash
# Upload a single file
jcfa attachment upload PROJ-123 design.pdf

# Upload multiple files
jcfa attachment upload PROJ-123 file1.pdf file2.png file3.docx

# Upload without progress bar (for automation)
jcfa attachment upload PROJ-123 large-file.zip --no-progress

# Files larger than 1MB automatically show progress bar
```

#### Download Attachment

```bash
# Download by filename
jcfa attachment download PROJ-123 design.pdf

# Download by ID
jcfa attachment download PROJ-123 10001

# Download to specific directory
jcfa attachment download PROJ-123 design.pdf --output ./downloads/

# Download with custom filename
jcfa attachment download PROJ-123 design.pdf --output custom-name.pdf

# Download without progress bar
jcfa attachment download PROJ-123 large-file.zip --no-progress
```

#### Delete Attachment

```bash
# Delete an attachment by ID (requires confirmation)
jcfa attachment delete 10001 --confirm

# Note: You need appropriate permissions to delete attachments
```

### Batch Operations

#### Batch Create

```bash
# Create multiple issues
jcfa batch create issues.json

# Dry-run mode (validation only)
jcfa batch create issues.json --dry-run

# Disable progress bar
jcfa batch create issues.json --no-progress
```

Example `issues.json`:
```json
[
  {
    "template": "epic",
    "data": {
      "Project": "PROJ",
      "Summary": "Q1 Platform Epic",
      "Description": "Epic description",
      "EpicName": "Q1 Platform"
    }
  },
  {
    "template": "story",
    "data": {
      "Project": "PROJ",
      "Summary": "User authentication",
      "Description": "Story description",
      "EpicKey": "PROJ-1",
      "StoryPoints": 5
    }
  }
]
```

### Field Management

#### List Fields

```bash
# List all fields
jcfa fields list

# List fields for specific project
jcfa fields list --project PROJ

# JSON output
jcfa fields list --json
```

#### Map Field Alias

```bash
jcfa fields map story_points customfield_10016
jcfa fields map epic_name customfield_10011
jcfa fields map epic_link customfield_10014
```

### Configuration

#### Configure

```bash
# Interactive configuration
jcfa configure
```

#### Version

```bash
jcfa version
jcfa version --json
```

### Allowlist Management

Manage command restrictions for sandboxed or AI-assisted execution.

#### View Status

```bash
# Show current allowlist status
jcfa allowlist status
jcfa allowlist status --json
```

Output:
```
Command Allowlist Status
========================

Status: ENABLED (read-only mode)
Mode:   JIRA_READONLY=1

Only read operations are allowed. Write commands are blocked.

Allowed commands:
  ✓ attachment list
  ✓ comments get
  ✓ comments list
  ✓ fields
  ✓ get
  ✓ help
  ...

Note: 'help', 'version', '--help', '-h' are always allowed.
```

#### List Commands by Category

```bash
# List all commands categorized as read/write
jcfa allowlist commands
jcfa allowlist commands --json
```

Output:
```
Available Commands by Category
==============================

READ COMMANDS (11) - Safe for read-only mode:
  ✓ attachment list
  ✓ comments get
  ✓ comments list
  ✓ fields
  ✓ get
  ✓ help
  ✓ link list
  ✓ link types
  ✓ list
  ✓ search
  ✓ version

WRITE COMMANDS (16) - Blocked in read-only mode:
  ✗ attachment delete
  ✗ attachment upload
  ✗ batch
  ✗ batch create
  ...

Total: 27 commands
```

#### Check Specific Command

```bash
# Check if a command is allowed (exit code 0=allowed, 1=blocked)
jcfa allowlist check get
jcfa allowlist check create

# Useful in scripts
if jcfa allowlist check create; then
  jcfa create --template story --data story.json
else
  echo "Create command is blocked"
fi
```

#### Enable Instructions

```bash
# Show how to enable allowlist restrictions
jcfa allowlist enable
```

Displays platform-specific instructions for enabling read-only mode or custom allowlists.

## Global Flags

All commands support these global flags:

- `--config <path>`: Override config file location (default: `~/.jcfa/config.yaml`)
- `--json`: Output in JSON format for scripting
- `--verbose` or `-v`: Enable verbose logging
- `--no-color`: Disable colored output

## Configuration File

Location: `~/.jcfa/config.yaml`

```yaml
domain: yourcompany.atlassian.net
email: you@example.com
api_token: your-api-token
default_project: PROJ
field_mappings:
  story_points: customfield_10016
  epic_link: customfield_10014
  epic_name: customfield_10011
max_attachment_size: 10  # Maximum attachment size in MB (default: 10)
download_path: ./downloads  # Default download directory (default: current directory)
```

**Security**: Config file is automatically set to `0600` permissions (read/write for owner only).

## Templates

Templates use Go's `text/template` syntax with field placeholders.

### Epic Template

File: `~/.jcfa/templates/epic.yaml`

```yaml
type: Epic
fields:
  project: "{{ .Project }}"
  summary: "{{ .Summary }}"
  description: "{{ .Description }}"
  labels: {{ .Labels | toJson }}
  customfield_10011: "{{ .EpicName }}"
```

### Story Template

File: `~/.jcfa/templates/story.yaml`

```yaml
type: Story
fields:
  project: "{{ .Project }}"
  summary: "{{ .Summary }}"
  description: "{{ .Description }}"
  priority: { "name": "{{ .Priority | default \"Medium\" }}" }
  labels: {{ .Labels | toJson }}
  customfield_10016: {{ .StoryPoints | default nil }}
  customfield_10014: "{{ .EpicKey }}"
```

### Bug Template

File: `~/.jcfa/templates/bug.yaml`

```yaml
type: Bug
fields:
  project: "{{ .Project }}"
  summary: "{{ .Summary }}"
  description: "{{ .Description }}"
  priority: { "name": "{{ .Priority | default \"High\" }}" }
  labels: {{ .Labels | toJson }}
```

## Usage with AI Assistants

This CLI is designed to work seamlessly with AI assistants (Claude, ChatGPT, Gemini, Copilot, etc.) for AI-assisted project management.

### Example Workflow

1. **Parse meeting transcript** with your AI assistant
2. **Generate issues.json** from discussion points
3. **Batch create** issues: `jcfa batch create issues.json`
4. **Review** created issues in Jira

See [examples/ai-workflow.md](examples/ai-workflow.md) for detailed examples.

## Security

### Secure Credential Storage

By default, API tokens are stored in `~/.jcfa/config.yaml` with restricted permissions (0600). For enhanced security, the `configure` command offers to store your API token in the OS keyring:

```bash
jcfa configure
# ...
# Store API token securely in system keyring? [Y/n]: y
# ✓ API token stored securely in keychain
```

You can also configure the backend via environment variables:

```bash
# Use OS keyring (macOS Keychain, Windows Credential Manager, Linux Secret Service)
export JIRA_KEYRING_BACKEND=keychain

# Or use encrypted file (for CI/headless environments)
export JIRA_KEYRING_BACKEND=file
export JIRA_KEYRING_PASSWORD='your-secure-password'
```

**Supported backends:**
- `auto` (default): Automatically selects the best backend for your platform
  - macOS/Windows: Uses OS keyring
  - Linux with GUI: Uses Secret Service
  - CI/headless: Uses encrypted file
- `keychain`: OS keyring (recommended for interactive use)
- `file`: Encrypted file (for CI/SSH environments)

### Command Allowlist (Agent Safety)

When running in sandboxed or AI-assisted environments, you can restrict which commands are allowed:

```bash
# Enable read-only mode (only allows get, search, list, fields, etc.)
export JIRA_READONLY=1

# Or specify exact commands allowed
export JIRA_COMMAND_ALLOWLIST="get,search,list,fields"
```

**Read-only commands** (safe for AI agents):
- `get`, `search`, `list`, `fields`, `version`, `help`
- `attachment list`, `comments list`, `comments get`, `link list`, `link types`

**Write commands** (blocked in read-only mode):
- `create`, `update`, `transition`, `batch`, `batch create`
- `comment`, `comments add/update/delete`
- `link`, `link create/delete`
- `attachment upload/delete`
- `configure`, `template`

Use `jcfa allowlist status` to view current restrictions or `jcfa allowlist commands` to see all commands by category.

## Exit Codes

The CLI uses specific exit codes for different error types:

- `0`: Success
- `1`: Authentication failure
- `2`: Validation error
- `3`: API error
- `4`: Configuration error

This allows for proper error handling in scripts:

```bash
if jcfa create --template story --data story.json; then
  echo "Issue created successfully"
else
  exit_code=$?
  case $exit_code in
    1) echo "Authentication failed - check your credentials" ;;
    2) echo "Validation error - check your data" ;;
    3) echo "API error - Jira may be unavailable" ;;
    4) echo "Configuration error - run 'jcfa configure'" ;;
  esac
fi
```

## Examples

### Create Epic with Stories

```bash
# Create epic
jcfa create --template epic --data epic.json

# Create stories linked to epic
jcfa batch create stories.json
```

### Search and Update

```bash
# Find all open bugs
jcfa search "project = PROJ AND type = Bug AND status = Open" --json > bugs.json

# Update a bug
jcfa update PROJ-456 --field priority="High"
jcfa comment PROJ-456 "Investigating the root cause"
jcfa transition PROJ-456 "In Progress"
```

### Automated Workflow

```bash
#!/bin/bash

# Create issues from JSON
jcfa batch create issues.json --json > results.json

# Check exit code
if [ $? -eq 0 ]; then
  echo "All issues created successfully"
  cat results.json | jq '.created[].key'
else
  echo "Some issues failed to create"
  cat results.json | jq '.errors'
fi
```

## Development

### Project Structure

```
jira-cli/
├── cmd/                    # CLI commands (Cobra)
│   ├── root.go            # Root command + global flags
│   ├── configure.go       # Interactive setup
│   ├── version.go         # Version command
│   ├── fields.go          # Field discovery
│   ├── template.go        # Template management
│   ├── create.go          # Single issue creation
│   ├── batch.go           # Batch creation
│   ├── get.go             # Get issue
│   ├── search.go          # JQL search
│   ├── list.go            # List issues
│   ├── update.go          # Update issue
│   ├── comment.go         # Add comments
│   ├── transition.go      # Status transitions
│   ├── link.go            # Issue linking (create, types, list, delete)
│   └── allowlist.go       # Allowlist management commands
├── pkg/
│   ├── allowlist/         # Command restriction for agent safety
│   ├── client/            # HTTP client with retry
│   ├── config/            # Config management
│   ├── jira/              # Jira services
│   │   ├── fields.go      # Field discovery
│   │   ├── metadata.go    # Schema validation
│   │   ├── issue.go       # Issue operations
│   │   ├── link.go        # Issue linking
│   │   └── search.go      # Search operations
│   ├── models/            # Data models
│   └── secrets/           # Secure credential storage
├── templates/             # Default templates
└── main.go                # Entry point
```

### Testing

```bash
# Run all tests
go test ./...

# With coverage
go test -cover ./...

# Specific package
go test ./pkg/jira -v
```

### Building

```bash
# Standard build
go build -o jcfa

# With version info
go build -ldflags="-X 'github.com/sanisideup/jira-cli-for-agents/cmd.Version=1.0.0'" -o jira-cli
```

## Architecture

For detailed information about the design decisions, API integration patterns, and technical implementation:

📖 **[Read the Architecture Documentation](ARCHITECTURE.md)**

The architecture document covers:
- System architecture and layer responsibilities
- API integration patterns (retry, rate limiting, pagination)
- Epic-story hierarchy handling (modern vs legacy)
- Field type system and handlers
- Template rendering engine
- Error handling strategy
- Performance optimizations
- Security considerations
- Testing strategy

## API Reference

Built on [Jira Cloud REST API v3](https://developer.atlassian.com/cloud/jira/platform/rest/v3/)

### Endpoints Used

- `GET /rest/api/3/myself` - Authentication validation
- `GET /rest/api/3/field` - Field discovery
- `GET /rest/api/3/issue/createmeta` - Schema validation
- `POST /rest/api/3/issue` - Create issue
- `POST /rest/api/3/issue/bulk` - Bulk create
- `GET /rest/api/3/issue/{key}` - Get issue
- `PUT /rest/api/3/issue/{key}` - Update issue
- `POST /rest/api/3/search` - JQL search
- `POST /rest/api/3/issue/{key}/comment` - Add comment
- `GET /rest/api/3/issue/{key}/transitions` - Get transitions
- `POST /rest/api/3/issue/{key}/transitions` - Transition issue
- `POST /rest/api/3/issueLink` - Link issues
- `DELETE /rest/api/3/issueLink/{linkId}` - Delete link
- `GET /rest/api/3/issueLinkType` - Get link types

## Troubleshooting

### Authentication Issues

- Verify API token at https://id.atlassian.com/manage/api-tokens
- Ensure email matches your Atlassian account
- Check domain format (should be `yourcompany.atlassian.net` without `https://`)

### Custom Field Not Found

- Run `jcfa fields list` to discover field IDs
- Field IDs vary by Jira instance
- Create field mapping: `jcfa fields map <alias> <field-id>`

### Rate Limiting

The CLI automatically handles rate limits (HTTP 429) with exponential backoff (3 retries: 1s, 2s, 4s).

### Template Errors

- Ensure template fields match your Jira instance
- Use `--dry-run` to validate before creating
- Check field mappings in config

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

MIT License - see LICENSE file for details

## Support

- [Jira Cloud REST API Documentation](https://developer.atlassian.com/cloud/jira/platform/rest/v3/)
- [Create API Token](https://id.atlassian.com/manage/api-tokens)
- [GitHub Issues](https://github.com/sanisideup/jira-cli-for-agents/issues)

## Acknowledgments

Built with:
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Resty](https://github.com/go-resty/resty) - HTTP client
- [YAML v3](https://gopkg.in/yaml.v3) - YAML parsing
