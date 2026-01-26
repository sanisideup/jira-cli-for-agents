package client

import (
	"math/rand"
	"time"
)

// CalculateBackoff calculates the backoff duration with jitter
func CalculateBackoff(attempt int) time.Duration {
	// Base backoff: 2^attempt seconds
	baseBackoff := time.Duration(1<<uint(attempt)) * time.Second

	// Max backoff is 30 seconds
	if baseBackoff > 30*time.Second {
		baseBackoff = 30 * time.Second
	}

	// Add jitter: multiply by 0.7 to 1.3
	jitter := 0.7 + rand.Float64()*0.6
	return time.Duration(float64(baseBackoff) * jitter)
}

// ShouldRetry determines if a request should be retried based on status code
func ShouldRetry(statusCode int) bool {
	// Retry on rate limits (429) and server errors (5xx)
	return statusCode == 429 || statusCode >= 500
}
