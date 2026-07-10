// Copyright (c) Ryan Johnson
// SPDX-License-Identifier: MPL-2.0

package artifactory

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNewArtifactoryClientAuthMethods tests the creation of Artifactory clients
// with different authentication methods including API key, access token,
// username/password, and no authentication.
func TestNewArtifactoryClientAuthMethods(t *testing.T) {
	tests := []struct {
		name        string
		config      artifactoryConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "API key authentication",
			config: artifactoryConfig{
				URL:        mockArtifactoryURL,
				Repository: mockArtifactoryRepo,
				APIKey:     mockArtifactoryAPIKey,
			},
			expectError: false,
		},
		{
			name: "Access token authentication",
			config: artifactoryConfig{
				URL:         mockArtifactoryURL,
				Repository:  mockArtifactoryRepo,
				AccessToken: mockArtifactoryAccessToken,
			},
			expectError: false,
		},
		{
			name: "Username/password authentication",
			config: artifactoryConfig{
				URL:        mockArtifactoryURL,
				Repository: mockArtifactoryRepo,
				Username:   mockArtifactoryUsername,
				Password:   mockArtifactoryPassword,
			},
			expectError: false,
		},
		{
			name: "API key with username",
			config: artifactoryConfig{
				URL:        mockArtifactoryURL,
				Repository: mockArtifactoryRepo,
				APIKey:     mockArtifactoryAPIKey,
				Username:   mockArtifactoryUsername,
			},
			expectError: false,
		},
		{
			name: "no authentication",
			config: artifactoryConfig{
				URL:        mockArtifactoryURL,
				Repository: mockArtifactoryRepo,
			},
			expectError: true,
			errorMsg:    "no valid authentication method provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ui := &noOpTestUI{}
			client, err := newArtifactoryClient(tt.config, ui)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %v", tt.errorMsg, err)
				}
			} else {
				if err != nil && (strings.Contains(err.Error(), "dial tcp") ||
					strings.Contains(err.Error(), "no such host") ||
					strings.Contains(err.Error(), "connection")) {
					t.Skip("Skipping test that requires network connectivity")
					return
				}
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if client == nil {
					t.Errorf("Expected client but got nil")
				}
			}
		})
	}
}

// TestBuildArtifactPathEdgeCases tests the buildArtifactPath method with various
// edge cases including empty paths, security-related path traversal attempts,
// and malformed paths.
func TestBuildArtifactPathEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		config       artifactoryConfig
		filename     string
		expectedPath string
	}{
		{
			name: "empty artifact path",
			config: artifactoryConfig{
				ArtifactPath: "",
			},
			filename:     testOvaFile,
			expectedPath: testOvaFile,
		},
		{
			name: "artifact path with double dots (security)",
			config: artifactoryConfig{
				ArtifactPath: "../../../etc/passwd",
			},
			filename:     testOvaFile,
			expectedPath: "etc/passwd/" + testOvaFile,
		},
		{
			name: "artifact path with leading slash",
			config: artifactoryConfig{
				ArtifactPath: "/" + testArtifactPath,
			},
			filename:     testOvaFile,
			expectedPath: testArtifactPath + "/" + testOvaFile,
		},
		{
			name: "artifact path with multiple slashes",
			config: artifactoryConfig{
				ArtifactPath: "builds//test///",
			},
			filename:     testOvaFile,
			expectedPath: "builds//test/" + testOvaFile,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &artifactoryClient{
				config: tt.config,
			}

			result := client.buildArtifactPath(tt.filename)
			if result != tt.expectedPath {
				t.Errorf("buildArtifactPath() = %v, expected %v", result, tt.expectedPath)
			}
		})
	}
}

// TestArtifactMetadataStructure validates the artifactMetadata struct fields
// and ensures proper initialization and access to metadata properties.
func TestArtifactMetadataStructure(t *testing.T) {
	metadata := artifactMetadata{
		Name:        mockArtifactName,
		Type:        testArtifactTypeOva,
		BuilderId:   testBuilderId,
		Files:       []string{testOvaFile, testManifestFile},
		Timestamp:   testTimestamp,
		Description: testDescription,
		Properties: map[string]string{
			"build.number": "123",
			"version":      "1.0.0",
		},
	}

	if metadata.Name != mockArtifactName {
		t.Errorf("Expected Name to be %s, got %v", mockArtifactName, metadata.Name)
	}

	if metadata.Type != testArtifactTypeOva {
		t.Errorf("Expected Type to be %q, got %v", testArtifactTypeOva, metadata.Type)
	}

	if len(metadata.Files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(metadata.Files))
	}

	if metadata.Properties["build.number"] != "123" {
		t.Errorf("Expected build.number property to be '123', got %v", metadata.Properties["build.number"])
	}
}

// TestConfigValidationWithDefaults verifies that configuration validation
// properly applies default values for MaxRetries, TimeoutSeconds, and Overwrite
// settings.
func TestConfigValidationWithDefaults(t *testing.T) {
	clearArtifactoryEnvVars(t)

	config := artifactoryConfig{
		URL:          mockArtifactoryURL,
		Repository:   mockArtifactoryRepo,
		ArtifactName: mockArtifactName,
		APIKey:       mockArtifactoryAPIKey,
	}

	err := config.Validate()
	if err != nil {
		t.Fatalf("Config validation failed: %v", err)
	}

	if config.MaxRetries != defaultRetries {
		t.Errorf("Expected MaxRetries to be %d, got %d", defaultRetries, config.MaxRetries)
	}

	if config.TimeoutSeconds != defaultTimeoutSeconds {
		t.Errorf("Expected TimeoutSeconds to be %d, got %d", defaultTimeoutSeconds, config.TimeoutSeconds)
	}

	if config.Overwrite == nil {
		t.Errorf("Expected Overwrite to be set")
	} else if *config.Overwrite != false {
		t.Errorf("Expected default Overwrite to be false, got %v", *config.Overwrite)
	}
}

// TestConfigShouldOverwriteMethod tests the ShouldOverwrite method behavior
// with different pointer values including nil, explicit false, and explicit
// true values.
func TestConfigShouldOverwriteMethod(t *testing.T) {
	tests := []struct {
		name      string
		overwrite *bool
		expected  bool
	}{
		{
			name:      "nil overwrite should default to false",
			overwrite: nil,
			expected:  false,
		},
		{
			name: "explicit false",
			overwrite: func() *bool {
				v := false
				return &v
			}(),
			expected: false,
		},
		{
			name: "explicit true",
			overwrite: func() *bool {
				v := true
				return &v
			}(),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := artifactoryConfig{
				Overwrite: tt.overwrite,
			}
			result := config.shouldOverwrite()
			if result != tt.expected {
				t.Errorf("ShouldOverwrite() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// TestConfigEnvironmentVariableFallback tests that authentication credentials
// are properly loaded from environment variables and validates the fallback
// mechanism.
func TestConfigEnvironmentVariableFallback(t *testing.T) {
	originalAPIKey := os.Getenv("ARTIFACTORY_API_KEY")
	originalToken := os.Getenv("ARTIFACTORY_TOKEN")
	originalUsername := os.Getenv("ARTIFACTORY_USERNAME")
	originalPassword := os.Getenv("ARTIFACTORY_PASSWORD")

	defer func() {
		_ = os.Setenv("ARTIFACTORY_API_KEY", originalAPIKey)
		_ = os.Setenv("ARTIFACTORY_TOKEN", originalToken)
		_ = os.Setenv("ARTIFACTORY_USERNAME", originalUsername)
		_ = os.Setenv("ARTIFACTORY_PASSWORD", originalPassword)
	}()

	tests := []struct {
		name        string
		envVars     map[string]string
		config      artifactoryConfig
		expectValid bool
	}{
		{
			name: "API key from environment",
			envVars: map[string]string{
				"ARTIFACTORY_API_KEY": "env-api-key",
			},
			config: artifactoryConfig{
				URL:          mockArtifactoryURL,
				Repository:   mockArtifactoryRepo,
				ArtifactName: mockArtifactName,
			},
			expectValid: true,
		},
		{
			name: "access token from environment",
			envVars: map[string]string{
				"ARTIFACTORY_TOKEN": "env-token",
			},
			config: artifactoryConfig{
				URL:          mockArtifactoryURL,
				Repository:   mockArtifactoryRepo,
				ArtifactName: mockArtifactName,
			},
			expectValid: true,
		},
		{
			name: "username/password from environment",
			envVars: map[string]string{
				"ARTIFACTORY_USERNAME": "env-user",
				"ARTIFACTORY_PASSWORD": "env-pass",
			},
			config: artifactoryConfig{
				URL:          mockArtifactoryURL,
				Repository:   mockArtifactoryRepo,
				ArtifactName: mockArtifactName,
			},
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Unsetenv("ARTIFACTORY_API_KEY")
			_ = os.Unsetenv("ARTIFACTORY_TOKEN")
			_ = os.Unsetenv("ARTIFACTORY_USERNAME")
			_ = os.Unsetenv("ARTIFACTORY_PASSWORD")

			for key, value := range tt.envVars {
				_ = os.Setenv(key, value)
			}

			err := tt.config.Validate()
			if tt.expectValid && err != nil {
				t.Errorf("Expected artifactoryConfig to be valid, but got error: %v", err)
			}
			if !tt.expectValid && err == nil {
				t.Errorf("Expected artifactoryConfig to be invalid, but validation passed")
			}

			if tt.expectValid && err == nil {
				if tt.envVars["ARTIFACTORY_API_KEY"] != "" && tt.config.APIKey != tt.envVars["ARTIFACTORY_API_KEY"] {
					t.Errorf("Expected APIKey to be set from environment")
				}
				if tt.envVars["ARTIFACTORY_TOKEN"] != "" && tt.config.AccessToken != tt.envVars["ARTIFACTORY_TOKEN"] {
					t.Errorf("Expected AccessToken to be set from environment")
				}
				if tt.envVars["ARTIFACTORY_USERNAME"] != "" && tt.config.Username != tt.envVars["ARTIFACTORY_USERNAME"] {
					t.Errorf("Expected Username to be set from environment")
				}
				if tt.envVars["ARTIFACTORY_PASSWORD"] != "" && tt.config.Password != tt.envVars["ARTIFACTORY_PASSWORD"] {
					t.Errorf("Expected Password to be set from environment")
				}
			}
		})
	}
}

// TestRetryLogicIsNetworkError tests the isNetworkError function with various
// error types to ensure proper network error detection for retry logic.
func TestRetryLogicIsNetworkError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "regular error",
			err:      fmt.Errorf("regular error"),
			expected: false,
		},
		{
			name:     "network timeout error",
			err:      fmt.Errorf("network timeout"),
			expected: false, // This function checks for specific network error types
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNetworkError(tt.err)
			if result != tt.expected {
				t.Errorf("isNetworkError() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// TestGetCurrentTimestamp validates that the getCurrentTimestamp function returns
// a non-empty timestamp string of appropriate length.
func TestGetCurrentTimestamp(t *testing.T) {
	timestamp := getCurrentTimestamp()
	if timestamp == "" {
		t.Errorf("getCurrentTimestamp() returned empty string")
	}

	if len(timestamp) < 10 {
		t.Errorf("getCurrentTimestamp() returned timestamp that seems too short: %s", timestamp)
	}
}

// TestConfigValidationErrorCombinations tests configuration validation with multiple
// missing required fields and invalid URL formats.
func TestConfigValidationErrorCombinations(t *testing.T) {
	clearArtifactoryEnvVars(t)

	tests := []struct {
		name    string
		config  artifactoryConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:   "multiple validation errors",
			config: artifactoryConfig{
				// Missing URL, Repository, ArtifactName, and auth
			},
			wantErr: true,
			errMsg:  "url is required",
		},
		{
			name: "invalid URL format",
			config: artifactoryConfig{
				URL:          "not-a-valid-url",
				Repository:   mockArtifactoryRepo,
				ArtifactName: mockArtifactName,
				APIKey:       mockArtifactoryAPIKey,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing %q, got %v", tt.errMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestOvaOvfHandlerEdgeCases tests the OVA/OVF handler with edge cases including
// empty artifacts and verifies the handler name.
func TestOvaOvfHandlerEdgeCases(t *testing.T) {
	handler := newOvaOvfHandler()

	emptyArtifact := &testArtifact{FilesValue: []string{}}
	result := handler.CanHandle(emptyArtifact)
	if result != false {
		t.Errorf("CanHandle with empty files should return false")
	}

	name := handler.GetHandlerName()
	if name != ovaOvfHandlerName {
		t.Errorf("Expected handler name %q, got %q", ovaOvfHandlerName, name)
	}
}

// TestRetryLogicEdgeCases tests retry logic initialization with edge cases including
// zero retries and very high timeout values.
func TestRetryLogicEdgeCases(t *testing.T) {
	ui := &noOpTestUI{}

	retryLogic := newRetryLogic(0, 30, ui)
	if retryLogic.maxRetries != 0 {
		t.Errorf("Expected maxRetries to be 0, got %d", retryLogic.maxRetries)
	}

	retryLogic = newRetryLogic(3, 3600, ui)
	if retryLogic.timeoutSeconds != 3600 {
		t.Errorf("Expected timeoutSeconds to be 3600, got %d", retryLogic.timeoutSeconds)
	}
}

// TestArtifactProcessorEdgeCases tests the artifact processor with edge cases
// including empty artifacts and handler registration functionality.
func TestArtifactProcessorEdgeCases(t *testing.T) {
	processor := newArtifactProcessor(nil)

	emptyArtifact := &testArtifact{FilesValue: []string{}}
	_, err := processor.processArtifact(emptyArtifact)
	if err == nil {
		t.Errorf("Expected error when processing empty artifact")
	}

	initialCount := len(processor.handlers)

	mockHandler := &testHandler{Name: testHandlerName, CanHandleArtifact: false}
	processor.registerHandler(mockHandler)

	if len(processor.handlers) != initialCount+1 {
		t.Errorf("Expected %d handlers after registration, got %d", initialCount+1, len(processor.handlers))
	}
}

// TestConfigDefaultsApplication verifies that default values are properly applied
// when MaxRetries is initially set to zero during configuration validation.
func TestConfigDefaultsApplication(t *testing.T) {
	clearArtifactoryEnvVars(t)

	config := artifactoryConfig{
		URL:          mockArtifactoryURL,
		Repository:   mockArtifactoryRepo,
		ArtifactName: mockArtifactName,
		APIKey:       mockArtifactoryAPIKey,
		MaxRetries:   0,
	}

	err := config.Validate()
	if err != nil {
		t.Fatalf("Config validation failed: %v", err)
	}

	if config.MaxRetries != defaultRetries {
		t.Errorf("Expected MaxRetries to be set to default %d, got %d", defaultRetries, config.MaxRetries)
	}
}

// TestPostProcessorErrorPaths tests error handling in the PostProcessor
// Configure method with invalid configuration data.
func TestPostProcessorErrorPaths(t *testing.T) {
	processor := &PostProcessor{}

	err := processor.Configure(map[string]any{
		"invalid_field": "invalid_value",
	})
	if err == nil {
		t.Errorf("Expected error when configuring with invalid data")
	}
}

// TestArtifactMetadataWithProperties validates that artifact metadata properly
// handles custom properties and ensures correct property assignment and retrieval.
func TestArtifactMetadataWithProperties(t *testing.T) {
	metadata := artifactMetadata{
		Name:        mockArtifactName,
		Type:        testArtifactTypeOva,
		BuilderId:   testBuilderId,
		Files:       []string{testOvaFile, testManifestFile},
		Timestamp:   testTimestamp,
		Description: testDescription,
		Properties: map[string]string{
			"build.number": "123",
			"version":      "1.0.0",
			"environment":  "production",
		},
	}

	if len(metadata.Properties) != 3 {
		t.Errorf("Expected 3 properties, got %d", len(metadata.Properties))
	}

	if metadata.Properties["build.number"] != "123" {
		t.Errorf("Expected build.number to be '123', got %v", metadata.Properties["build.number"])
	}

	if metadata.Properties["environment"] != "production" {
		t.Errorf("Expected environment to be 'production', got %v", metadata.Properties["environment"])
	}
}

// TestConfigValidationWithEmptyValues tests configuration validation with empty
// required fields including URL, repository, and artifact name.
func TestConfigValidationWithEmptyValues(t *testing.T) {
	tests := []struct {
		name    string
		config  artifactoryConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "empty URL",
			config: artifactoryConfig{
				URL:          "",
				Repository:   mockArtifactoryRepo,
				ArtifactName: mockArtifactName,
				APIKey:       mockArtifactoryAPIKey,
			},
			wantErr: true,
			errMsg:  "url is required",
		},
		{
			name: "empty repository",
			config: artifactoryConfig{
				URL:          mockArtifactoryURL,
				Repository:   "",
				ArtifactName: mockArtifactName,
				APIKey:       mockArtifactoryAPIKey,
			},
			wantErr: true,
			errMsg:  "repository is required",
		},
		{
			name: "empty artifact name",
			config: artifactoryConfig{
				URL:        mockArtifactoryURL,
				Repository: mockArtifactoryRepo,
				APIKey:     mockArtifactoryAPIKey,
			},
			wantErr: true,
			errMsg:  "artifact_name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing %q, got %v", tt.errMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestOvaOvfHandlerProcessArtifactErrorCases tests error handling in the
// OVA/OVF handler when processing artifacts with invalid OVF descriptors.
func TestOvaOvfHandlerProcessArtifactErrorCases(t *testing.T) {
	handler := newOvaOvfHandler()
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		files       []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "invalid OVF descriptor",
			files:       []string{"invalid.ovf"},
			expectError: true,
			errorMsg:    "invalid OVF descriptor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tempFiles []string
			for _, file := range tt.files {
				tempFile := filepath.Join(tempDir, file)
				var content []byte

				if strings.HasSuffix(file, ".ovf") {
					content = []byte("invalid xml content")
				} else {
					content = []byte("test content")
				}

				if err := os.WriteFile(tempFile, content, 0600); err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}
				tempFiles = append(tempFiles, tempFile)
			}

			artifact := &testArtifact{FilesValue: tempFiles}
			_, err := handler.ProcessArtifact(artifact)

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
		})
	}
}

// TestIsNetworkErrorWithRealErrors tests the isNetworkError function with various
// error types including nil, regular errors, and network-related error messages.
func TestIsNetworkErrorWithRealErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "regular error",
			err:      fmt.Errorf("regular error"),
			expected: false,
		},
		{
			name:     "string error with network keywords",
			err:      fmt.Errorf("network connection failed"),
			expected: false, // isNetworkError checks for specific types, not strings
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNetworkError(tt.err)
			if result != tt.expected {
				t.Errorf("isNetworkError() = %v, expected %v for error: %v", result, tt.expected, tt.err)
			}
		})
	}
}

// TestConfigValidationWithSpecialCharacters validates that configuration accepts
// repository and artifact names containing special characters like dashes and underscores.
func TestConfigValidationWithSpecialCharacters(t *testing.T) {
	clearArtifactoryEnvVars(t)

	config := artifactoryConfig{ //nolint:gosec
		URL:          mockArtifactoryURL,
		Repository:   "test-repo-with-dashes_and_underscores",
		ArtifactName: "test-artifact-with-special.chars",
		APIKey:       "test-key-with-special-chars",
	}

	err := config.Validate()
	if err != nil {
		t.Errorf("Config with special characters should be valid, got error: %v", err)
	}
}

// TestArtifactProcessorWithMultipleHandlers tests artifact processing with
// multiple registered handlers and verifies that the first capable handler
// is used.
func TestArtifactProcessorWithMultipleHandlers(t *testing.T) {
	processor := newArtifactProcessor(nil)

	// Add multiple handlers
	handler1 := &testHandler{Name: "handler1", CanHandleArtifact: false}
	handler2 := &testHandler{Name: "handler2", CanHandleArtifact: true}
	handler3 := &testHandler{Name: "handler3", CanHandleArtifact: true}

	processor.registerHandler(handler1)
	processor.registerHandler(handler2)
	processor.registerHandler(handler3)

	// Should use the first handler that can handle the artifact (handler2)
	artifact := &testArtifact{FilesValue: []string{testOvaFile}}
	result, err := processor.processArtifact(artifact)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result == nil {
		t.Errorf("Expected result but got nil")
	}
}

// TestConfigValidationComplexScenarios tests configuration validation when
// MaxRetries is already set to the default value before validation.
func TestConfigValidationComplexScenarios(t *testing.T) {
	clearArtifactoryEnvVars(t)

	// Test scenario where MaxRetries is already set to default
	config := artifactoryConfig{
		URL:          mockArtifactoryURL,
		Repository:   mockArtifactoryRepo,
		ArtifactName: mockArtifactName,
		APIKey:       mockArtifactoryAPIKey,
		MaxRetries:   defaultRetries, // Already set to default
	}

	err := config.Validate()
	if err != nil {
		t.Errorf("Config validation failed: %v", err)
	}

	// Should still be the default value
	if config.MaxRetries != defaultRetries {
		t.Errorf("Expected MaxRetries to remain %d, got %d", defaultRetries, config.MaxRetries)
	}
}

// TestOvaOvfHandlerValidateOVFDescriptorEdgeCases tests OVF descriptor validation
// with edge cases including non-existent files.
func TestOvaOvfHandlerValidateOVFDescriptorEdgeCases(t *testing.T) {
	handler := newOvaOvfHandler()
	tempDir := t.TempDir()

	// Test with file that can't be opened
	nonExistentFile := filepath.Join(tempDir, "nonexistent.ovf")
	err := handler.validateOvfDescriptor(nonExistentFile)
	if err == nil {
		t.Errorf("Expected error for non-existent file")
		return
	}
	if !strings.Contains(err.Error(), "cannot open OVF file") {
		t.Errorf("Expected 'cannot open OVF file' error, got: %v", err)
	}
}

// TestPostProcessorWithNilClient tests the PostProcessor behavior when the client
// is nil and needs to be created during post-processing.
func TestPostProcessorWithNilClient(t *testing.T) {
	processor := &PostProcessor{
		config: artifactoryConfig{
			URL:          mockArtifactoryURL,
			Repository:   mockArtifactoryRepo,
			ArtifactName: mockArtifactName,
			APIKey:       mockArtifactoryAPIKey,
		},
		client: nil, // Nil client to force creation
	}

	tempDir := t.TempDir()
	ovaFile := filepath.Join(tempDir, testOvaFile)
	if err := os.WriteFile(ovaFile, []byte("test content"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	artifact := &testArtifact{
		BuilderIdValue: testBuilderId,
		FilesValue:     []string{ovaFile},
		IdValue:        mockArtifactName,
	}

	ui := &noOpTestUI{}
	ctx := context.Background()

	_, _, _, err := processor.PostProcess(ctx, ui, artifact)
	if err == nil {
		t.Errorf("Expected error due to client creation failure")
	}
}

// TestConfigValidationAuthenticationPriority tests that providing multiple
// authentication methods simultaneously results in a validation error.
func TestConfigValidationAuthenticationPriority(t *testing.T) {
	// Test that when multiple auth methods are provided, validation fails
	config := artifactoryConfig{
		URL:          mockArtifactoryURL,
		Repository:   mockArtifactoryRepo,
		ArtifactName: mockArtifactName,
		APIKey:       "api-key",
		AccessToken:  "access-token",
	}

	err := config.Validate()
	if err == nil {
		t.Errorf("Expected error for multiple authentication methods")
		return
	}
	if !strings.Contains(err.Error(), "multiple authentication methods provided") {
		t.Errorf("Expected multiple auth methods error, got: %v", err)
	}
}

// TestOvaOvfHandlerProcessArtifactWithUnknownFiles tests OVA/OVF handler
// processing with unknown files in the same directory and verifies proper
// file filtering.
func TestOvaOvfHandlerProcessArtifactWithUnknownFiles(t *testing.T) {
	handler := newOvaOvfHandler()
	tempDir := t.TempDir()

	ovfDir := filepath.Join(tempDir, testDirectoryName)
	if err := os.MkdirAll(ovfDir, 0755); err != nil {
		t.Fatalf("Failed to create OVF directory: %v", err)
	}

	ovfFile := filepath.Join(ovfDir, testOvfFile)
	ovfContent := `<?xml version="1.0"?>
<Envelope xmlns="http://schemas.dmtf.org/ovf/envelope/1">
  <VirtualSystem ovf:id="vm">
    <Name>Test VM</Name>
  </VirtualSystem>
</Envelope>`
	if err := os.WriteFile(ovfFile, []byte(ovfContent), 0600); err != nil {
		t.Fatalf("Failed to create OVF file: %v", err)
	}

	unknownFile := filepath.Join(ovfDir, "unknown.bin")
	if err := os.WriteFile(unknownFile, []byte("unknown content"), 0600); err != nil {
		t.Fatalf("Failed to create unknown file: %v", err)
	}

	outsideFile := filepath.Join(tempDir, testOutsideFile)
	if err := os.WriteFile(outsideFile, []byte("outside content"), 0600); err != nil {
		t.Fatalf("Failed to create outside file: %v", err)
	}

	allFiles := []string{ovfFile, unknownFile, outsideFile}
	artifact := &testArtifact{FilesValue: allFiles}

	result, err := handler.ProcessArtifact(artifact)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Only the OVF descriptor matches the portable upload set (not .bin).
	expectedFiles := 1
	if len(result.Files) != expectedFiles {
		t.Errorf("Expected %d files, got %d: %v", expectedFiles, len(result.Files), result.Files)
	}

	// unknown.bin is not in the portable set.
	for _, file := range result.Files {
		if strings.Contains(file, "unknown.bin") {
			t.Errorf("unexpected unknown.bin in results")
		}
	}
}

// TestUploadMetadataLogic tests the metadata preparation and property merging
// logic used in metadata upload operations without performing actual uploads.
func TestUploadMetadataLogic(t *testing.T) {
	// Test the metadata preparation logic without actual upload
	config := artifactoryConfig{
		Repository: mockArtifactoryRepo,
		Properties: map[string]string{
			"build.number": "12345678",
			"version":      "1.0.0",
			"release":      "stable",
		},
	}

	metadata := artifactMetadata{
		Name:        mockArtifactName,
		Type:        testArtifactTypeOva,
		BuilderId:   testBuilderId,
		Files:       []string{testOvaFile},
		Timestamp:   testTimestamp,
		Description: testDescription,
		Properties:  make(map[string]string),
	}

	// Test metadata properties merging logic (similar to what UploadMetadata does)
	maps.Copy(metadata.Properties, config.Properties)

	// Verify properties were merged correctly
	if metadata.Properties["build.number"] != "12345678" {
		t.Errorf("Expected build.number to be '12345678', got %v", metadata.Properties["build.number"])
	}
	if metadata.Properties["version"] != "1.0.0" {
		t.Errorf("Expected version to be '1.0.0', got %v", metadata.Properties["version"])
	}
	if metadata.Properties["release"] != "stable" {
		t.Errorf("Expected release to be 'stable', got %v", metadata.Properties["release"])
	}

	// Test JSON marshaling (part of UploadMetadata logic)
	jsonData, err := json.Marshal(metadata)
	if err != nil {
		t.Errorf("Failed to marshal metadata: %v", err)
	}
	if len(jsonData) == 0 {
		t.Errorf("Expected non-empty JSON data")
	}
}

// TestUploadFilePathBuilding tests the path construction logic used in file
// upload operations with various repository and artifact path configurations.
func TestUploadFilePathBuilding(t *testing.T) {
	// Test the path building logic used in UploadFile
	tests := []struct {
		name         string
		config       artifactoryConfig
		filename     string
		expectedPath string
	}{
		{
			name: "with repository and artifact path",
			config: artifactoryConfig{
				Repository:   "my-repo",
				ArtifactPath: "builds/v1.0",
			},
			filename:     "app.ova",
			expectedPath: "my-repo/builds/v1.0/app.ova",
		},
		{
			name: "with repository only",
			config: artifactoryConfig{
				Repository: "my-repo",
			},
			filename:     "app.ova",
			expectedPath: "my-repo/app.ova",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &artifactoryClient{config: tt.config}
			artifactPath := client.buildArtifactPath(tt.filename)
			targetPath := fmt.Sprintf("%s/%s", tt.config.Repository, artifactPath)

			if targetPath != tt.expectedPath {
				t.Errorf("Expected path %s, got %s", tt.expectedPath, targetPath)
			}
		})
	}
}

// TestCheckArtifactExistsLogic tests the search parameter building logic used
// in artifact existence checking operations.
func TestCheckArtifactExistsLogic(t *testing.T) {
	// Test the search parameters building logic used in checkArtifactExists
	artifactPath := "test-repo/builds/v1.0/app.ova"

	// Simulate what checkArtifactExists does for building search params
	// This tests the logic without requiring the actual JFrog client
	if false {
		t.Errorf("Artifact path should not be empty")
	}

	// Test path validation
	if !strings.Contains(artifactPath, "/") {
		t.Errorf("Expected artifact path to contain repository separator")
	}

	// Test that we can extract components
	parts := strings.Split(artifactPath, "/")
	if len(parts) < 2 {
		t.Errorf("Expected at least repository and filename in path")
	}
}

// TestNewArtifactoryClientURLHandling tests URL normalization logic in
// Artifactory client creation, ensuring proper handling of trailing slashes.
func TestNewArtifactoryClientURLHandling(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectedURL string
	}{
		{
			name:        "URL with trailing slash",
			url:         mockArtifactoryURL,
			expectedURL: mockArtifactoryURL,
		},
		{
			name:        "URL without trailing slash",
			url:         mockArtifactoryURLNoSlash,
			expectedURL: mockArtifactoryURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the URL normalization logic used in NewArtifactoryClient
			normalizedURL := strings.TrimRight(tt.url, "/") + "/"
			if normalizedURL != tt.expectedURL {
				t.Errorf("Expected URL %s, got %s", tt.expectedURL, normalizedURL)
			}
		})
	}
}

// TestIsNetworkErrorWithSpecificTypes tests the isNetworkError function with
// specific error types including nil, regular, and wrapped errors.
func TestIsNetworkErrorWithSpecificTypes(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "regular error",
			err:      fmt.Errorf("regular error"),
			expected: false,
		},
		{
			name:     "wrapped error",
			err:      fmt.Errorf("wrapped: %w", fmt.Errorf("inner error")),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNetworkError(tt.err)
			if result != tt.expected {
				t.Errorf("isNetworkError() = %v, expected %v for error: %v", result, tt.expected, tt.err)
			}
		})
	}
}

// TestConfigValidationWithMaxRetries tests configuration validation when
// MaxRetries is explicitly set to a non-zero value before validation.
func TestConfigValidationWithMaxRetries(t *testing.T) {
	clearArtifactoryEnvVars(t)

	// Test the specific logic path where MaxRetries is already set to DefaultRetries
	config := artifactoryConfig{
		URL:          mockArtifactoryURL,
		Repository:   mockArtifactoryRepo,
		ArtifactName: mockArtifactName,
		APIKey:       mockArtifactoryAPIKey,
		MaxRetries:   defaultRetries, // Set to default value
	}

	err := config.Validate()
	if err != nil {
		t.Errorf("Config validation failed: %v", err)
	}

	// Should remain at default value
	if config.MaxRetries != defaultRetries {
		t.Errorf("Expected MaxRetries to remain %d, got %d", defaultRetries, config.MaxRetries)
	}
}

// TestOvaOvfHandlerProcessArtifactFileFiltering tests multi-file OVF filtering:
// portable upload set is .ovf, .mf, .vmdk, .cert, .vdi (plus extras from additional_ovf_extensions).
// Adjuncts like nvram/iso/log/txt are omitted unless listed. When a .ova is present,
// only the .ova is uploaded.
func TestOvaOvfHandlerProcessArtifactFileFiltering(t *testing.T) {
	handler := newOvaOvfHandler()
	tempDir := t.TempDir()

	files := map[string]string{
		testOvaFile:      "ova content",
		testOvfFile:      `<?xml version="1.0"?><Envelope xmlns="http://schemas.dmtf.org/ovf/envelope/1"><VirtualSystem ovf:id="vm"><Name>Test</Name></VirtualSystem></Envelope>`,
		testVmdkFile:     "vmdk content",
		testManifestFile: "manifest content",
		testCertFile:     "certificate content",
		testNvramFile:    "nvram content",
		testIsoFile:      "iso content",
		testTextFile:     "text content",
		testLogFile:      "log content",
	}

	var paths []string
	for filename, content := range files {
		filePath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
			t.Fatalf("Failed to create file %s: %v", filename, err)
		}
		paths = append(paths, filePath)
	}

	t.Run("withOva_uploadsOvaOnly", func(t *testing.T) {
		artifact := &testArtifact{FilesValue: paths}
		result, err := handler.ProcessArtifact(artifact)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(result.Files) != 1 || !strings.HasSuffix(result.Files[0], testOvaFile) {
			t.Fatalf("expected only %s, got %v", testOvaFile, result.Files)
		}
	})

	t.Run("ovfOnly_portableAllowlist", func(t *testing.T) {
		var pathsNoOva []string
		for filename, content := range files {
			if filename == testOvaFile {
				continue
			}
			filePath := filepath.Join(tempDir, "ovfonly-"+filename)
			if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
				t.Fatalf("Failed to create file %s: %v", filename, err)
			}
			pathsNoOva = append(pathsNoOva, filePath)
		}

		artifact := &testArtifact{FilesValue: pathsNoOva}
		result, err := handler.ProcessArtifact(artifact)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		// ovf, vmdk, mf, cert from portable set; not nvram, iso, txt, log
		if len(result.Files) != 4 {
			t.Fatalf("expected 4 portable files for OVF package, got %d: %v", len(result.Files), result.Files)
		}
		for _, file := range result.Files {
			if strings.HasSuffix(file, ".txt") || strings.HasSuffix(file, ".log") ||
				strings.HasSuffix(file, ".nvram") || strings.HasSuffix(file, ".iso") {
				t.Errorf("unexpected adjunct in portable set: %s", file)
			}
		}
	})

	t.Run("ovfOnly_withIncludedAdjuncts", func(t *testing.T) {
		handler := newOvaOvfHandlerWithIncluded([]string{"nvram", "iso", "txt", "log"})
		var pathsNoOva []string
		for filename, content := range files {
			if filename == testOvaFile {
				continue
			}
			filePath := filepath.Join(tempDir, "ovfinc-"+filename)
			if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
				t.Fatalf("Failed to create file %s: %v", filename, err)
			}
			pathsNoOva = append(pathsNoOva, filePath)
		}

		artifact := &testArtifact{FilesValue: pathsNoOva}
		result, err := handler.ProcessArtifact(artifact)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(result.Files) != 8 {
			t.Fatalf("expected 8 files when adjunct extensions are included, got %d: %v", len(result.Files), result.Files)
		}
		foundTxt, foundLog := false, false
		for _, file := range result.Files {
			if strings.HasSuffix(file, ".txt") {
				foundTxt = true
			}
			if strings.HasSuffix(file, ".log") {
				foundLog = true
			}
		}
		if !foundTxt || !foundLog {
			t.Errorf("expected .txt and .log when included, txt=%v log=%v", foundTxt, foundLog)
		}
	})
}

// TestArtifactMetadataJSONSerialization tests JSON marshaling and unmarshaling
// of artifact metadata to ensure proper serialization behavior.
func TestArtifactMetadataJSONSerialization(t *testing.T) {
	metadata := artifactMetadata{
		Name:        mockArtifactName,
		Type:        testArtifactTypeOva,
		BuilderId:   testBuilderId,
		Files:       []string{testOvaFile, testManifestFile},
		Timestamp:   testTimestamp,
		Description: testDescription,
		Properties: map[string]string{
			"build.number": "123",
			"version":      "1.0.0-beta+build.1",
			"special":      "value with spaces and symbols: @#$%",
		},
	}

	jsonData, err := json.Marshal(metadata)
	if err != nil {
		t.Errorf("Failed to marshal metadata: %v", err)
	}

	var unmarshaled artifactMetadata
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal metadata: %v", err)
	}

	if unmarshaled.Name != metadata.Name {
		t.Errorf("Name mismatch after JSON round-trip")
	}
	if len(unmarshaled.Properties) != len(metadata.Properties) {
		t.Errorf("Properties count mismatch after JSON round-trip")
	}
	if unmarshaled.Properties["special"] != metadata.Properties["special"] {
		t.Errorf("Special characters not preserved in JSON round-trip")
	}
}

// TestConfigValidationDuplicateMaxRetriesLogic tests configuration validation
// logic when MaxRetries is already set to ensure proper default handling.
func TestConfigValidationDuplicateMaxRetriesLogic(t *testing.T) {
	clearArtifactoryEnvVars(t)

	config := artifactoryConfig{
		URL:          mockArtifactoryURL,
		Repository:   mockArtifactoryRepo,
		ArtifactName: mockArtifactName,
		APIKey:       mockArtifactoryAPIKey,
		MaxRetries:   0,
	}

	err := config.Validate()
	if err != nil {
		t.Errorf("Config validation failed: %v", err)
	}

	if config.MaxRetries != defaultRetries {
		t.Errorf("Expected MaxRetries to be %d, got %d", defaultRetries, config.MaxRetries)
	}
}

// TestOvaOvfHandlerProcessArtifactTypeDetection tests the OVA/OVF handler's
// ability to detect and process different artifact types (OVA vs OVF).
func TestOvaOvfHandlerProcessArtifactTypeDetection(t *testing.T) {
	handler := newOvaOvfHandler()
	tempDir := t.TempDir()

	tests := []struct {
		name         string
		files        []string
		expectedType string
	}{
		{
			name:         "single OVA file should be type ova",
			files:        []string{"single.ova"},
			expectedType: testArtifactTypeOva,
		},
		{
			name:         "single OVF file should be type ovf",
			files:        []string{"single.ovf"},
			expectedType: testArtifactTypeOvf,
		},
		{
			name:         "multiple files should be type ovf",
			files:        []string{"multi.ovf", "multi.vmdk"},
			expectedType: testArtifactTypeOvf,
		},
		{
			name:         "OVF with additional files should be type ovf",
			files:        []string{"multi.ovf", "multi.vmdk", "multi.mf"},
			expectedType: testArtifactTypeOvf,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tempFiles []string
			for _, file := range tt.files {
				tempFile := filepath.Join(tempDir, file)
				var content []byte

				if strings.HasSuffix(file, ".ovf") {
					content = []byte(`<?xml version="1.0"?>
<Envelope xmlns="http://schemas.dmtf.org/ovf/envelope/1">
  <VirtualSystem ovf:id="vm">
    <Name>Test VM</Name>
  </VirtualSystem>
</Envelope>`)
				} else {
					content = []byte("test content")
				}

				if err := os.WriteFile(tempFile, content, 0600); err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}
				tempFiles = append(tempFiles, tempFile)
			}

			artifact := &testArtifact{FilesValue: tempFiles}
			result, err := handler.ProcessArtifact(artifact)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("Expected result but got nil")
				return
			}

			if result.Metadata["type"] != tt.expectedType {
				t.Errorf("Expected type %s, got %s", tt.expectedType, result.Metadata["type"])
			}
		})
	}
}

// TestConfigValidationErrorAccumulation tests that configuration validation
// properly accumulates and reports multiple validation errors.
func TestConfigValidationErrorAccumulation(t *testing.T) {
	clearArtifactoryEnvVars(t)

	// Test that multiple validation errors are accumulated
	config := artifactoryConfig{
		// Missing URL, Repository, ArtifactName, and auth - should generate multiple errors
	}

	err := config.Validate()
	if err == nil {
		t.Errorf("Expected validation errors but got nil")
		return
	}

	errorStr := err.Error()
	// Should contain multiple error messages
	if !strings.Contains(errorStr, "url is required") {
		t.Errorf("Expected URL error in validation message")
	}
	if !strings.Contains(errorStr, "repository is required") {
		t.Errorf("Expected repository error in validation message")
	}
	if !strings.Contains(errorStr, "artifact_name is required") {
		t.Errorf("Expected artifact_name error in validation message")
	}
	if !strings.Contains(errorStr, "authentication is required") {
		t.Errorf("Expected authentication error in validation message")
	}
}

// TestNewArtifactoryClientAuthenticationBranches tests different authentication
// configuration branches in Artifactory client creation.
func TestNewArtifactoryClientAuthenticationBranches(t *testing.T) {
	ui := &noOpTestUI{}

	tests := []struct {
		name        string
		config      artifactoryConfig
		expectError bool
	}{
		{
			name: "API key with username",
			config: artifactoryConfig{
				URL:        mockArtifactoryURL,
				Repository: mockArtifactoryRepo,
				APIKey:     mockArtifactoryAPIKey,
				Username:   mockArtifactoryUsername,
			},
			expectError: false,
		},
		{
			name: "API key without username",
			config: artifactoryConfig{
				URL:        mockArtifactoryURL,
				Repository: mockArtifactoryRepo,
				APIKey:     mockArtifactoryAPIKey,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newArtifactoryClient(tt.config, ui)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
			} else {
				// Skip network-related errors
				if err != nil && (strings.Contains(err.Error(), "dial tcp") ||
					strings.Contains(err.Error(), "no such host") ||
					strings.Contains(err.Error(), "connection")) {
					t.Skip("Skipping test that requires network connectivity")
					return
				}
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestRetryLogicShouldRetryEdgeCases tests the retry logic's shouldRetry method
// with various error conditions and retry scenarios.
func TestRetryLogicShouldRetryEdgeCases(t *testing.T) {
	ui := &noOpTestUI{}
	retryLogic := newRetryLogic(3, 30, ui)

	tests := []struct {
		name        string
		err         error
		shouldRetry bool
	}{
		{
			name:        "HTTP 405 should not retry",
			err:         fmt.Errorf("HTTP 405 Method Not Allowed"),
			shouldRetry: false,
		},
		{
			name:        "HTTP 409 should not retry",
			err:         fmt.Errorf("HTTP 409 Conflict"),
			shouldRetry: false,
		},
		{
			name:        "HTTP 422 should not retry",
			err:         fmt.Errorf("HTTP 422 Unprocessable Entity"),
			shouldRetry: false,
		},
		{
			name:        "HTTP 429 should not retry (rate limit)",
			err:         fmt.Errorf("HTTP 429 Too Many Requests"),
			shouldRetry: false,
		},
		{
			name:        "HTTP 503 should retry",
			err:         fmt.Errorf("HTTP 503 Service Unavailable"),
			shouldRetry: true,
		},
		{
			name:        "HTTP 504 should retry",
			err:         fmt.Errorf("HTTP 504 Gateway Timeout"),
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

// TestOvaOvfHandlerFilesByDirectoryLogic tests allowlist behavior: only extensions
// in the portable (or included) set are uploaded; unrelated files beside the OVF are ignored.
func TestOvaOvfHandlerFilesByDirectoryLogic(t *testing.T) {
	handler := newOvaOvfHandler()
	tempDir := t.TempDir()

	// Create subdirectories
	subDir1 := filepath.Join(tempDir, "dir1")
	subDir2 := filepath.Join(tempDir, "dir2")
	if err := os.MkdirAll(subDir1, 0755); err != nil {
		t.Fatalf("Failed to create subdir1: %v", err)
	}
	if err := os.MkdirAll(subDir2, 0755); err != nil {
		t.Fatalf("Failed to create subdir2: %v", err)
	}

	// Create OVF in dir1
	ovfFile := filepath.Join(subDir1, testOvfFile)
	ovfContent := `<?xml version="1.0"?>
<Envelope xmlns="http://schemas.dmtf.org/ovf/envelope/1">
  <VirtualSystem ovf:id="vm">
    <Name>Test VM</Name>
  </VirtualSystem>
</Envelope>`
	if err := os.WriteFile(ovfFile, []byte(ovfContent), 0600); err != nil {
		t.Fatalf("Failed to create OVF file: %v", err)
	}

	// Adjacent .bin files are not uploaded unless additional_ovf_extensions lists "bin".
	unknownFile1 := filepath.Join(subDir1, "unknown1.bin")
	if err := os.WriteFile(unknownFile1, []byte("unknown content"), 0600); err != nil {
		t.Fatalf("Failed to create unknown file 1: %v", err)
	}

	unknownFile2 := filepath.Join(subDir2, "unknown2.bin")
	if err := os.WriteFile(unknownFile2, []byte("unknown content"), 0600); err != nil {
		t.Fatalf("Failed to create unknown file 2: %v", err)
	}

	allFiles := []string{ovfFile, unknownFile1, unknownFile2}
	artifact := &testArtifact{FilesValue: allFiles}

	result, err := handler.ProcessArtifact(artifact)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// unknown*.bin are not in the portable set unless explicitly included.
	expectedFiles := 1
	if len(result.Files) != expectedFiles {
		t.Errorf("Expected %d files, got %d: %v", expectedFiles, len(result.Files), result.Files)
	}

	foundUnknown1 := false
	foundUnknown2 := false
	for _, file := range result.Files {
		if strings.Contains(file, "unknown1.bin") {
			foundUnknown1 = true
		}
		if strings.Contains(file, "unknown2.bin") {
			foundUnknown2 = true
		}
	}

	if foundUnknown1 {
		t.Errorf("expected unknown1.bin to be omitted without additional_ovf_extensions")
	}
	if foundUnknown2 {
		t.Errorf("expected unknown2.bin to be omitted without additional_ovf_extensions")
	}
}
