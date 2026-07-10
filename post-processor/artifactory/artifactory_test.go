// Copyright (c) Ryan Johnson
// SPDX-License-Identifier: MPL-2.0

package artifactory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jfrog/jfrog-client-go/artifactory/services"
)

// TestBuildArtifactPath tests artifact path building with various configurations.
func TestBuildArtifactPath(t *testing.T) {
	tests := []struct {
		name         string
		config       artifactoryConfig
		filename     string
		expectedPath string
	}{
		{
			name: "no artifact path specified",
			config: artifactoryConfig{
				ArtifactPath: "",
			},
			filename:     testOvaFile,
			expectedPath: testOvaFile,
		},
		{
			name: "artifact path specified",
			config: artifactoryConfig{
				ArtifactPath: "builds/2025-01-01",
			},
			filename:     testOvaFile,
			expectedPath: "builds/2025-01-01/" + testOvaFile,
		},
		{
			name: "artifact path with trailing slash",
			config: artifactoryConfig{
				ArtifactPath: "builds/2025-01-01/",
			},
			filename:     testOvaFile,
			expectedPath: "builds/2025-01-01/" + testOvaFile,
		},
		{
			name: "nested artifact path",
			config: artifactoryConfig{
				ArtifactPath: testNestedPath,
			},
			filename:     "app.ovf",
			expectedPath: testNestedPath + "/app.ovf",
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

// TestNewArtifactoryClientValidation tests client creation with invalid
// configurations.
func TestNewArtifactoryClientValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  artifactoryConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "invalid artifactoryConfig - no auth",
			config: artifactoryConfig{
				URL:          mockArtifactoryURL,
				Repository:   mockArtifactoryRepo,
				ArtifactName: mockArtifactName,
			},
			wantErr: true,
			errMsg:  "no valid authentication method provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUI := &noOpTestUI{}
			_, err := newArtifactoryClient(tt.config, mockUI)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewArtifactoryClient() expected error but got nil")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("NewArtifactoryClient() error = %v, expected to contain %v", err, tt.errMsg)
				}
				return
			}

			if err != nil && (strings.Contains(err.Error(), "dial tcp") ||
				strings.Contains(err.Error(), "no such host") ||
				strings.Contains(err.Error(), "connection")) {
				t.Skip("Skipping test that requires network connectivity")
				return
			}

			if err != nil {
				t.Errorf("NewArtifactoryClient() unexpected error = %v", err)
			}
		})
	}
}

// TestArtifactMetadata tests artifact metadata structure and field validation.
func TestArtifactMetadata(t *testing.T) {
	metadata := artifactMetadata{
		Name:        mockArtifactName,
		Type:        testArtifactTypeOva,
		BuilderId:   testBuilderId,
		Files:       []string{testOvaFile},
		Timestamp:   testTimestamp,
		Description: testDescription,
	}

	if metadata.Name != mockArtifactName {
		t.Errorf("Expected Name to be %s, got %v", mockArtifactName, metadata.Name)
	}

	if metadata.Type != testArtifactTypeOva {
		t.Errorf("Expected Type to be %q, got %v", testArtifactTypeOva, metadata.Type)
	}

	if len(metadata.Files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(metadata.Files))
	}

	if metadata.Files[0] != testOvaFile {
		t.Errorf("Expected file to be %q, got %v", testOvaFile, metadata.Files[0])
	}
}

// TestUploadFileValidation tests file upload validation and path building logic.
func TestUploadFileValidation(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, testOvaFile)
	err := os.WriteFile(tempFile, []byte("test content"), 0600)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	config := artifactoryConfig{
		URL:          mockArtifactoryURL,
		Repository:   mockArtifactoryRepo,
		ArtifactName: mockArtifactName,
		APIKey:       mockArtifactoryAPIKey,
		ArtifactPath: testArtifactPath,
	}

	client := &artifactoryClient{
		config: config,
	}

	path := client.buildArtifactPath(testOvaFile)
	expectedPath := testArtifactPath + "/" + testOvaFile
	if path != expectedPath {
		t.Errorf("buildArtifactPath() = %v, expected %v", path, expectedPath)
	}

	nonExistentFile := "/non/existent/file.ova"
	if _, err := os.Stat(nonExistentFile); err == nil {
		t.Errorf("Test file %s should not exist", nonExistentFile)
	}
}

// TestResolvedPropertiesDropsEmptyValues verifies that properties entries
// with an empty value (e.g. an unset "{{ env `VAR` }}") are omitted rather
// than sent to Artifactory as "key=", which its upload API rejects with
// "Invalid property format".
func TestResolvedPropertiesDropsEmptyValues(t *testing.T) {
	logOutput := captureLogOutput(t)
	props := exampleProperties()
	props["build.number"] = ""
	client := &artifactoryClient{
		config: artifactoryConfig{
			Properties: props,
		},
	}

	resolved := client.resolvedProperties()

	if len(resolved) != len(props)-1 {
		t.Fatalf("resolvedProperties() = %v, expected %d entries", resolved, len(props)-1)
	}
	for key, value := range exampleProperties() {
		if key == "build.number" {
			continue
		}
		if resolved[key] != value {
			t.Errorf("resolvedProperties()[%q] = %q, expected %q", key, resolved[key], value)
		}
	}
	if _, present := resolved["build.number"]; present {
		t.Errorf("resolvedProperties() should have dropped empty %q", "build.number")
	}

	if !strings.Contains(logOutput.String(), `Skipping property "build.number"`) {
		t.Errorf("expected a [WARN] log about skipped %q, got log output: %v", "build.number", logOutput.String())
	}

	// Calling it again should not duplicate the warning for the same key.
	logOutput.Reset()
	client.resolvedProperties()
	count := strings.Count(logOutput.String(), `Skipping property "build.number"`)
	if count != 0 {
		t.Errorf("expected the empty-property warning not to repeat, got %d additional occurrences", count)
	}
}

// TestSetPathPropertiesDisabled verifies that SetPathProperties is a no-op
// that never touches the services manager when artifact_path_properties is
// disabled (the default).
func TestSetPathPropertiesDisabled(t *testing.T) {
	client := &artifactoryClient{
		config: artifactoryConfig{
			ArtifactPath: testArtifactPath,
			Properties:   exampleProperties(),
		},
		// No servicesManager configured: a call to SetProps would panic,
		// which proves the disabled path never reaches it.
	}

	if err := client.SetPathProperties(context.Background()); err != nil {
		t.Fatalf("SetPathProperties() unexpected error: %v", err)
	}
}

// TestSetPathPropertiesNoArtifactPath verifies that SetPathProperties skips
// the folder update and warns once when artifact_path is not configured.
func TestSetPathPropertiesNoArtifactPath(t *testing.T) {
	logOutput := captureLogOutput(t)
	client := &artifactoryClient{
		config: artifactoryConfig{
			ArtifactPathProperties: true,
			Properties:             exampleProperties(),
		},
	}

	if err := client.SetPathProperties(context.Background()); err != nil {
		t.Fatalf("SetPathProperties() unexpected error: %v", err)
	}

	if !strings.Contains(logOutput.String(), "artifact_path is not set") {
		t.Errorf("expected a [WARN] log about missing artifact_path, got log output: %v", logOutput.String())
	}
}

// TestSetPathPropertiesNoProperties verifies that SetPathProperties skips
// the folder update and warns once when there are no resolved properties.
func TestSetPathPropertiesNoProperties(t *testing.T) {
	logOutput := captureLogOutput(t)
	client := &artifactoryClient{
		config: artifactoryConfig{
			ArtifactPathProperties: true,
			ArtifactPath:           testArtifactPath,
		},
	}

	if err := client.SetPathProperties(context.Background()); err != nil {
		t.Fatalf("SetPathProperties() unexpected error: %v", err)
	}

	if !strings.Contains(logOutput.String(), "no properties are configured") {
		t.Errorf("expected a [WARN] log about missing properties, got log output: %v", logOutput.String())
	}
}

// TestSetPathPropertiesSuccess verifies that SetPathProperties calls SetProps
// with a folder item matching artifact_path and the resolved properties.
func TestSetPathPropertiesSuccess(t *testing.T) {
	mockUI := &capturingTestUI{}
	var item struct {
		Repo string `json:"repo"`
		Path string `json:"path"`
		Name string `json:"name"`
		Type string `json:"type"`
	}
	mockManager := &mockServicesManager{
		SetPropsFunc: func(params services.PropsParams) (int, error) {
			// Consume the reader here (as the real jfrog-client-go
			// implementation does) since SetPathProperties closes and
			// removes the reader's backing file once the call returns.
			if err := params.Reader.NextRecord(&item); err != nil {
				return 0, err
			}
			return 1, nil
		},
	}
	client := &artifactoryClient{
		ui:              mockUI,
		servicesManager: mockManager,
		retryLogic:      newRetryLogic(3, 30, mockUI),
		config: artifactoryConfig{
			ArtifactPathProperties: true,
			Repository:             mockArtifactoryRepo,
			ArtifactPath:           testArtifactPath,
			Properties:             exampleProperties(),
		},
	}

	if err := client.SetPathProperties(context.Background()); err != nil {
		t.Fatalf("SetPathProperties() unexpected error: %v", err)
	}

	if len(mockManager.SetPropsCalls) != 1 {
		t.Fatalf("expected 1 call to SetProps, got %d", len(mockManager.SetPropsCalls))
	}

	call := mockManager.SetPropsCalls[0]
	gotPairs := strings.Split(call.Props, ";")
	if len(gotPairs) != len(exampleProperties()) {
		t.Fatalf("SetProps() Props = %q, expected %d key=value pairs", call.Props, len(exampleProperties()))
	}
	for key, value := range exampleProperties() {
		if !slices.Contains(gotPairs, fmt.Sprintf("%s=%s", key, value)) {
			t.Errorf("SetProps() Props = %q, missing expected pair %q", call.Props, fmt.Sprintf("%s=%s", key, value))
		}
	}

	if item.Repo != mockArtifactoryRepo || item.Path != "." || item.Name != testArtifactPath || item.Type != "folder" {
		t.Errorf("unexpected folder record: %+v", item)
	}
}

// TestSetPathPropertiesError verifies that a non-retryable SetProps error is
// propagated as-is. max_retries is set to 0 to avoid the retry backoff sleep.
func TestSetPathPropertiesError(t *testing.T) {
	mockUI := &capturingTestUI{}
	mockManager := &mockServicesManager{
		SetPropsFunc: func(services.PropsParams) (int, error) {
			return 0, fmt.Errorf("boom")
		},
	}
	client := &artifactoryClient{
		ui:              mockUI,
		servicesManager: mockManager,
		retryLogic:      newRetryLogic(0, 30, mockUI),
		config: artifactoryConfig{
			ArtifactPathProperties: true,
			Repository:             mockArtifactoryRepo,
			ArtifactPath:           testArtifactPath,
			Properties:             exampleProperties(),
		},
	}

	err := client.SetPathProperties(context.Background())
	if err == nil {
		t.Fatal("SetPathProperties() expected error but got nil")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("SetPathProperties() error = %v, expected to contain %q", err, "boom")
	}
}

// TestSetPathPropertiesProLicenseRequired verifies that a "Set Item
// Properties" 400 response indicating the REST API is Pro/Enterprise-only is
// treated as a graceful, one-time-warned skip rather than a retried failure
// (this response is permanent, not transient, so retrying wastes time and
// unnecessarily fails builds on Artifactory instances without that license).
func TestSetPathPropertiesProLicenseRequired(t *testing.T) {
	logOutput := captureLogOutput(t)
	mockUI := &capturingTestUI{}
	mockManager := &mockServicesManager{
		SetPropsFunc: func(services.PropsParams) (int, error) {
			return 0, fmt.Errorf(`server response: 400 Bad Request
{
  "errors": [
    {
      "status": 400,
      "message": "This REST API is available only in Artifactory Pro (see: jfrog.com/artifactory/features). If you are already running Artifactory Pro please make sure your server is activated with a valid license key.\n"
    }
  ]
}`)
		},
	}
	client := &artifactoryClient{
		ui:              mockUI,
		servicesManager: mockManager,
		retryLogic:      newRetryLogic(3, 30, mockUI),
		config: artifactoryConfig{
			ArtifactPathProperties: true,
			Repository:             mockArtifactoryRepo,
			ArtifactPath:           testArtifactPath,
			Properties:             exampleProperties(),
		},
	}

	if err := client.SetPathProperties(context.Background()); err != nil {
		t.Fatalf("SetPathProperties() unexpected error: %v", err)
	}

	if len(mockManager.SetPropsCalls) != 1 {
		t.Errorf("expected exactly 1 call to SetProps (no retries), got %d", len(mockManager.SetPropsCalls))
	}

	if !strings.Contains(logOutput.String(), "only available in Artifactory Pro/Enterprise") {
		t.Errorf("expected a [WARN] log about the Pro/Enterprise license requirement, got log output: %v", logOutput.String())
	}
}

// TestConfigAuthenticationMethods tests various authentication method
// configurations.
func TestConfigAuthenticationMethods(t *testing.T) {
	clearArtifactoryEnvVars(t)

	tests := []struct {
		name        string
		config      artifactoryConfig
		expectValid bool
	}{
		{
			name: "API key authentication",
			config: artifactoryConfig{
				URL:          mockArtifactoryURL,
				Repository:   mockArtifactoryRepo,
				ArtifactName: mockArtifactName,
				APIKey:       mockArtifactoryAPIKey,
			},
			expectValid: true,
		},
		{
			name: "Access token authentication",
			config: artifactoryConfig{
				URL:          mockArtifactoryURL,
				Repository:   mockArtifactoryRepo,
				ArtifactName: mockArtifactName,
				AccessToken:  mockArtifactoryAccessToken,
			},
			expectValid: true,
		},
		{
			name: "Username/password authentication",
			config: artifactoryConfig{
				URL:          mockArtifactoryURL,
				Repository:   mockArtifactoryRepo,
				ArtifactName: mockArtifactName,
				Username:     mockArtifactoryUsername,
				Password:     mockArtifactoryPassword,
			},
			expectValid: true,
		},
		{
			name: "Multiple authentication methods (should be invalid)",
			config: artifactoryConfig{
				URL:          mockArtifactoryURL,
				Repository:   mockArtifactoryRepo,
				ArtifactName: mockArtifactName,
				APIKey:       mockArtifactoryAPIKey,
				AccessToken:  mockArtifactoryAccessToken,
				Username:     mockArtifactoryUsername,
				Password:     mockArtifactoryPassword,
			},
			expectValid: false,
		},
		{
			name: "No authentication",
			config: artifactoryConfig{
				URL:          mockArtifactoryURL,
				Repository:   mockArtifactoryRepo,
				ArtifactName: mockArtifactName,
			},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
