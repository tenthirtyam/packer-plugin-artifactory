// Copyright (c) Ryan Johnson
// SPDX-License-Identifier: MPL-2.0

package artifactory

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

// TestRetryLogic_ExecuteWithRetry tests the ExecuteWithRetry method with
// various error scenarios and retry configurations.
func TestRetryLogic_ExecuteWithRetry(t *testing.T) {
	tests := []struct {
		name           string
		maxRetries     int
		timeoutSeconds int
		operation      func() error
		expectError    bool
		errorMsg       string
		expectedCalls  int
	}{
		{
			name:           "successful operation on first try",
			maxRetries:     3,
			timeoutSeconds: 30,
			operation:      func() error { return nil },
			expectError:    false,
			expectedCalls:  1,
		},
		{
			name:           "successful operation on second try",
			maxRetries:     3,
			timeoutSeconds: 30,
			operation: func() func() error {
				calls := 0
				return func() error {
					calls++
					if calls == 1 {
						return fmt.Errorf("temporary network error")
					}
					return nil
				}
			}(),
			expectError:   false,
			expectedCalls: 2,
		},
		{
			name:           "5xx error should be retried",
			maxRetries:     2,
			timeoutSeconds: 30,
			operation:      func() error { return fmt.Errorf("HTTP 500 Internal Server Error") },
			expectError:    true,
			errorMsg:       "upload failed after 3 attempts",
			expectedCalls:  3,
		},
		{
			name:           "4xx error should not be retried",
			maxRetries:     3,
			timeoutSeconds: 30,
			operation:      func() error { return fmt.Errorf("HTTP 404 Not Found") },
			expectError:    true,
			errorMsg:       "HTTP 404 Not Found",
			expectedCalls:  1,
		},
		{
			name:           "401 error should be retried",
			maxRetries:     2,
			timeoutSeconds: 30,
			operation:      func() error { return fmt.Errorf("HTTP 401 Unauthorized") },
			expectError:    true,
			errorMsg:       "upload failed after 3 attempts",
			expectedCalls:  3,
		},
		{
			name:           "403 error should be retried",
			maxRetries:     2,
			timeoutSeconds: 30,
			operation:      func() error { return fmt.Errorf("HTTP 403 Forbidden") },
			expectError:    true,
			errorMsg:       "upload failed after 3 attempts",
			expectedCalls:  3,
		},
		{
			name:           "timeout error should be retried",
			maxRetries:     2,
			timeoutSeconds: 30,
			operation:      func() error { return fmt.Errorf("request timeout") },
			expectError:    true,
			errorMsg:       "upload failed after 3 attempts",
			expectedCalls:  3,
		},
		{
			name:           "connection error should be retried",
			maxRetries:     2,
			timeoutSeconds: 30,
			operation:      func() error { return fmt.Errorf("connection refused") },
			expectError:    true,
			errorMsg:       "upload failed after 3 attempts",
			expectedCalls:  3,
		},
		{
			name:           "max_retries set to 0 should only try once",
			maxRetries:     0,
			timeoutSeconds: 30,
			operation:      func() error { return fmt.Errorf("some error") },
			expectError:    true,
			errorMsg:       "upload failed after 1 attempts",
			expectedCalls:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ui := &capturingTestUI{}
			retryLogic := newRetryLogic(tt.maxRetries, tt.timeoutSeconds, ui)

			calls := 0
			wrappedOperation := func() error {
				calls++
				return tt.operation()
			}

			ctx := context.Background()
			err := retryLogic.executeWithRetry(ctx, wrappedOperation, "test operation")

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			if calls != tt.expectedCalls {
				t.Errorf("Expected %d calls, got %d", tt.expectedCalls, calls)
			}
		})
	}
}

// TestRetryLogic_ShouldRetry tests the shouldRetry method with different
// error types to ensure proper retry decision logic.
func TestRetryLogic_ShouldRetry(t *testing.T) {
	ui := &capturingTestUI{}
	retryLogic := newRetryLogic(3, 30, ui)

	tests := []struct {
		name        string
		err         error
		shouldRetry bool
	}{
		{
			name:        "nil error should not retry",
			err:         nil,
			shouldRetry: false,
		},
		{
			name:        "5xx error should retry",
			err:         fmt.Errorf("HTTP 500 Internal Server Error"),
			shouldRetry: true,
		},
		{
			name:        "502 error should retry",
			err:         fmt.Errorf("HTTP 502 Bad Gateway"),
			shouldRetry: true,
		},
		{
			name:        "401 error should retry",
			err:         fmt.Errorf("HTTP 401 Unauthorized"),
			shouldRetry: true,
		},
		{
			name:        "403 error should retry",
			err:         fmt.Errorf("HTTP 403 Forbidden"),
			shouldRetry: true,
		},
		{
			name:        "404 error should not retry",
			err:         fmt.Errorf("HTTP 404 Not Found"),
			shouldRetry: false,
		},
		{
			name:        "400 error should not retry",
			err:         fmt.Errorf("HTTP 400 Bad Request"),
			shouldRetry: false,
		},
		{
			name:        "timeout error should retry",
			err:         fmt.Errorf("request timeout"),
			shouldRetry: true,
		},
		{
			name:        "deadline exceeded should retry",
			err:         fmt.Errorf("context deadline exceeded"),
			shouldRetry: true,
		},
		{
			name:        "connection error should retry",
			err:         fmt.Errorf("connection refused"),
			shouldRetry: true,
		},
		{
			name:        "network error should retry",
			err:         &net.OpError{Op: "dial", Net: "tcp", Err: fmt.Errorf("connection refused")},
			shouldRetry: true,
		},
		{
			name:        "DNS error should retry",
			err:         &net.DNSError{Name: "example.com", Err: "no such host"},
			shouldRetry: true,
		},
		{
			name:        "unknown error should retry (conservative)",
			err:         fmt.Errorf("some unknown error"),
			shouldRetry: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := retryLogic.shouldRetry(tt.err)
			if result != tt.shouldRetry {
				t.Errorf("shouldRetry() = %v, expected %v for error: %v", result, tt.shouldRetry, tt.err)
			}
		})
	}
}

// TestRetryLogic_CalculateBackoff tests the calculateBackoff method to
// ensure proper exponential backoff timing with maximum cap.
func TestRetryLogic_CalculateBackoff(t *testing.T) {
	ui := &capturingTestUI{}
	retryLogic := newRetryLogic(3, 30, ui)

	tests := []struct {
		name     string
		attempt  int
		expected time.Duration
	}{
		{
			name:     "attempt 0 should be 1 second",
			attempt:  0,
			expected: 1 * time.Second,
		},
		{
			name:     "attempt 1 should be 2 seconds",
			attempt:  1,
			expected: 2 * time.Second,
		},
		{
			name:     "attempt 2 should be 4 seconds",
			attempt:  2,
			expected: 4 * time.Second,
		},
		{
			name:     "attempt 3 should be 8 seconds",
			attempt:  3,
			expected: 8 * time.Second,
		},
		{
			name:     "attempt 4 should be 16 seconds",
			attempt:  4,
			expected: 16 * time.Second,
		},
		{
			name:     "attempt 5 should be 32 seconds",
			attempt:  5,
			expected: 32 * time.Second,
		},
		{
			name:     "attempt 6 should be capped at 60 seconds",
			attempt:  6,
			expected: 60 * time.Second,
		},
		{
			name:     "negative attempt should be 1 second",
			attempt:  -1,
			expected: 1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := retryLogic.calculateBackoff(tt.attempt)
			if result != tt.expected {
				t.Errorf("calculateBackoff(%d) = %v, expected %v", tt.attempt, result, tt.expected)
			}
		})
	}
}

// TestRetryLogic_ContextCancellation tests that retry logic properly
// handles context cancellation during retry attempts.
func TestRetryLogic_ContextCancellation(t *testing.T) {
	ui := &capturingTestUI{}
	retryLogic := newRetryLogic(5, 30, ui)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	calls := 0
	operation := func() error {
		calls++
		if calls == 2 {
			cancel()
		}
		return fmt.Errorf("some retryable error")
	}

	err := retryLogic.executeWithRetry(ctx, operation, "test operation")

	if err == nil {
		t.Errorf("Expected context cancellation error but got nil")
	}
	if calls < 2 {
		t.Errorf("Expected at least 2 calls, got %d", calls)
	}
}

// TestRetryLogic_WrappedErrors tests that network error detection works
// correctly with wrapped errors using errors.As instead of type assertions.
func TestRetryLogic_WrappedErrors(t *testing.T) {
	ui := &capturingTestUI{}
	retryLogic := newRetryLogic(3, 30, ui)

	tests := []struct {
		name        string
		err         error
		shouldRetry bool
	}{
		{
			name:        "direct network error",
			err:         &net.OpError{Op: "dial", Net: "tcp", Err: fmt.Errorf("connection refused")},
			shouldRetry: true,
		},
		{
			name:        "wrapped network error",
			err:         fmt.Errorf("failed to connect: %w", &net.OpError{Op: "dial", Net: "tcp", Err: fmt.Errorf("connection refused")}),
			shouldRetry: true,
		},
		{
			name:        "direct DNS error",
			err:         &net.DNSError{Err: "no such host", Name: "example.com"},
			shouldRetry: true,
		},
		{
			name:        "wrapped DNS error",
			err:         fmt.Errorf("lookup failed: %w", &net.DNSError{Err: "no such host", Name: "example.com"}),
			shouldRetry: true,
		},
		{
			name:        "double wrapped network error",
			err:         fmt.Errorf("outer error: %w", fmt.Errorf("inner error: %w", &net.OpError{Op: "dial", Net: "tcp", Err: fmt.Errorf("connection refused")})),
			shouldRetry: true,
		},
		{
			name:        "non-network error",
			err:         fmt.Errorf("some other error"),
			shouldRetry: true, // Default behavior is to retry
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := retryLogic.shouldRetry(tt.err)
			if result != tt.shouldRetry {
				t.Errorf("shouldRetry() = %v, want %v for error: %v", result, tt.shouldRetry, tt.err)
			}
		})
	}
}

// TestIsNetworkError tests the isNetworkError function specifically
// to ensure it properly handles wrapped errors.
func TestIsNetworkError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		isNetwork bool
	}{
		{
			name:      "nil error",
			err:       nil,
			isNetwork: false,
		},
		{
			name:      "direct OpError",
			err:       &net.OpError{Op: "dial", Net: "tcp", Err: fmt.Errorf("connection refused")},
			isNetwork: true,
		},
		{
			name:      "wrapped OpError",
			err:       fmt.Errorf("connection failed: %w", &net.OpError{Op: "dial", Net: "tcp", Err: fmt.Errorf("connection refused")}),
			isNetwork: true,
		},
		{
			name:      "direct DNSError",
			err:       &net.DNSError{Err: "no such host", Name: "example.com"},
			isNetwork: true,
		},
		{
			name:      "wrapped DNSError",
			err:       fmt.Errorf("DNS lookup failed: %w", &net.DNSError{Err: "no such host", Name: "example.com"}),
			isNetwork: true,
		},
		{
			name:      "double wrapped network error",
			err:       fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", &net.OpError{Op: "dial", Net: "tcp", Err: fmt.Errorf("refused")})),
			isNetwork: true,
		},
		{
			name:      "non-network error",
			err:       fmt.Errorf("some application error"),
			isNetwork: false,
		},
		{
			name:      "wrapped non-network error",
			err:       fmt.Errorf("wrapped: %w", fmt.Errorf("application error")),
			isNetwork: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNetworkError(tt.err)
			if result != tt.isNetwork {
				t.Errorf("isNetworkError() = %v, want %v for error: %v", result, tt.isNetwork, tt.err)
			}
		})
	}
}
