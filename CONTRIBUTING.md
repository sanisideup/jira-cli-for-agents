# Contributing to Jira CLI

Thank you for your interest in contributing to Jira CLI! This document provides guidelines and information for contributors.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [How to Contribute](#how-to-contribute)
- [Development Setup](#development-setup)
- [Code Style Guidelines](#code-style-guidelines)
- [Testing](#testing)
- [Submitting Changes](#submitting-changes)
- [Reporting Bugs](#reporting-bugs)
- [Requesting Features](#requesting-features)

## Code of Conduct

By participating in this project, you agree to maintain a respectful and inclusive environment for everyone. Please be kind and constructive in all interactions.

## Getting Started

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/jira-cli.git
   cd jira-cli
   ```
3. Add the upstream remote:
   ```bash
   git remote add upstream https://github.com/sanisideup/jira-cli.git
   ```
4. Create a branch for your changes:
   ```bash
   git checkout -b feature/your-feature-name
   ```

## How to Contribute

### Types of Contributions

- **Bug fixes**: Fix issues reported in the issue tracker
- **Features**: Implement new functionality (please discuss first in an issue)
- **Documentation**: Improve README, code comments, or add examples
- **Tests**: Add or improve test coverage
- **Refactoring**: Code improvements that don't change functionality

## Development Setup

### Prerequisites

- Go 1.21 or later
- A Jira Cloud instance for testing (optional but recommended)
- Git

### Building

```bash
# Build the binary
go build -o jira-cli main.go

# Build with version info
go build -ldflags="-X 'github.com/sanisideup/jira-cli/cmd.Version=dev'" -o jira-cli main.go
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for a specific package
go test -v ./pkg/config
```

## Code Style Guidelines

This project follows standard Go conventions:

### General Rules

1. **Format code** with `gofmt` or `go fmt ./...`
2. **Use `go vet`** to catch common mistakes: `go vet ./...`
3. **Follow naming conventions**:
   - Use `camelCase` for unexported identifiers
   - Use `PascalCase` for exported identifiers
   - Use descriptive names that convey meaning
4. **Keep functions focused** on a single responsibility
5. **Handle errors explicitly** - never ignore errors silently
6. **Add comments** for all exported functions and types

### Code Organization

```
jira-cli/
├── cmd/           # CLI commands (Cobra)
├── pkg/
│   ├── allowlist/ # Command restriction for sandboxed execution
│   ├── client/    # Jira API client
│   ├── config/    # Configuration management
│   ├── jira/      # Jira API services
│   ├── models/    # Data structures
│   ├── secrets/   # Secure credential storage (keyring/file)
│   └── template/  # Issue templates
├── examples/      # Example files and scripts
└── docs/          # Documentation
```

### Adding a New Command

1. Create `cmd/yourcommand.go`
2. Define the command using Cobra
3. Register with the root command in `init()`
4. Add tests in `cmd/yourcommand_test.go`
5. Update README.md with usage examples

### Adding an API Endpoint

1. Define request/response types in `pkg/models/`
2. Add the method to `pkg/client/client.go`
3. Handle errors appropriately
4. Add retry logic if needed for rate limits

## Testing

### Test Requirements

- All new features should include tests
- Bug fixes should include a test that reproduces the bug
- Aim for meaningful test coverage, not just high percentages

### Testing Against Jira

For integration testing, you'll need:
1. A Jira Cloud instance
2. API token from https://id.atlassian.com/manage/api-tokens
3. A test project with appropriate permissions

**Important**: Never commit real credentials or test data.

### Security Feature Testing

#### Allowlist Testing

```bash
# Test with read-only mode
JIRA_READONLY=1 go test ./pkg/allowlist/...

# Test with custom allowlist
JIRA_COMMAND_ALLOWLIST="get,search" go test ./pkg/allowlist/...

# Run all allowlist tests (includes edge cases)
go test -v ./pkg/allowlist/...
```

#### Credential Storage Testing

```bash
# Test file backend (safe for CI)
JIRA_KEYRING_BACKEND=file JIRA_KEYRING_PASSWORD=test go test ./pkg/secrets/...

# Keychain tests require interactive session on macOS/Windows
# They are automatically skipped on unsupported platforms
go test -v ./pkg/secrets/...
```

### Environment Variables for Testing

| Variable | Description |
|----------|-------------|
| `JIRA_READONLY` | Enable read-only mode for allowlist tests |
| `JIRA_COMMAND_ALLOWLIST` | Comma-separated list of allowed commands |
| `JIRA_KEYRING_BACKEND` | Credential storage: `auto`, `keychain`, `file` |
| `JIRA_KEYRING_PASSWORD` | Password for file backend tests |
| `JIRA_TEST_PROJECT` | Project key for integration tests |
| `CI` | Set automatically in CI; triggers file backend |

## Submitting Changes

### Pull Request Process

1. **Update your branch** with the latest upstream changes:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Ensure all tests pass**:
   ```bash
   go test ./...
   go vet ./...
   ```

3. **Push your changes**:
   ```bash
   git push origin feature/your-feature-name
   ```

4. **Create a Pull Request** on GitHub with:
   - A clear title describing the change
   - A description of what the PR does and why
   - Reference to any related issues (e.g., "Fixes #123")
   - Screenshots or examples if applicable

### PR Guidelines

- Keep PRs focused on a single change
- Write clear commit messages
- Respond to review feedback promptly
- Squash commits if requested

## Reporting Bugs

When reporting bugs, please include:

1. **Summary**: A clear, concise description of the bug
2. **Steps to reproduce**: Detailed steps to reproduce the issue
3. **Expected behavior**: What you expected to happen
4. **Actual behavior**: What actually happened
5. **Environment**:
   - OS and version
   - Go version (`go version`)
   - jira-cli version (`jira-cli version`)
6. **Logs/Output**: Relevant error messages or output (with sensitive data removed)

## Requesting Features

For feature requests, please:

1. **Check existing issues** to avoid duplicates
2. **Describe the feature** clearly and concisely
3. **Explain the use case**: Why is this feature needed?
4. **Provide examples**: How would you use this feature?

## Questions?

If you have questions about contributing, feel free to:
- Open a GitHub issue with the "question" label
- Check existing documentation and issues first

Thank you for contributing!
