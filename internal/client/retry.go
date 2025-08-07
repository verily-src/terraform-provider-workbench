package client

import (
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
)

const (
	// INITIAL_INTERVAL is the initial interval between retries.
	INITIAL_INTERVAL = 5 * time.Second
	// MAX_ELAPSED_TIME is the maximum total time for retries.
	MAX_ELAPSED_TIME = 2 * time.Minute
	// MAX_INTERVAL is the maximum interval between retries.
	MAX_INTERVAL = 250 * time.Millisecond
	// MULTIPLIER is the multiplier for increasing the interval after each retry.
	MULTIPLIER = 1.5
)

// RetryClient is a client for retrying operations with exponential backoff.
type RetryClient struct {
	backoff *backoff.ExponentialBackOff
}

// NewRetryClient creates a retry configuration.
func NewRetryClient() *RetryClient {
	var expBackoff = backoff.NewExponentialBackOff()
	expBackoff.InitialInterval = INITIAL_INTERVAL
	expBackoff.MaxElapsedTime = MAX_ELAPSED_TIME
	expBackoff.MaxInterval = MAX_INTERVAL
	expBackoff.Multiplier = MULTIPLIER

	return &RetryClient{
		backoff: expBackoff,
	}
}

// Retry retries a function up to the max elapsed time with exponential backoff.
func (rc *RetryClient) Retry(retryableFunc func() error) error {
	err := backoff.Retry(func() error {
		return retryableFunc()
	}, rc.backoff)

	if err != nil {
		return fmt.Errorf("retry timeout after %v: %v", MAX_ELAPSED_TIME, err)
	}

	return nil
}
