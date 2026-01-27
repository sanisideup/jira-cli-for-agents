package client

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/sanisideup/jira-cli-for-agents/pkg/config"
)

// Client represents a Jira API client
type Client struct {
	BaseURL    string
	Email      string
	APIToken   string
	HTTPClient *resty.Client
}

// New creates a new Jira API client from config
func New(cfg *config.Config) *Client {
	client := &Client{
		BaseURL:  cfg.GetBaseURL(),
		Email:    cfg.Email,
		APIToken: cfg.APIToken,
	}

	// Initialize resty client
	client.HTTPClient = resty.New().
		SetBaseURL(client.BaseURL).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetTimeout(30 * time.Second).
		SetRetryCount(3).
		SetRetryWaitTime(1 * time.Second).
		SetRetryMaxWaitTime(4 * time.Second).
		AddRetryCondition(func(r *resty.Response, err error) bool {
			// Retry on rate limits (429) and server errors (5xx)
			return r.StatusCode() == 429 || r.StatusCode() >= 500
		})

	// Set authentication header
	authHeader := client.getAuthHeader()
	client.HTTPClient.SetHeader("Authorization", authHeader)

	return client
}

// getAuthHeader returns the Basic Auth header value
func (c *Client) getAuthHeader() string {
	credentials := fmt.Sprintf("%s:%s", c.Email, c.APIToken)
	encoded := base64.StdEncoding.EncodeToString([]byte(credentials))
	return fmt.Sprintf("Basic %s", encoded)
}

// GetRequest creates a new GET request
func (c *Client) GetRequest() *resty.Request {
	return c.HTTPClient.R()
}

// PostRequest creates a new POST request
func (c *Client) PostRequest() *resty.Request {
	return c.HTTPClient.R()
}

// PutRequest creates a new PUT request
func (c *Client) PutRequest() *resty.Request {
	return c.HTTPClient.R()
}

// DeleteRequest creates a new DELETE request
func (c *Client) DeleteRequest() *resty.Request {
	return c.HTTPClient.R()
}
