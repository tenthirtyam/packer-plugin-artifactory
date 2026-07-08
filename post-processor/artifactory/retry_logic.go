// Copyright (c) Ryan Johnson
// SPDX-License-Identifier: MPL-2.0

package artifactory

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/packer"
)

// retryableOperation represents an operation that can be retried.
type retryableOperation func() error

// retryLogic handles retry logic with exponential backoff.
type retryLogic struct {
	maxRetries     int
	timeoutSeconds int
	ui             packer.Ui
}

// newRetryLogic creates a new retryLogic instance.
func newRetryLogic(maxRetries, timeoutSeconds int, ui packer.Ui) *retryLogic {
	return &retryLogic{
		maxRetries:     maxRetries,
		timeoutSeconds: timeoutSeconds,
		ui:             ui,
	}
}

// executeWithRetry executes an operation with retry logic and exponential backoff.
func (rl *retryLogic) executeWithRetry(ctx context.Context, operation retryableOperation, operationName string) error {
	var lastErr error

	for attempt := 0; attempt <= rl.maxRetries; attempt++ {
		err := operation()

		if err == nil {
			return nil
		}

		lastErr = err

		if !rl.shouldRetry(err) {
			rl.ui.Error(fmt.Sprintf("%s failed with non-retryable error: %s", operationName, err))
			return err
		}

		if attempt == rl.maxRetries {
			break
		}

		delay := rl.calculateBackoff(attempt)
		rl.ui.Say(fmt.Sprintf("%s failed (attempt %d/%d): %s. Retrying in %v...",
			operationName, attempt+1, rl.maxRetries+1, err, delay))

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return fmt.Errorf("upload failed after %d attempts: %s", rl.maxRetries+1, lastErr)
}

// shouldRetry determines if an error should trigger a retry.
func (rl *retryLogic) shouldRetry(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	if isNetworkError(err) {
		return true
	}

	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded") {
		rl.ui.Say(fmt.Sprintf("Request timeout after %ds", rl.timeoutSeconds))
		return true
	}

	if strings.Contains(errStr, "connection") || strings.Contains(errStr, "connect") {
		return true
	}

	if strings.Contains(errStr, "http") {
		if strings.Contains(errStr, "5") && (strings.Contains(errStr, "50") || strings.Contains(errStr, "51") || strings.Contains(errStr, "52") || strings.Contains(errStr, "53")) {
			return true
		}

		if strings.Contains(errStr, "401") || strings.Contains(errStr, "403") {
			return true
		}

		if strings.Contains(errStr, "4") && (strings.Contains(errStr, "40") || strings.Contains(errStr, "41") || strings.Contains(errStr, "42") || strings.Contains(errStr, "43") || strings.Contains(errStr, "44")) {
			return false
		}
	}

	return true
}

// calculateBackoff calculates exponential backoff delay with a 60-second cap.
func (rl *retryLogic) calculateBackoff(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}

	// Cap attempt to prevent overflow (max 2^10 = 1024 seconds)
	if attempt > 10 {
		attempt = 10
	}
	delay := min(time.Duration(1<<attempt)*time.Second, 60*time.Second)

	return delay
}

// isNetworkError checks if an error is a network-level error.
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}

	if _, ok := errors.AsType[net.Error](err); ok {
		return true
	}

	if _, ok := errors.AsType[*net.OpError](err); ok {
		return true
	}

	var dnsErr *net.DNSError
	return errors.As(err, &dnsErr)
}
