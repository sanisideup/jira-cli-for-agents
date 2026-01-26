package models

// User represents a Jira user
type User struct {
	Self         string `json:"self"`
	AccountID    string `json:"accountId"`
	AccountType  string `json:"accountType"`
	EmailAddress string `json:"emailAddress"`
	DisplayName  string `json:"displayName"`
	Active       bool   `json:"active"`
	TimeZone     string `json:"timeZone"`
	Locale       string `json:"locale"`
}

// Project represents a Jira project
type Project struct {
	Self           string `json:"self"`
	ID             string `json:"id"`
	Key            string `json:"key"`
	Name           string `json:"name"`
	ProjectTypeKey string `json:"projectTypeKey"`
}

// IssueType represents a Jira issue type
type IssueType struct {
	Self           string `json:"self"`
	ID             string `json:"id"`
	Name           string `json:"name"`
	Subtask        bool   `json:"subtask"`
	HierarchyLevel int    `json:"hierarchyLevel"`
}

// FieldSchema represents the schema of a field
type FieldSchema struct {
	Type     string `json:"type"`
	System   string `json:"system,omitempty"`
	Custom   string `json:"custom,omitempty"`
	CustomID int    `json:"customId,omitempty"`
}

// Field represents a Jira field (standard or custom)
type Field struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Custom     bool        `json:"custom"`
	Orderable  bool        `json:"orderable"`
	Navigable  bool        `json:"navigable"`
	Searchable bool        `json:"searchable"`
	Schema     FieldSchema `json:"schema"`
	Required   bool        `json:"required,omitempty"`
}

// Issue represents a Jira issue
type Issue struct {
	ID     string                 `json:"id"`
	Key    string                 `json:"key"`
	Self   string                 `json:"self"`
	Fields map[string]interface{} `json:"fields"`
}

// IssueCreateResult represents the result of creating an issue
type IssueCreateResult struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Self string `json:"self"`
}

// BulkCreateResponse represents the response from bulk create issues
type BulkCreateResponse struct {
	Issues []IssueCreateResult `json:"issues"`
	Errors []BulkCreateError   `json:"errors"`
}

// BulkCreateError represents an error from bulk create
type BulkCreateError struct {
	FailedElementNumber int           `json:"failedElementNumber"`
	ElementErrors       ErrorResponse `json:"elementErrors"`
}

// ErrorResponse represents a Jira API error response
type ErrorResponse struct {
	ErrorMessages []string          `json:"errorMessages"`
	Errors        map[string]string `json:"errors"`
	Status        int               `json:"status,omitempty"`
}

// SearchResponse represents a JQL search response
type SearchResponse struct {
	Expand     string  `json:"expand"`
	StartAt    int     `json:"startAt"`
	MaxResults int     `json:"maxResults"`
	Total      int     `json:"total"`
	Issues     []Issue `json:"issues"`
}

// IssueLinkType represents a type of issue link
type IssueLinkType struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Inward  string `json:"inward"`
	Outward string `json:"outward"`
	Self    string `json:"self"`
}

// IssueLink represents a link between two issues
type IssueLink struct {
	ID           string        `json:"id,omitempty"`           // Link ID for deletion operations
	Self         string        `json:"self,omitempty"`         // API URL of this link
	Type         IssueLinkType `json:"type"`
	OutwardIssue *IssueRef     `json:"outwardIssue,omitempty"`
	InwardIssue  *IssueRef     `json:"inwardIssue,omitempty"`
}

// IssueLinkTypeResponse wraps the array response from /issueLinkType endpoint
type IssueLinkTypeResponse struct {
	IssueLinkTypes []IssueLinkType `json:"issueLinkTypes"`
}

// IssueParent represents a parent issue reference for subtasks
type IssueParent struct {
	ID   string `json:"id,omitempty"`
	Key  string `json:"key,omitempty"`
	Self string `json:"self,omitempty"`
}

// IssueRef represents a reference to an issue
type IssueRef struct {
	ID     string                 `json:"id,omitempty"`
	Key    string                 `json:"key,omitempty"`
	Self   string                 `json:"self,omitempty"`
	Fields map[string]interface{} `json:"fields,omitempty"`
}

// Status represents a workflow status
type Status struct {
	Self           string         `json:"self"`
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	StatusCategory StatusCategory `json:"statusCategory"`
}

// StatusCategory represents a status category
type StatusCategory struct {
	Self      string `json:"self"`
	ID        int    `json:"id"`
	Key       string `json:"key"`
	ColorName string `json:"colorName"`
	Name      string `json:"name"`
}

// CreateMetaResponse represents the create metadata response
type CreateMetaResponse struct {
	Expand   string              `json:"expand"`
	Projects []CreateMetaProject `json:"projects"`
}

// CreateMetaProject represents project info in create metadata
type CreateMetaProject struct {
	Key        string                  `json:"key"`
	Name       string                  `json:"name"`
	IssueTypes []CreateMetaIssueType   `json:"issuetypes"`
}

// CreateMetaIssueType represents issue type info in create metadata
type CreateMetaIssueType struct {
	Name   string               `json:"name"`
	Fields map[string]FieldMeta `json:"fields"`
}

// FieldMeta represents field metadata for issue creation
type FieldMeta struct {
	Required        bool          `json:"required"`
	Schema          FieldSchema   `json:"schema"`
	Name            string        `json:"name"`
	HasDefaultValue bool          `json:"hasDefaultValue"`
	AllowedValues   []interface{} `json:"allowedValues,omitempty"`
	AutoCompleteURL string        `json:"autoCompleteUrl,omitempty"`
}

// Comment represents a comment on an issue
type Comment struct {
	Self         string      `json:"self"`
	ID           string      `json:"id"`
	Author       User        `json:"author"`
	Body         interface{} `json:"body"` // ADF format
	Created      string      `json:"created"`
	Updated      string      `json:"updated"`
	UpdateAuthor User        `json:"updateAuthor,omitempty"`
}

// Transition represents a workflow transition
type Transition struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	To   Status `json:"to"`
}

// TransitionsResponse represents available transitions for an issue
type TransitionsResponse struct {
	Expand      string       `json:"expand"`
	Transitions []Transition `json:"transitions"`
}

// CommentsResponse represents a paginated list of comments
type CommentsResponse struct {
	StartAt    int       `json:"startAt"`
	MaxResults int       `json:"maxResults"`
	Total      int       `json:"total"`
	Comments   []Comment `json:"comments"`
}

// Attachment represents a file attachment on an issue
type Attachment struct {
	Self      string `json:"self"`
	ID        string `json:"id"`
	Filename  string `json:"filename"`
	Author    User   `json:"author"`
	Created   string `json:"created"`
	Size      int64  `json:"size"`
	MimeType  string `json:"mimeType"`
	Content   string `json:"content"`   // Download URL
	Thumbnail string `json:"thumbnail,omitempty"`
}
