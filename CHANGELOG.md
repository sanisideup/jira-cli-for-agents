# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.3.0] - 2026-01-27

### Added

#### Secure Credential Storage
- **Keyring integration**: Store API tokens securely in OS keyring
  - macOS Keychain, Windows Credential Manager, Linux Secret Service
  - Automatic backend selection based on platform and environment
- **Encrypted file backend**: For CI/headless environments
  - Password-protected via `JIRA_KEYRING_PASSWORD` environment variable
- **`configure` command enhancement**: Now prompts for keyring storage during setup
  - "Store API token securely in system keyring? [Y/n]"

#### Command Allowlist System
- **Read-only mode**: `JIRA_READONLY=1` restricts to non-destructive commands
- **Custom allowlist**: `JIRA_COMMAND_ALLOWLIST=cmd1,cmd2` for fine-grained control
- **`allowlist` command group** with subcommands:
  - `status`: Show current allowlist configuration and allowed commands
  - `commands`: List all commands by category (read/write)
  - `check <cmd>`: Check if specific command is allowed (exit code 0/1)
  - `enable`: Show platform-specific instructions for enabling restrictions

#### New Files & Packages
- `pkg/secrets/` - Secure credential storage system with keychain and file backends
- `pkg/allowlist/` - Command restriction logic with read-only and explicit modes
- `cmd/allowlist.go` - Allowlist CLI management commands
- `AGENTS.md` - AI agent and contributor guidelines

### Changed
- `configure` command now offers keyring storage option with automatic backend selection
- Root command validates commands against allowlist before execution
- Config file supports new fields: `use_keyring`, `keyring_backend`

### Technical Details
- Comprehensive test suites for `pkg/secrets/` and `pkg/allowlist/`
- 844 lines of allowlist tests covering edge cases (special chars, whitespace, partial matches)
- 261 lines of secrets tests covering keychain and file backends
- Platform-specific backend selection (darwin, windows, linux)
- CI environment detection for automatic file backend selection

## [1.2.0] - 2026-01-26

### Added

#### Enhanced Link Command Group
- **`link create`**: New explicit subcommand for creating links between issues
  - Usage: `jira-cli link create PROJ-123 PROJ-456 --type Blocks`
- **`link types`**: List all available link types in your Jira instance
  - Shows name, inward description, and outward description
  - Supports `--json` output
- **`link list`**: List all links on a specific issue
  - Shows link ID, direction (→/←), type, linked issue, status, and summary
  - Supports `--json` output
- **`link delete`**: Delete a link by its ID
  - Requires `--confirm` flag for safety
  - Supports `--json` output

#### Subtask Creation
- **`--parent` flag** on `create` command for creating subtasks
  - Usage: `jira-cli create --template task --data task.json --parent PROJ-123`
  - Validates parent issue exists
  - Prevents creating sub-subtasks (parent cannot be a subtask)
  - Works with `--dry-run` for validation
  - Short flag: `-p`

### Changed

- **Link command restructured** as a command group with subcommands
- **Backward compatibility maintained**: Legacy `jira-cli link PROJ-123 PROJ-456 --type Blocks` syntax continues to work

### Technical Details

- New `GetIssueLinks()` method in `pkg/jira/link.go` for retrieving issue links
- New `DeleteIssueLink()` method in `pkg/jira/link.go` for deleting links
- Updated `IssueLink` model with `ID` and `Self` fields
- New `IssueParent` model for subtask parent references
- Added `setupParentIssue()` validation in `cmd/create.go`

## [1.1.0] - 2026-01-26

### Added

#### Enhanced `get` Command
- **Description Display**: Descriptions are now properly parsed from Atlassian Document Format (ADF) to plain text
  - Supports paragraphs, headings, code blocks, lists, blockquotes, tables, and more
  - Code blocks show language indicators (e.g., `[json]`)
  - Bullet and numbered lists render with proper formatting
  - @mentions, emojis, and smart links are preserved
- **Attachments Display**: Shows attachment list with filename, size, author, and date
- **Linked Issues** (`--links`, `-l`): Display linked issues with direction arrows and status
- **Subtasks** (`--subtasks`, `-s`): Display subtasks with status and summary
- **Comments** (`--comments`, `-c`): Fetch and display issue comments with ADF parsing
- **Full View** (`--full`, `-f`): Show all optional sections at once

#### New Files
- `pkg/jira/adf.go` - Comprehensive Atlassian Document Format parser
- `pkg/jira/adf_test.go` - 19 unit tests for ADF parser

### Changed
- `get` command output format improved with two-column layout and 80-char separators
- Epic link detection now checks multiple common field IDs
- Labels displayed comma-separated on a single line

### Fixed
- Description field now properly renders rich text content instead of showing raw ADF JSON
- Proper handling of nil/empty descriptions and attachments

### Technical Details
- ADF parser supports: doc, paragraph, heading, codeBlock, blockquote, bulletList, orderedList, listItem, text, hardBreak, mediaSingle, mediaGroup, rule, table, tableRow, tableHeader, tableCell, inlineCard, mention, emoji
- Comments are fetched via separate API call only when `--comments` or `--full` flag is used (lazy loading)
- Short flags can be combined (e.g., `-lsc` for links + subtasks + comments)

## [1.0.0] - 2026-01-23

### Added

#### Core Features
- **Issue Operations**
  - Single issue creation with templates
  - Batch issue creation (supports up to 50 issues per API request, auto-chunks larger batches)
  - Get issue details by key
  - Update issue fields
  - Search issues using JQL
  - List recent issues with filters
  - Add comments to issues
  - Transition issues to new status
  - Link issues (Blocks, Relates, Duplicate, Epic)

#### Field Management
- Custom field discovery (`fields list`)
- Field alias mapping (`fields map`)
- Automatic field resolution in update commands
- Support for all standard Jira field types

#### Template System
- Built-in templates: Epic, Story, Bug, Charter
- Template initialization (`template init`)
- Go text/template syntax support
- Default value handling
- JSON rendering helpers

#### Schema Validation
- Pre-creation validation using Jira metadata API
- Required field checking
- Field type validation
- Allowed value validation
- Detailed error messages

#### Batch Operations
- Epic creation with automatic story linking
- Progress tracking with visual progress bar
- Reference resolution system (@epic1 → actual key)
- Dry-run mode for validation
- Aggregated results reporting
- Error handling per item

#### Search & Discovery
- JQL query support
- Configurable result limits
- Project-based filtering
- Assignee filtering
- Status filtering
- Tabular output format

#### Authentication & Configuration
- API token authentication
- Interactive configuration wizard
- Credential validation
- Secure config storage (0600 permissions)
- Custom config file support

### Commands

#### Configuration
- `configure` - Interactive setup wizard
- `version` - Show version information

#### Field Management
- `fields list` - Discover available fields
- `fields map` - Create field aliases
- `fields get` - Get field details

#### Template Management
- `template init` - Initialize default templates
- `template list` - List available templates
- `template show` - Show template content

#### Issue Operations
- `create` - Create single issue from template
- `batch create` - Create multiple issues from JSON
- `get` - Get issue details by key
- `search` - Search issues using JQL
- `list` - List recent issues with filters
- `update` - Update issue fields
- `comment` - Add comment to issue
- `transition` - Transition issue to new status
- `link` - Create link between issues

### Features

#### User Experience
- Human-readable output by default
- JSON output mode (`--json`) for all commands
- Verbose logging (`--verbose`)
- Colored output (disable with `--no-color`)
- Progress bars for batch operations
- Helpful error messages
- Field alias support

#### Automation & Scripting
- Exit codes: 0 (success), 1 (auth), 2 (validation), 3 (API), 4 (config)
- JSON output for all commands
- Dry-run mode for validation
- Batch processing support
- Reference resolution for epic linking

#### Developer Experience
- Comprehensive documentation
- Example JSON files
- AI assistant workflow guide
- Troubleshooting section
- API endpoint reference
- Template examples

#### Technical Features
- Exponential backoff retry (3 attempts)
- Rate limit handling (HTTP 429)
- Automatic request batching for >50 issues
- Metadata caching (5-minute TTL)
- HTTP timeout (30s)
- Connection reuse

### Documentation

- **README.md** - Complete user guide with examples
- **examples/story.json** - Sample story data
- **examples/epic.json** - Sample epic data
- **examples/batch-input.json** - Sample batch file
- **examples/ai-workflow.md** - AI assistant workflow guide

### Performance

- Single issue creation: 1 API call
- Batch creation (≤50): 1 API call
- Batch creation (>50): Auto-chunked into multiple requests
- Field discovery: 1 API call (cached)
- Metadata validation: 1 API call per project+type (cached)
- Search: 1 API call
- Update: 1 API call
- Transition: 2 API calls (get transitions + execute)

### Dependencies

- `github.com/spf13/cobra` - CLI framework
- `github.com/go-resty/resty/v2` - HTTP client
- `gopkg.in/yaml.v3` - YAML parsing
- `github.com/schollz/progressbar/v3` - Progress bars

### Security

- API tokens stored in config file with 0600 permissions
- Basic authentication over HTTPS
- No credentials in command output
- Secure credential validation

### Known Limitations

- Custom field display limited to common field IDs (10014, 10016)
- No bulk update command (single issue only)
- Epic Link field ID must be discovered per instance

---

## Release Notes

### Version 1.0.0 - Initial Release

This is the first stable release of jira-cli, a command-line interface for Jira Cloud designed for developer productivity and AI-assisted workflows.

**Highlights:**
- Complete CRUD operations for Jira issues
- Batch creation with automatic epic linking
- Template system for common issue types
- Custom field support with aliases
- JQL search capabilities
- Comprehensive documentation

**Getting Started:**
```bash
# Install
go install github.com/sanisideup/jira-cli@latest

# Configure
jira-cli configure

# Create issues
jira-cli batch create issues.json
```

For detailed usage instructions, see the [README](README.md).

---

[1.3.0]: https://github.com/sanisideup/jira-cli/releases/tag/v1.3.0
[1.2.0]: https://github.com/sanisideup/jira-cli/releases/tag/v1.2.0
[1.1.0]: https://github.com/sanisideup/jira-cli/releases/tag/v1.1.0
[1.0.0]: https://github.com/sanisideup/jira-cli/releases/tag/v1.0.0
