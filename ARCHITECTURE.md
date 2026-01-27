# Architecture Documentation

## Table of Contents
- [Overview](#overview)
- [Design Principles](#design-principles)
- [System Architecture](#system-architecture)
- [API Integration Patterns](#api-integration-patterns)
- [Data Flow](#data-flow)
- [Epic-Story Hierarchy Handling](#epic-story-hierarchy-handling)
- [Field Type System](#field-type-system)
- [Template Rendering Engine](#template-rendering-engine)
- [Error Handling Strategy](#error-handling-strategy)
- [Performance Optimizations](#performance-optimizations)
- [Security Considerations](#security-considerations)
- [Testing Strategy](#testing-strategy)

---

## Overview

The Jira CLI is a command-line interface for Jira Cloud designed specifically for **AI-assisted workflows** and **programmatic issue creation**. Unlike traditional Jira CLIs that focus on human interaction, this tool prioritizes:

1. **Batch operations** - Create dozens of issues from structured data
2. **Schema validation** - Catch errors before API submission
3. **Field abstraction** - Use human-friendly aliases instead of cryptic field IDs
4. **Scriptability** - JSON input/output for integration with other tools

### Use Case Example
```bash
# AI assistant analyzes a meeting transcript and generates:
cat issues.json | jira batch create --dry-run
# After validation, creates 50 user stories with proper epic linking
```

---

## Design Principles

### 1. **API-First Design**
- All operations map directly to Jira REST API v3 calls
- Minimize client-side logic; let Jira validate when possible
- Cache metadata to reduce API calls (5-minute TTL)

### 2. **Fail Fast**
- Validate credentials on first run (`/rest/api/3/myself`)
- Validate field schemas before issue creation
- Dry-run mode for batch operations
- Detailed error messages with actionable remediation

### 3. **Portability**
- Single binary with no runtime dependencies
- Config file is portable (YAML with field mappings)
- Templates are portable (embedded defaults + user overrides)

### 4. **Extensibility**
- Field type handlers can be added without changing core logic
- Template system supports custom templates in `~/.jcfa/templates/`
- Plugin-style architecture for future extensions (webhooks, automation)

### 5. **Observability**
- Structured logging (JSON in production, human-readable in dev)
- Exit codes map to error types (auth, validation, API, config)
- Progress tracking for long-running operations

---

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        CLI Layer (cmd/)                      │
│  ┌──────────┬──────────┬──────────┬──────────┬───────────┐  │
│  │configure │  fields  │ template │  create  │   batch   │  │
│  │  get     │  search  │  update  │transition│   link    │  │
│  │ allowlist│          │          │          │           │  │
│  └──────────┴──────────┴──────────┴──────────┴───────────┘  │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│                   Service Layer (pkg/)                       │
│  ┌──────────────┬─────────────┬──────────────┬───────────┐  │
│  │IssueService  │FieldService │MetadataService│LinkService│  │
│  │TemplateServ  │ConfigService│ HierarchyServ │ LogService│  │
│  └──────────────┴─────────────┴──────────────┴───────────┘  │
│  ┌──────────────┬─────────────┐                             │
│  │AllowlistChkr │SecretsStore │  (Security Layer)           │
│  └──────────────┴─────────────┘                             │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│                  HTTP Client Layer (pkg/client/)             │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  • Authentication (Basic Auth)                      │    │
│  │  • Retry Logic (Exponential Backoff)                │    │
│  │  • Rate Limiting (Token Bucket: 300 req/min)        │    │
│  │  • Request/Response Logging                         │    │
│  └─────────────────────────────────────────────────────┘    │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│                   Jira Cloud REST API v3                     │
│  https://{domain}.atlassian.net/rest/api/3/*                │
│  https://{domain}.atlassian.net/rest/agile/1.0/*            │
└─────────────────────────────────────────────────────────────┘
```

### Layer Responsibilities

#### **CLI Layer** (`cmd/`)
- Parse command-line arguments and flags
- Load configuration from file
- Coordinate service calls
- Format output (human-readable vs JSON)
- Handle exit codes

#### **Service Layer** (`pkg/`)
- Business logic for operations
- Data validation and transformation
- Template rendering
- Field alias resolution
- Epic/parent linking logic
- Command allowlist validation (`pkg/allowlist/`)
- Secure credential storage (`pkg/secrets/`)

#### **HTTP Client Layer** (`pkg/client/`)
- Low-level HTTP communication
- Authentication header injection
- Retry logic with exponential backoff
- Rate limiting to prevent API bans
- Error response parsing

---

## API Integration Patterns

### Authentication Flow
```
1. User runs: jira configure
2. CLI prompts for: domain, email, api_token
3. CLI validates credentials:
   GET /rest/api/3/myself
   Authorization: Basic base64(email:token)
4. On success (200), save to ~/.jcfa/config.yaml (chmod 0600)
5. All subsequent requests use saved credentials
```

### API Token Generation
Users create tokens at: https://id.atlassian.com/manage-profile/security/api-tokens

**Security Note**: Tokens have full account permissions. Never log or expose tokens in error messages.

---

### Retry Strategy

**Problem**: Jira Cloud has rate limits (~300 requests/minute) and occasional 5xx errors.

**Solution**: Exponential backoff with jitter
```go
// Retry on: 429 (Too Many Requests), 500, 502, 503, 504
// Attempts: 3 total (initial + 2 retries)
// Delays: 1s → 2s → 4s (with random jitter ±20%)

func (c *Client) doWithRetry(req *http.Request) (*http.Response, error) {
    delays := []time.Duration{1*time.Second, 2*time.Second, 4*time.Second}

    for attempt := 0; attempt < len(delays); attempt++ {
        resp, err := c.HTTPClient.Do(req)

        if shouldRetry(resp, err) {
            time.Sleep(delays[attempt] + jitter())
            continue
        }

        return resp, err
    }
}
```

**Why exponential backoff?** Prevents thundering herd problem. If 100 clients hit rate limit simultaneously, staggered retries prevent all retrying at once.

---

### Rate Limiting (Proactive)

**Problem**: Batch operations could trigger rate limits even with retry logic.

**Solution**: Token bucket algorithm (golang.org/x/time/rate)
```go
// Allow 5 requests/second (300/minute) with burst of 10
limiter := rate.NewLimiter(rate.Every(200*time.Millisecond), 10)

// Before each request
limiter.Wait(ctx)  // Blocks until token available
```

**Why client-side rate limiting?** Prevents requests from ever hitting the rate limit, avoiding delays and potential temporary bans.

---

### Pagination Pattern

**Problem**: Jira returns max 50-100 items per page. Large result sets need pagination.

**Solution**: Cursor-based iteration
```go
func (c *Client) SearchAll(jql string) ([]Issue, error) {
    var allIssues []Issue
    startAt := 0
    maxResults := 50

    for {
        resp, err := c.Search(jql, startAt, maxResults)
        if err != nil {
            return nil, err
        }

        allIssues = append(allIssues, resp.Issues...)

        if startAt + len(resp.Issues) >= resp.Total {
            break  // All results fetched
        }

        startAt += maxResults
    }

    return allIssues, nil
}
```

**Used in**: `jira search`, `jira list`, `jira fields list` (for large instances)

---

## Data Flow

### Single Issue Creation
```
User Input (JSON/Template)
    │
    ▼
Template Service
├─ Load template (epic.yaml, story.yaml, etc.)
├─ Render with user data
└─ Resolve field aliases → field IDs
    │
    ▼
Metadata Service
├─ Fetch /rest/api/3/issue/createmeta
├─ Validate required fields present
├─ Validate field types match
└─ Validate allowed values
    │
    ▼
Field Type Handlers
├─ User Picker: email → accountId
├─ Multi-Select: ["val1", "val2"] → [{"value": "val1"}, {"value": "val2"}]
├─ Sprint: name → sprint ID (via Agile API)
└─ Date: string → ISO 8601 format
    │
    ▼
Issue Service
├─ POST /rest/api/3/issue
├─ Parse response
└─ Return created issue key
    │
    ▼
Output Formatter
└─ Human: "✅ Created PROJ-123: User authentication story"
    JSON: {"key": "PROJ-123", "id": "10042", "self": "https://..."}
```

### Batch Issue Creation
```
Batch Input (JSON array)
    │
    ▼
Parser & Validator
├─ Parse JSON
├─ Validate structure
└─ Separate epics from stories
    │
    ├─────────────────┬─────────────────┐
    │                 │                 │
    ▼                 ▼                 ▼
Epic Batch         Story Batch      Other Issues
    │                 │                 │
    ▼                 │                 │
Create Epics         │                 │
├─ POST /rest/api/3/issue/bulk (max 50 per request)
├─ Store mapping: temp_id → PROJ-100, PROJ-101, etc.
└─ Update story references
    │                 │                 │
    └─────────────────▼                 │
                Create Stories          │
                ├─ Replace epic placeholders with real keys
                ├─ POST /rest/api/3/issue/bulk
                └─ Link to epics (if needed)
                      │                 │
                      └─────────────────▼
                                  Create Others
                                        │
                                        ▼
                              Aggregate Results
                              ├─ Successes: 48 issues
                              ├─ Failures: 2 issues (with details)
                              └─ Output summary
```

---

## Epic-Story Hierarchy Handling

### The Problem
Jira has **three different methods** for linking stories to epics, depending on the instance configuration:

1. **Modern (2021+)**: `parent` field (preferred)
2. **Legacy (2017-2020)**: Epic Link custom field (e.g., `customfield_10014`)
3. **Fallback**: Issue Link API with "Epic-Story" link type

### Detection Strategy

```go
type HierarchyMethod string

const (
    HierarchyParent    HierarchyMethod = "parent"
    HierarchyEpicLink  HierarchyMethod = "epic_link"
    HierarchyIssueLink HierarchyMethod = "issue_link"
)

func (s *HierarchyService) DetectMethod(projectKey string) (HierarchyMethod, error) {
    // Step 1: Check if 'parent' field is available for Story issue type
    meta, err := s.metadata.GetCreateMetadata(projectKey, "Story")
    if err != nil {
        return "", err
    }

    if _, exists := meta.Fields["parent"]; exists {
        return HierarchyParent, nil  // Modern Jira
    }

    // Step 2: Check for Epic Link custom field
    fields, err := s.fields.ListFields(projectKey)
    if err != nil {
        return "", err
    }

    for _, field := range fields {
        if field.Name == "Epic Link" || field.ClauseNames.Contains("Epic Link") {
            // Cache this field ID in config
            s.config.FieldMappings["epic_link"] = field.ID
            s.config.Save()
            return HierarchyEpicLink, nil
        }
    }

    // Step 3: Fallback to issue link API
    return HierarchyIssueLink, nil
}
```

### Linking Implementation

```go
func (s *HierarchyService) LinkToEpic(storyKey, epicKey string) error {
    method := s.cachedMethod  // Detect once, cache forever

    switch method {
    case HierarchyParent:
        // PUT /rest/api/3/issue/{storyKey}
        return s.client.UpdateIssue(storyKey, map[string]interface{}{
            "fields": map[string]interface{}{
                "parent": map[string]string{"key": epicKey},
            },
        })

    case HierarchyEpicLink:
        // PUT /rest/api/3/issue/{storyKey}
        epicLinkFieldID := s.config.FieldMappings["epic_link"]
        return s.client.UpdateIssue(storyKey, map[string]interface{}{
            "fields": map[string]interface{}{
                epicLinkFieldID: epicKey,
            },
        })

    case HierarchyIssueLink:
        // POST /rest/api/3/issueLink
        return s.client.CreateIssueLink(map[string]interface{}{
            "type": map[string]string{"name": "Epic-Story"},
            "inwardIssue":  map[string]string{"key": storyKey},
            "outwardIssue": map[string]string{"key": epicKey},
        })
    }
}
```

**Caching**: Method detection result is cached in config to avoid repeated API calls:
```yaml
# ~/.jcfa/config.yaml
hierarchy_method: "parent"  # or "epic_link" or "issue_link"
field_mappings:
  epic_link: "customfield_10014"  # Only if using epic_link method
```

---

## Field Type System

### Challenge
Jira has complex field types with different JSON structures:

| Field Type | Display Name | API Format |
|------------|--------------|------------|
| String | Text field | `"value"` |
| Number | Number field | `42` |
| User | User picker | `{"accountId": "5b10a2844c20165700ede21g"}` |
| Option | Single select | `{"value": "High"}` |
| Array | Multi-select | `[{"value": "Bug"}, {"value": "Security"}]` |
| Date | Date picker | `"2024-01-15"` (YYYY-MM-DD) |
| DateTime | Date time picker | `"2024-01-15T14:30:00.000+0000"` (ISO 8601) |
| Parent | Epic/parent | `{"key": "PROJ-100"}` |

### Field Type Handler Pattern

```go
type FieldTypeHandler interface {
    // Validate checks if value is valid for this field type
    Validate(value interface{}, meta FieldMeta) error

    // Format converts user-friendly input to Jira API format
    Format(value interface{}, meta FieldMeta) (interface{}, error)
}

// Example: User Picker Handler
type UserPickerHandler struct {
    client *client.Client
}

func (h *UserPickerHandler) Format(value interface{}, meta FieldMeta) (interface{}, error) {
    email := value.(string)

    // Search for user by email
    users, err := h.client.SearchUsers(email)
    if err != nil {
        return nil, err
    }

    if len(users) == 0 {
        return nil, fmt.Errorf("user not found: %s", email)
    }

    return map[string]string{
        "accountId": users[0].AccountId,
    }, nil
}

// Example: Multi-Select Handler
type MultiSelectHandler struct{}

func (h *MultiSelectHandler) Format(value interface{}, meta FieldMeta) (interface{}, error) {
    values := value.([]string)
    result := make([]map[string]string, len(values))

    for i, v := range values {
        // Validate against allowedValues if present
        if !h.isAllowed(v, meta.AllowedValues) {
            return nil, fmt.Errorf("value '%s' not in allowed values", v)
        }
        result[i] = map[string]string{"value": v}
    }

    return result, nil
}
```

### Handler Registration

```go
var fieldTypeHandlers = map[string]FieldTypeHandler{
    "user":     &UserPickerHandler{},
    "option":   &OptionHandler{},
    "array":    &MultiSelectHandler{},
    "date":     &DateHandler{},
    "datetime": &DateTimeHandler{},
    "parent":   &ParentHandler{},
}

func GetHandler(schema Schema) FieldTypeHandler {
    // Determine handler based on schema.Type and schema.System/Custom
    if schema.Type == "user" {
        return fieldTypeHandlers["user"]
    }
    if schema.Type == "array" && schema.Items == "option" {
        return fieldTypeHandlers["array"]
    }
    // ... etc
}
```

---

## Template Rendering Engine

### Template Format (JSON with Go Template Syntax)

**Why JSON instead of YAML?**
- No syntax overlap with Go template syntax (`{{ }}`)
- Easier to validate structure
- Direct mapping to Jira API JSON format

**Example Template** (`~/.jcfa/templates/story.json`):
```json
{
  "type": "Story",
  "fields": {
    "project": {"key": "{{.Project}}"},
    "summary": "{{.Summary}}",
    "description": {{.Description | toJson}},
    "assignee": {{if .Assignee}}{"accountId": "{{.Assignee | resolveUser}}"}{{else}}null{{end}},
    "priority": {"name": "{{.Priority | default "Medium"}}"},
    "labels": {{.Labels | toJson}},
    "{{resolveField "story_points"}}": {{.StoryPoints | default "null"}},
    "{{resolveField "epic_link"}}": {{if .EpicKey}}"{{.EpicKey}}"{{else}}null{{end}}
  }
}
```

### Custom Template Functions

```go
var templateFuncs = template.FuncMap{
    // Convert Go value to JSON
    "toJson": func(v interface{}) string {
        b, _ := json.Marshal(v)
        return string(b)
    },

    // Provide default value if input is nil/empty
    "default": func(defaultVal, val interface{}) interface{} {
        if val == nil || val == "" {
            return defaultVal
        }
        return val
    },

    // Resolve field alias to field ID
    "resolveField": func(aliasOrID string) string {
        if fieldID, exists := config.FieldMappings[aliasOrID]; exists {
            return fieldID
        }
        return aliasOrID  // Already a field ID
    },

    // Resolve email to Jira accountId
    "resolveUser": func(email string) string {
        users, _ := client.SearchUsers(email)
        if len(users) > 0 {
            return users[0].AccountId
        }
        return ""
    },
}
```

### Rendering Process

```go
func (s *TemplateService) Render(templateName string, data map[string]interface{}) (map[string]interface{}, error) {
    // 1. Load template file
    tmplContent, err := s.loadTemplate(templateName)
    if err != nil {
        return nil, err
    }

    // 2. Parse as Go template
    tmpl, err := template.New(templateName).Funcs(templateFuncs).Parse(tmplContent)
    if err != nil {
        return nil, fmt.Errorf("template parse error: %w", err)
    }

    // 3. Render template with data
    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, data); err != nil {
        return nil, fmt.Errorf("template execution error: %w", err)
    }

    // 4. Parse rendered JSON
    var result map[string]interface{}
    if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
        return nil, fmt.Errorf("rendered template is not valid JSON: %w", err)
    }

    // 5. Apply field type formatting
    fields := result["fields"].(map[string]interface{})
    for fieldID, value := range fields {
        if value == nil {
            delete(fields, fieldID)  // Remove null fields
            continue
        }

        // Get field metadata to determine type
        meta := s.metadata.GetFieldMeta(result["type"].(string), fieldID)
        handler := GetHandler(meta.Schema)

        // Format value for Jira API
        formatted, err := handler.Format(value, meta)
        if err != nil {
            return nil, fmt.Errorf("field %s: %w", fieldID, err)
        }
        fields[fieldID] = formatted
    }

    return result, nil
}
```

---

## Error Handling Strategy

### Error Types

```go
// pkg/errors/errors.go

type ErrorType int

const (
    ErrAuth       ErrorType = 1  // Authentication failure
    ErrValidation ErrorType = 2  // Client-side validation failure
    ErrAPI        ErrorType = 3  // Jira API error
    ErrConfig     ErrorType = 4  // Configuration error
)

type CLIError struct {
    Type    ErrorType
    Message string
    Cause   error
    Context map[string]interface{}  // Additional context for debugging
}

func (e *CLIError) Error() string {
    return fmt.Sprintf("[%s] %s: %v", e.TypeString(), e.Message, e.Cause)
}

func (e *CLIError) ExitCode() int {
    return int(e.Type)
}
```

### Error Handling in Commands

```go
// cmd/create.go

func runCreate(cmd *cobra.Command, args []string) error {
    // ... create logic ...

    if err != nil {
        // Convert to CLIError if not already
        cliErr, ok := err.(*errors.CLIError)
        if !ok {
            cliErr = &errors.CLIError{
                Type:    errors.ErrAPI,
                Message: "Issue creation failed",
                Cause:   err,
            }
        }

        // Print user-friendly error
        fmt.Fprintf(os.Stderr, "❌ %s\n", cliErr.Message)
        if verbose {
            fmt.Fprintf(os.Stderr, "Details: %v\n", cliErr.Cause)
            fmt.Fprintf(os.Stderr, "Context: %+v\n", cliErr.Context)
        }

        os.Exit(cliErr.ExitCode())
    }

    return nil
}
```

### Jira API Error Parsing

Jira returns errors in two formats:

**Format 1: Error Messages Array**
```json
{
  "errorMessages": ["Field 'summary' is required."],
  "errors": {}
}
```

**Format 2: Errors Object**
```json
{
  "errorMessages": [],
  "errors": {
    "customfield_10016": "Story Points must be a number"
  }
}
```

**Parser:**
```go
type JiraErrorResponse struct {
    ErrorMessages []string          `json:"errorMessages"`
    Errors        map[string]string `json:"errors"`
}

func parseJiraError(resp *http.Response) error {
    var jiraErr JiraErrorResponse
    json.NewDecoder(resp.Body).Decode(&jiraErr)

    var messages []string
    messages = append(messages, jiraErr.ErrorMessages...)

    for field, msg := range jiraErr.Errors {
        messages = append(messages, fmt.Sprintf("%s: %s", field, msg))
    }

    return &CLIError{
        Type:    determineErrorType(resp.StatusCode),
        Message: strings.Join(messages, "; "),
        Context: map[string]interface{}{
            "status_code": resp.StatusCode,
            "url":         resp.Request.URL.String(),
        },
    }
}

func determineErrorType(statusCode int) ErrorType {
    switch statusCode {
    case 401, 403:
        return ErrAuth
    case 400:
        return ErrValidation
    default:
        return ErrAPI
    }
}
```

---

## Performance Optimizations

### 1. Metadata Caching
**Problem**: `/issue/createmeta` is slow (~500ms) and rarely changes.

**Solution**: In-memory cache with TTL
```go
type MetadataCache struct {
    data      map[string]*CacheEntry
    ttl       time.Duration
    mu        sync.RWMutex
}

type CacheEntry struct {
    metadata  *IssueTypeMeta
    timestamp time.Time
}

func (c *MetadataCache) Get(key string) (*IssueTypeMeta, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    entry, exists := c.data[key]
    if !exists || time.Since(entry.timestamp) > c.ttl {
        return nil, false
    }

    return entry.metadata, true
}
```

**Impact**: Batch creation of 50 issues: 50 API calls → 1 API call

---

### 2. Parallel Batch Requests
**Problem**: Creating 200 issues requires 4 sequential bulk API calls (50 each).

**Solution**: Parallel goroutines with semaphore
```go
func (s *IssueService) BulkCreateParallel(issues []map[string]interface{}) (*BulkCreateResponse, error) {
    chunks := chunkSlice(issues, 50)
    results := make([]*BulkCreateResponse, len(chunks))
    errors := make([]error, len(chunks))

    var wg sync.WaitGroup
    sem := make(chan struct{}, 3)  // Max 3 concurrent requests

    for i, chunk := range chunks {
        wg.Add(1)
        go func(idx int, c []map[string]interface{}) {
            defer wg.Done()
            sem <- struct{}{}        // Acquire
            defer func() { <-sem }() // Release

            results[idx], errors[idx] = s.client.BulkCreate(c)
        }(i, chunk)
    }

    wg.Wait()

    // Aggregate results...
}
```

**Impact**: 200 issues: 8s → 3s (with 3 parallel requests)

---

### 3. Connection Pooling
**Problem**: Each request creates new TCP connection (expensive).

**Solution**: HTTP client with keep-alive
```go
httpClient := &http.Client{
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
    },
    Timeout: 30 * time.Second,
}
```

**Impact**: 100 requests: 5s → 2s (reused connections)

---

## Security Considerations

### 1. API Token Storage
- **Storage**: `~/.jcfa/config.yaml` with `chmod 0600` (owner-only)
- **Never**: Log tokens, include in error messages, commit to git
- **Environment override**: Support `JIRA_API_TOKEN` env var for CI/CD

### 2. Credential Validation
- Validate on first setup (`configure` command)
- Re-validate if API returns 401 (expired token)
- Clear error message: "Authentication failed. Run 'jira configure' to update credentials."

### 3. Sensitive Field Handling
- Sanitize user input before logging
- Redact fields like `password`, `secret` in debug output

### 4. HTTPS Enforcement
- Always use HTTPS (upgrade HTTP to HTTPS)
- Validate TLS certificates
- Allow `--insecure` flag only for testing (with warning)

---

## Credential Storage Layer (`pkg/secrets/`)

**Problem**: API tokens in plaintext config files are a security risk, especially on shared systems.

**Solution**: Multi-backend secure storage system with automatic platform detection.

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Store Interface                          │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  Store(account, creds)  Retrieve(account)  Delete() │    │
│  └─────────────────────────────────────────────────────┘    │
└────────────────────────┬────────────────────────────────────┘
                         │
         ┌───────────────┼───────────────┐
         │               │               │
         ▼               ▼               ▼
┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│  Keychain   │  │    File     │  │    Auto     │
│  Backend    │  │   Backend   │  │  Selector   │
├─────────────┤  ├─────────────┤  ├─────────────┤
│ macOS:      │  │ Encrypted   │  │ Platform    │
│  Keychain   │  │ JSON file   │  │ detection   │
│ Windows:    │  │ + password  │  │ CI/env      │
│  Credential │  │             │  │ awareness   │
│  Manager    │  │             │  │             │
│ Linux:      │  │             │  │             │
│  Secret Svc │  │             │  │             │
└─────────────┘  └─────────────┘  └─────────────┘
```

### Backend Selection Logic

```go
func selectBestBackend() Backend {
    // CI environments use file backend
    if os.Getenv("CI") != "" {
        return BackendFile
    }
    // Explicit override
    if os.Getenv("JIRA_KEYRING_BACKEND") == "file" {
        return BackendFile
    }

    switch runtime.GOOS {
    case "darwin", "windows":
        return BackendKeychain  // OS keyring available
    case "linux":
        if hasDisplay() {       // GUI available
            return BackendKeychain
        }
        return BackendFile      // Headless/SSH
    default:
        return BackendFile
    }
}
```

### Backends

| Backend | Use Case | Security | Env Vars |
|---------|----------|----------|----------|
| `keychain` | Interactive (macOS/Windows) | OS-level encryption | None required |
| `file` | CI/SSH/Headless | Password-based encryption | `JIRA_KEYRING_PASSWORD` |
| `auto` | Default | Selects best for platform | Optional overrides |

### Integration Flow

```
configure command
    │
    ├─► Prompt: "Store API token securely in system keyring? [Y/n]"
    │
    ├─► Yes ─► NewStore(BackendAuto)
    │         ├─► Store(email, {APIToken})
    │         ├─► Set cfg.UseKeyring = true
    │         └─► Set cfg.KeyringBackend = detected
    │
    └─► No ──► Store token in config.yaml (legacy)

root command (on every request)
    │
    └─► if cfg.UseKeyring
        └─► store.Retrieve(email) ─► cfg.APIToken
```

---

## Command Allowlist Layer (`pkg/allowlist/`)

**Problem**: AI agents or sandboxed scripts may accidentally execute destructive commands (create, delete, update).

**Solution**: Environment-based command restriction system.

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                       Checker                                │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  IsAllowed(cmd)  Check(cmd)  GetAllowedCommands()   │    │
│  └─────────────────────────────────────────────────────┘    │
└────────────────────────┬────────────────────────────────────┘
                         │
         ┌───────────────┴───────────────┐
         │                               │
         ▼                               ▼
┌─────────────────────┐      ┌─────────────────────┐
│  Read-Only Mode     │      │  Explicit Allowlist │
│  JIRA_READONLY=1    │      │  JIRA_COMMAND_      │
├─────────────────────┤      │  ALLOWLIST=cmd,cmd  │
│ Auto-populates from │      ├─────────────────────┤
│ ReadOnlyCommands[]  │      │ User specifies      │
│                     │      │ exact commands      │
└─────────────────────┘      └─────────────────────┘
```

### Modes

1. **Read-Only Mode** (`JIRA_READONLY=1`)
   - Auto-allows all read commands (get, search, list, etc.)
   - Blocks all write commands (create, update, delete, etc.)
   - Ideal for AI agents that should only observe

2. **Explicit Allowlist** (`JIRA_COMMAND_ALLOWLIST=cmd1,cmd2`)
   - Only specified commands are allowed
   - Fine-grained control for specific use cases

### Command Categories

**Read Commands** (11 total):
- `get`, `search`, `list`, `fields`, `version`, `help`
- `attachment list`, `comments list`, `comments get`
- `link list`, `link types`

**Write Commands** (16 total):
- `create`, `update`, `transition`, `comment`
- `comments add`, `comments update`, `comments delete`
- `batch`, `batch create`
- `link`, `link create`, `link delete`
- `attachment upload`, `attachment delete`
- `configure`, `template`

### Integration Flow

```
root.PersistentPreRunE
    │
    ├─► Skip for: help, version, allowlist (always allowed)
    │
    └─► allowlistChecker.Check(cmdPath)
        │
        ├─► Allowed ─► Continue to command execution
        │
        └─► Blocked ─► Return error with explanation
            "command 'create' is blocked: JIRA_READONLY mode enabled"
```

### CLI Management

The `allowlist` command provides runtime introspection:

```bash
jira-cli allowlist status    # View current configuration
jira-cli allowlist commands  # List commands by category
jira-cli allowlist check X   # Test if command X is allowed
jira-cli allowlist enable    # Show enable instructions
```

---

## Testing Strategy

### Unit Tests
**Target**: Service layer and utilities
**Framework**: `testing` package + `testify/assert`

```go
// pkg/jira/fields_test.go
func TestResolveFieldID(t *testing.T) {
    service := &FieldService{}
    config := &config.Config{
        FieldMappings: map[string]string{
            "story_points": "customfield_10016",
        },
    }

    tests := []struct {
        input    string
        expected string
    }{
        {"story_points", "customfield_10016"},     // Alias
        {"customfield_10016", "customfield_10016"}, // Already ID
        {"summary", "summary"},                     // Standard field
    }

    for _, tt := range tests {
        result := service.ResolveFieldID(tt.input, config)
        assert.Equal(t, tt.expected, result)
    }
}
```

---

### Integration Tests
**Target**: HTTP client and API interactions
**Framework**: `httptest` for mock server

```go
// pkg/client/client_test.go
func TestCreateIssue(t *testing.T) {
    // Mock Jira API server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, "POST", r.Method)
        assert.Equal(t, "/rest/api/3/issue", r.URL.Path)

        // Verify auth header
        auth := r.Header.Get("Authorization")
        assert.Contains(t, auth, "Basic")

        // Return mock response
        w.WriteHeader(http.StatusCreated)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "id":  "10042",
            "key": "PROJ-123",
        })
    }))
    defer server.Close()

    client := NewClient(&config.Config{
        Domain:   server.URL,
        Email:    "test@example.com",
        APIToken: "test-token",
    })

    result, err := client.CreateIssue(map[string]interface{}{
        "fields": map[string]interface{}{
            "project": map[string]string{"key": "PROJ"},
            "summary": "Test issue",
        },
    })

    assert.NoError(t, err)
    assert.Equal(t, "PROJ-123", result.Key)
}
```

---

### End-to-End Tests
**Target**: Full CLI commands
**Framework**: Bash scripts + real Jira sandbox instance

```bash
#!/bin/bash
# test/e2e/create_issue.sh

# Setup
export JIRA_CONFIG=./test/fixtures/config.yaml

# Test: Create issue
output=$(./jira-cli create --template story --data test/fixtures/story.json)
assert_contains "$output" "Created PROJ-"

# Verify issue exists
issue_key=$(echo "$output" | grep -oE 'PROJ-[0-9]+')
verify_output=$(./jira-cli get "$issue_key")
assert_contains "$verify_output" "User authentication"

# Cleanup
./jira-cli delete "$issue_key"
```

---

### Test Coverage Goals
- **Unit tests**: 80%+ coverage
- **Integration tests**: All API endpoints
- **E2E tests**: All commands with realistic data

---

## Performance Benchmarks

### Target Performance
- Single issue creation: <1s
- Batch 50 issues: <5s
- Batch 200 issues: <15s (with parallelization)
- Field discovery: <2s (cached after first call)
- Search 1000 issues: <10s (with pagination)

### Monitoring
```go
// pkg/logger/metrics.go
type Metrics struct {
    APICallDuration   *prometheus.HistogramVec
    BatchSize         *prometheus.HistogramVec
    ErrorRate         *prometheus.CounterVec
}

// Example usage
start := time.Now()
resp, err := client.CreateIssue(fields)
metrics.APICallDuration.WithLabelValues("POST", "/issue").Observe(time.Since(start).Seconds())
```

---

## Future Extensions

### Planned Features
1. **Webhooks** - Listen for Jira events (issue created, updated)
2. **Automation** - Trigger actions based on events
3. **Plugins** - Custom field type handlers
4. **Cloud/Server dual support** - Support Jira Server (not just Cloud)
5. **Offline mode** - Queue operations when offline, sync later

### Plugin Architecture (Future)
```go
// Allow users to register custom field handlers
jira.RegisterFieldHandler("custom_type", &MyCustomHandler{})

// Allow users to register custom commands
jira.RegisterCommand(&MyCustomCommand{})
```

---

## Conclusion

This architecture prioritizes:
1. **Reliability** - Retry logic, validation, error handling
2. **Performance** - Caching, parallelization, connection pooling
3. **Usability** - Field aliases, templates, dry-run mode
4. **Security** - Token protection, HTTPS enforcement
5. **Extensibility** - Handlers, templates, plugins

The design is informed by real-world Jira API quirks (epic linking variations, field contexts, rate limits) and optimized for AI-assisted batch operations.

---

**Document Version**: 1.1
**Last Updated**: 2026-01-26
**Maintainer**: CLI Development Team
