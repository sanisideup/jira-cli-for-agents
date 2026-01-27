package client

import (
	"fmt"
	"net/http"

	"github.com/sanisideup/jira-cli-for-agents/pkg/models"
)

// ValidateCredentials validates the API credentials by calling the /myself endpoint
func (c *Client) ValidateCredentials() (*models.User, error) {
	var user models.User
	var errorResp models.ErrorResponse

	resp, err := c.GetRequest().
		SetResult(&user).
		SetError(&errorResp).
		Get("/myself")

	if err != nil {
		return nil, fmt.Errorf("failed to validate credentials: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, formatError(resp.StatusCode(), &errorResp)
	}

	return &user, nil
}

// formatError formats an error response from the Jira API
func formatError(statusCode int, errorResp *models.ErrorResponse) error {
	if len(errorResp.ErrorMessages) > 0 {
		return fmt.Errorf("API error (HTTP %d): %s", statusCode, errorResp.ErrorMessages[0])
	}

	if len(errorResp.Errors) > 0 {
		// Get first error from map
		for field, msg := range errorResp.Errors {
			return fmt.Errorf("API error (HTTP %d): %s: %s", statusCode, field, msg)
		}
	}

	return fmt.Errorf("API error (HTTP %d): unknown error", statusCode)
}
