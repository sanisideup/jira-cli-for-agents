# AGENTS.md — AI Agent & Contributor Guidelines

This document provides guidelines for AI agents (Claude, GPT, Gemini, Copilot, etc.) and contributors working with this codebase.

## Project Overview

`jcfa` is a fast, scriptable CLI for Jira Cloud designed for AI-assisted workflows and developer productivity. It prioritizes:

- **Batch operations** — Create dozens of issues from structured JSON
- **Schema validation** — Catch errors before API submission
- **Custom field support** — Human-friendly aliases instead of cryptic field IDs
- **Scriptability** — JSON input/output for integration with other tools

## Project Structure

```
jira-cli-for-agents/
├── cmd/                # CLI commands (cobra-based)
│   ├── root.go         # Root command, global flags, allowlist integration
│   ├── create.go       # Issue creation
│   ├── batch.go        # Batch operations
│   ├── get.go          # Issue retrieval
│   ├── search.go       # JQL search
│   └── ...
├── pkg/
│   ├── client/         # HTTP client with retry/rate limiting
│   ├── config/         # Configuration management
│   ├── secrets/        # Secure credential storage (keyring/encrypted file)
│   ├── allowlist/      # Command restriction for agent safety
│   ├── jira/           # Jira API service layer
│   ├── models/         # Data structures
│   └── template/       # Template rendering engine
├── templates/          # Default issue templates
├── docs/               # Additional documentation
├── examples/           # Usage examples
├── ARCHITECTURE.md     # Detailed system architecture
├── CONTRIBUTING.md     # Contribution guidelines
└── README.md           # User documentation
```

## Build & Development

```bash
# Build
go build -o jira-cli-for-agents

# Run tests
go test ./...

# Run specific package tests
go test ./pkg/config/...

# Build with version info
go build -ldflags "-X main.version=1.0.0" -o jcfa
```

## Coding Conventions

### Go Style
- Follow standard Go formatting (`gofmt`)
- Use meaningful variable names
- Keep functions focused and small
- Add comments for exported functions

### Command Structure
Commands use [Cobra](https://github.com/spf13/cobra):

```go
var myCmd = &cobra.Command{
    Use:   "mycommand",
    Short: "Brief description",
    Long:  `Detailed description`,
    RunE: func(cmd *cobra.Command, args []string) error {
        // Implementation
        return nil
    },
}
```

### Error Handling
Use specific exit codes (defined in `cmd/root.go`):
- `0`: Success
- `1`: Authentication failure
- `2`: Validation error
- `3`: API error
- `4`: Configuration error

### Output Conventions
- **stdout**: Data output (JSON, tables)
- **stderr**: Progress, hints, errors
- Support `--json` flag for machine-readable output
- Human-readable output should be the default

## Security Guidelines

### Never Commit Secrets
- API tokens
- Credentials files
- Personal configuration

### Credential Storage
The CLI supports secure credential storage via:
1. **OS Keyring** (macOS Keychain, Windows Credential Manager, Linux Secret Service)
2. **Encrypted file** (for headless/CI environments)

Configure via:
```bash
export JIRA_KEYRING_BACKEND=keychain  # or "file"
export JIRA_KEYRING_PASSWORD=...      # Required for file backend
```

### Agent Safety Features

#### Command Allowlist
Restrict commands for sandboxed/agent execution:

```bash
# Only allow read operations
export JIRA_READONLY=1

# Or specify exact commands
export JIRA_COMMAND_ALLOWLIST="get,search,list,fields"
```

This prevents accidentally running destructive operations.

#### Read-Only Commands
Safe:
- `get`, `search`, `list`
- `fields`, `version`, `help`
- `attachment list`, `comments list`, `link list`

#### Write Commands (Require Explicit Allowlisting)
- `create`, `update`, `transition`
- `batch`, `comment`, `link create`
- `attachment upload/delete`

## Testing Guidelines

### Unit Tests
```go
func TestMyFunction(t *testing.T) {
    // Arrange
    input := "test"
    
    // Act
    result := MyFunction(input)
    
    // Assert
    if result != expected {
        t.Errorf("expected %v, got %v", expected, result)
    }
}
```

### Integration Tests
For tests that hit the Jira API:
- Use a test project
- Clean up created issues
- Skip if credentials not available

```go
func TestJiraIntegration(t *testing.T) {
    if os.Getenv("JIRA_TEST_PROJECT") == "" {
        t.Skip("JIRA_TEST_PROJECT not set")
    }
    // ...
}
```

## Commit Guidelines

### Conventional Commits
Use conventional commit format:

```
feat(cmd): add --dry-run flag to batch create
fix(client): handle rate limit with exponential backoff
docs(readme): add installation instructions
refactor(template): simplify field resolution logic
```

### PR Guidelines
- Keep PRs focused on a single concern
- Include tests for new functionality
- Update documentation as needed
- Note any breaking changes

## Common Tasks

### Adding a New Command

1. Create `cmd/mycommand.go`:
```go
package cmd

import "github.com/spf13/cobra"

var myCmd = &cobra.Command{
    Use:   "mycommand",
    Short: "Brief description",
    RunE:  runMyCommand,
}

func init() {
    rootCmd.AddCommand(myCmd)
    myCmd.Flags().StringP("flag", "f", "", "Flag description")
}

func runMyCommand(cmd *cobra.Command, args []string) error {
    // Check allowlist if this is a write operation
    if err := allowlistChecker.Check("mycommand"); err != nil {
        return err
    }
    // Implementation
    return nil
}
```

2. Add to allowlist in `pkg/allowlist/allowlist.go` (ReadOnlyCommands or WriteCommands)

### Adding a New Field Type Handler

See `pkg/jira/field_handlers.go` for examples of how to handle custom Jira field types.

### Adding Template Support

Templates live in `templates/` and use Go's `text/template` syntax.

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `JIRA_READONLY` | Enable read-only mode | (disabled) |
| `JIRA_COMMAND_ALLOWLIST` | Comma-separated allowed commands | (all allowed) |
| `JIRA_KEYRING_BACKEND` | Credential storage: `auto`, `keychain`, `file` | `auto` |
| `JIRA_KEYRING_PASSWORD` | Password for file backend | (required if file) |

## Troubleshooting

### Common Issues

**"config file not found"**
```bash
jcfa configure
```

**"authentication failed"**
- Check API token at https://id.atlassian.com/manage/api-tokens
- Verify email matches Atlassian account

**"command not in allowlist"**
- Check `JIRA_COMMAND_ALLOWLIST` or `JIRA_READONLY`
- Add command to allowlist if intentional

**"keyring unavailable"**
- Use file backend: `export JIRA_KEYRING_BACKEND=file`
- Set password: `export JIRA_KEYRING_PASSWORD=...`

## Resources

- [ARCHITECTURE.md](ARCHITECTURE.md) — Detailed system design
- [CONTRIBUTING.md](CONTRIBUTING.md) — Contribution guidelines
- [README.md](README.md) — User documentation
- [Jira REST API](https://developer.atlassian.com/cloud/jira/platform/rest/v3/) — Official API docs
