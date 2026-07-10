// Copyright (c) Ryan Johnson
// SPDX-License-Identifier: MPL-2.0

package artifactory

import (
	"os"
	"strings"
	"testing"
)

// TestConfigValidate tests the artifactoryConfig.Validate method with various configuration
// scenarios including valid configurations, missing required fields, and
// different authentication methods.
func TestConfigValidate(t *testing.T) {
	clearArtifactoryEnvVars(t)

	tests := []struct {
		name    string
		config  artifactoryConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid artifactoryConfig with all required fields",
			config: artifactoryConfig{
				URL:          mockArtifactoryURL,
				Repository:   mockArtifactoryRepo,
				ArtifactName: mockArtifactName,
				APIKey:       mockArtifactoryAPIKey,
			},
			wantErr: false,
		},
		{
			name: "missing URL",
			config: artifactoryConfig{
				Repository:   mockArtifactoryRepo,
				ArtifactName: mockArtifactName,
				APIKey:       mockArtifactoryAPIKey,
			},
			wantErr: true,
			errMsg:  "url is required",
		},
		{
			name: "missing repository",
			config: artifactoryConfig{
				URL:          mockArtifactoryURL,
				ArtifactName: mockArtifactName,
				APIKey:       mockArtifactoryAPIKey,
			},
			wantErr: true,
			errMsg:  "repository is required",
		},
		{
			name: "missing artifact name",
			config: artifactoryConfig{
				URL:        mockArtifactoryURL,
				Repository: mockArtifactoryRepo,
				APIKey:     mockArtifactoryAPIKey,
			},
			wantErr: true,
			errMsg:  "artifact_name is required",
		},
		{
			name: "missing authentication",
			config: artifactoryConfig{
				URL:          mockArtifactoryURL,
				Repository:   mockArtifactoryRepo,
				ArtifactName: mockArtifactName,
			},
			wantErr: true,
			errMsg:  "authentication is required",
		},
		{
			name: "valid artifactoryConfig with username/password auth",
			config: artifactoryConfig{
				URL:          mockArtifactoryURL,
				Repository:   mockArtifactoryRepo,
				ArtifactName: mockArtifactName,
				Username:     mockArtifactoryUsername,
				Password:     mockArtifactoryPassword,
			},
			wantErr: false,
		},
		{
			name: "valid artifactoryConfig with access token",
			config: artifactoryConfig{
				URL:          mockArtifactoryURL,
				Repository:   mockArtifactoryRepo,
				ArtifactName: mockArtifactName,
				AccessToken:  "access-token",
			},
			wantErr: false,
		},
		{
			name: "username without password",
			config: artifactoryConfig{
				URL:          mockArtifactoryURL,
				Repository:   mockArtifactoryRepo,
				ArtifactName: mockArtifactName,
				Username:     mockArtifactoryUsername,
			},
			wantErr: true,
			errMsg:  "authentication is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Config.Validate() expected error but got nil")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Config.Validate() error = %v, expected to contain %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Config.Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestConfigDefaults verifies that default values are properly applied to
// configuration fields during validation, specifically testing MaxRetries and
// TimeoutSeconds defaults.
func TestConfigDefaults(t *testing.T) {
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

	if config.MaxRetries != 3 {
		t.Errorf("Expected default retries to be 3, got %d", config.MaxRetries)
	}

	if config.TimeoutSeconds != 30 {
		t.Errorf("Expected default timeout to be 30, got %d", config.TimeoutSeconds)
	}
}

// TestConfigAuthenticationMutuallyExclusive ensures that providing multiple
// authentication methods (API key, access token, username/password)
// simultaneously results in a validation error.
func TestConfigAuthenticationMutuallyExclusive(t *testing.T) {
	// Test that multiple authentication methods are rejected.
	config := artifactoryConfig{
		URL:          mockArtifactoryURL,
		Repository:   mockArtifactoryRepo,
		ArtifactName: mockArtifactName,
		APIKey:       mockArtifactoryAPIKey,
		AccessToken:  mockArtifactoryAccessToken,
		Username:     mockArtifactoryUsername,
		Password:     mockArtifactoryUsername,
	}

	err := config.Validate()
	if err == nil {
		t.Errorf("Config.Validate() expected error for multiple authentication methods, but got nil")
	}

	expectedError := "multiple authentication methods provided"
	if err != nil && !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Config.Validate() expected error containing %q, got %v", expectedError, err)
	}
}

// TestConfigShouldOverwrite tests the ShouldOverwrite method behavior with
// different overwrite pointer values including nil, explicit false, and
// explicit true.
func TestConfigShouldOverwrite(t *testing.T) {
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

// TestConfigEnvironmentVariables validates that authentication credentials can
// be provided using environment variables.
func TestConfigEnvironmentVariables(t *testing.T) {
	// Save environment variables.
	originalAPIKey := os.Getenv("ARTIFACTORY_API_KEY")
	originalToken := os.Getenv("ARTIFACTORY_TOKEN")
	originalUsername := os.Getenv("ARTIFACTORY_USERNAME")
	originalPassword := os.Getenv("ARTIFACTORY_PASSWORD")

	// Clean up.
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
			// Clear environment variables.
			_ = os.Unsetenv("ARTIFACTORY_API_KEY")
			_ = os.Unsetenv("ARTIFACTORY_TOKEN")
			_ = os.Unsetenv("ARTIFACTORY_USERNAME")
			_ = os.Unsetenv("ARTIFACTORY_PASSWORD")

			// Set test environment variables.
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
		})
	}
}

// TestConfigValidationEdgeCases tests edge cases in configuration validation
// including incomplete username/password pairs and verifies that default values
// are properly applied.
func TestConfigValidationEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		config  artifactoryConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "username without password should fail",
			config: artifactoryConfig{
				URL:          mockArtifactoryURL,
				Repository:   mockArtifactoryRepo,
				ArtifactName: mockArtifactName,
				Username:     mockArtifactoryUsername,
			},
			wantErr: true,
			errMsg:  "username and password must be provided together",
		},
		{
			name: "password without username should fail",
			config: artifactoryConfig{
				URL:          mockArtifactoryURL,
				Repository:   mockArtifactoryRepo,
				ArtifactName: mockArtifactName,
				Password:     mockArtifactoryPassword,
			},
			wantErr: true,
			errMsg:  "username and password must be provided together",
		},
		{
			name: "defaults should be applied",
			config: artifactoryConfig{
				URL:          mockArtifactoryURL,
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
				// Check that defaults were applied.
				if tt.config.MaxRetries != defaultRetries {
					t.Errorf("Expected MaxRetries to be %d, got %d", defaultRetries, tt.config.MaxRetries)
				}
				if tt.config.TimeoutSeconds != defaultTimeoutSeconds {
					t.Errorf("Expected TimeoutSeconds to be %d, got %d", defaultTimeoutSeconds, tt.config.TimeoutSeconds)
				}
				if tt.config.Overwrite == nil {
					t.Errorf("Expected Overwrite to be set to default")
				}
			}
		})
	}
}
