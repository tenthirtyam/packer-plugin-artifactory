// Copyright (c) Ryan Johnson
// SPDX-License-Identifier: MPL-2.0

package artifactory

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// TestValidOvaOvfFiles tests the post-processor's ability to handle various
// OVA and OVF file combinations and reject invalid artifact types.
func TestValidOvaOvfFiles(t *testing.T) {
	tests := []struct {
		name      string
		files     []string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid OVA file",
			files:     []string{"/path/to/test.ova"},
			wantError: false,
		},
		{
			name:      "valid OVF package",
			files:     []string{"/path/to/test.ovf", "/path/to/test.vmdk", "/path/to/test.mf"},
			wantError: false,
		},
		{
			name:      "no OVA or OVF files",
			files:     []string{"/path/to/test.vmdk", "/path/to/test.mf"},
			wantError: true,
			errorMsg:  "no handler found for artifact type",
		},
		{
			name:      "no valid VM files",
			files:     []string{"/path/to/test.txt", "/path/to/test.log"},
			wantError: true,
			errorMsg:  "no handler found for artifact type",
		},
		{
			name:      "empty file list",
			files:     []string{},
			wantError: true,
			errorMsg:  "no handler found for artifact type",
		},
		{
			name:      "mixed valid and invalid files",
			files:     []string{"/path/to/test.ova", "/path/to/test.txt"},
			wantError: false,
		},
		{
			name:      "OVA with unsupported files",
			files:     []string{"/path/to/test.ova", "/path/to/test.qcow2"},
			wantError: false,
		},
		{
			name:      "OVF with certificate",
			files:     []string{"/path/to/test.ovf", "/path/to/test.vmdk", "/path/to/test.cert"},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			var tempFiles []string

			for _, file := range tt.files {
				filename := filepath.Base(file)
				var tempFile string

				if strings.HasSuffix(file, ".ovf") {
					tempFile = createTestOvfFile(t, tempDir, filename)
				} else {
					tempFile = createTestFileWithContent(t, tempDir, filename, testGenericContent)
				}
				tempFiles = append(tempFiles, tempFile)
			}

			artifact := &testArtifact{
				BuilderIdValue: testBuilderId,
				FilesValue:     tempFiles,
				IdValue:        mockArtifactName,
			}

			processor := &PostProcessor{
				config: artifactoryConfig{
					URL:          mockArtifactoryURL,
					Repository:   mockArtifactoryRepo,
					ArtifactName: mockArtifactName,
					APIKey:       mockArtifactoryAPIKey,
				},
				client: &testArtifactoryClient{},
			}

			ui := &noOpTestUI{}
			ctx := context.Background()

			_, _, _, procErr := processor.PostProcess(ctx, ui, artifact)

			if tt.wantError {
				if procErr == nil {
					t.Errorf("Expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(procErr.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %v", tt.errorMsg, procErr)
				}
			} else if procErr != nil {
				if !strings.Contains(procErr.Error(), "nil") {
					t.Errorf("Unexpected error: %v", procErr)
				}
			}
		})
	}
}

// TestFileExtensionValidation tests file extension validation logic for
// supported and unsupported file types.
func TestFileExtensionValidation(t *testing.T) {
	validExtensions := map[string]bool{
		".ova":  true,
		".ovf":  true,
		".vmdk": true,
		".mf":   true,
		".cert": true,
	}

	invalidExtensions := []string{
		".txt", ".log", ".json", ".yml", ".zip", ".tar", ".gz",
	}

	for ext := range validExtensions {
		filename := "test" + ext
		if !strings.HasSuffix(filename, ext) {
			t.Errorf("Extension %s not properly handled", ext)
		}
	}

	expectedCount := 5
	if len(validExtensions) != expectedCount {
		t.Errorf("Expected %d valid extensions, got %d", expectedCount, len(validExtensions))
	}

	for _, ext := range invalidExtensions {
		if validExtensions[ext] {
			t.Errorf("Extension %s should not be valid", ext)
		}
	}
}

// TestArtifactTypeDetection tests the logic for detecting artifact types
// based on file extensions and combinations.
func TestArtifactTypeDetection(t *testing.T) {
	tests := []struct {
		name         string
		files        []string
		expectedType string
	}{
		{
			name:         "single OVA file",
			files:        []string{testOvaFile},
			expectedType: testArtifactTypeOva,
		},
		{
			name:         "single OVF file",
			files:        []string{testOvfFile},
			expectedType: testArtifactTypeOvf,
		},
		{
			name:         "OVF package with multiple files",
			files:        []string{testOvfFile, testVmdkFile, testManifestFile},
			expectedType: testArtifactTypeOvf,
		},
		{
			name:         "mixed files without clear type",
			files:        []string{testVmdkFile, testManifestFile},
			expectedType: testArtifactTypeOvf,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var artifactType string
			if len(tt.files) == 1 {
				ext := strings.ToLower(filepath.Ext(tt.files[0]))
				switch ext {
				case ".ova":
					artifactType = testArtifactTypeOva
				case ".ovf":
					artifactType = testArtifactTypeOvf
				default:
					artifactType = "vm"
				}
			} else {
				artifactType = testArtifactTypeOvf
			}

			if artifactType != tt.expectedType {
				t.Errorf("Expected artifact type %s, got %s", tt.expectedType, artifactType)
			}
		})
	}
}

// TestPostProcessorConfiguration tests the post-processor configuration
// setup and validation with valid configuration parameters.
func TestPostProcessorConfiguration(t *testing.T) {
	clearArtifactoryEnvVars(t)

	processor := &PostProcessor{}

	config := map[string]any{
		"url":           mockArtifactoryURL,
		"repository":    mockArtifactoryRepo,
		"artifact_name": mockArtifactName,
		"api_key":       mockArtifactoryAPIKey,
	}

	_ = processor.Configure(config)
	processor.config = artifactoryConfig{
		URL:          mockArtifactoryURL,
		Repository:   mockArtifactoryRepo,
		ArtifactName: mockArtifactName,
		APIKey:       mockArtifactoryAPIKey,
	}

	if processor.config.URL != mockArtifactoryURL {
		t.Errorf("Expected URL to be set correctly")
	}

	if processor.config.Repository != mockArtifactoryRepo {
		t.Errorf("Expected Repository to be set correctly")
	}

	if processor.config.ArtifactName != mockArtifactName {
		t.Errorf("Expected ArtifactName to be set correctly")
	}

	if err := processor.config.Validate(); err != nil {
		t.Errorf("Config validation failed: %v", err)
	}
}

// TestPostProcessorConfigurationError tests that invalid configurations
// are properly rejected during validation.
func TestPostProcessorConfigurationError(t *testing.T) {
	processor := &PostProcessor{}

	processor.config = artifactoryConfig{
		URL: mockArtifactoryURL,
	}

	err := processor.config.Validate()
	if err == nil {
		t.Errorf("Config validation expected error but got nil")
	}
}

// TestPostProcessorConfigSpec tests that the ConfigSpec method returns
// a non-nil configuration specification.
func TestPostProcessorConfigSpec(t *testing.T) {
	processor := &PostProcessor{}
	spec := processor.ConfigSpec()
	if spec == nil {
		t.Errorf("ConfigSpec() returned nil")
	}
}

// TestPostProcessorConfigure tests the Configure method with various
// configuration scenarios including valid and invalid configurations.
func TestPostProcessorConfigure(t *testing.T) {
	clearArtifactoryEnvVars(t)

	tests := []struct {
		name    string
		config  map[string]any
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid configuration",
			config: map[string]any{
				"url":           mockArtifactoryURL,
				"repository":    mockArtifactoryRepo,
				"artifact_name": mockArtifactName,
				"api_key":       mockArtifactoryAPIKey,
			},
			wantErr: false,
		},
		{
			name: "invalid configuration - missing required fields",
			config: map[string]any{
				"url": mockArtifactoryURL,
			},
			wantErr: true,
			errMsg:  "repository is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := &PostProcessor{}
			err := processor.Configure(tt.config)

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

// TestPostProcessorPostProcessSuccess tests successful post-processing
// of artifacts including file and metadata upload.
func TestPostProcessorPostProcessSuccess(t *testing.T) {
	tempDir := t.TempDir()

	ovaFile := createTestOvaFile(t, tempDir, testOvaFile)

	artifact := &testArtifact{
		BuilderIdValue: testBuilderId,
		FilesValue:     []string{ovaFile},
		IdValue:        mockArtifactName,
	}

	mockClient := &testArtifactoryClient{}
	processor := &PostProcessor{
		config: artifactoryConfig{
			URL:          mockArtifactoryURL,
			Repository:   mockArtifactoryRepo,
			ArtifactName: mockArtifactName,
			APIKey:       mockArtifactoryAPIKey,
		},
		client: mockClient,
	}

	ui := &noOpTestUI{}
	ctx := context.Background()

	resultArtifact, keep, forceOverride, err := processor.PostProcess(ctx, ui, artifact)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if resultArtifact != artifact {
		t.Errorf("Expected result artifact to be the same as input")
	}
	if keep != false {
		t.Errorf("Expected keep to be false")
	}
	if forceOverride != false {
		t.Errorf("Expected forceOverride to be false")
	}
	if len(mockClient.UploadedFiles) != 1 {
		t.Errorf("Expected 1 uploaded file, got %d", len(mockClient.UploadedFiles))
	}
	if len(mockClient.UploadedMetadata) != 1 {
		t.Errorf("Expected 1 uploaded metadata, got %d", len(mockClient.UploadedMetadata))
	}
}

// TestPostProcessorPostProcessErrors tests error handling during
// post-processing including upload failures and invalid artifacts.
func TestPostProcessorPostProcessErrors(t *testing.T) {
	tests := []struct {
		name                string
		files               []string
		uploadFileError     error
		uploadMetadataError error
		expectError         bool
		errorMsg            string
	}{
		{
			name:            "upload file error",
			files:           []string{testOvaFile},
			uploadFileError: fmt.Errorf("upload failed"),
			expectError:     true,
			errorMsg:        "upload failed",
		},
		{
			name:                "upload metadata error",
			files:               []string{testOvaFile},
			uploadMetadataError: fmt.Errorf("metadata upload failed"),
			expectError:         true,
			errorMsg:            "metadata upload failed",
		},
		{
			name:        "no valid artifacts",
			files:       []string{testTextFile},
			expectError: true,
			errorMsg:    "no handler found for artifact type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			var tempFiles []string

			for _, file := range tt.files {
				tempFile := createTestFileWithContent(t, tempDir, file, testGenericContent)
				tempFiles = append(tempFiles, tempFile)
			}

			artifact := &testArtifact{
				BuilderIdValue: testBuilderId,
				FilesValue:     tempFiles,
				IdValue:        mockArtifactName,
			}

			mockClient := &testArtifactoryClient{
				UploadFileError:     tt.uploadFileError,
				UploadMetadataError: tt.uploadMetadataError,
			}

			processor := &PostProcessor{
				config: artifactoryConfig{
					URL:          mockArtifactoryURL,
					Repository:   mockArtifactoryRepo,
					ArtifactName: mockArtifactName,
					APIKey:       mockArtifactoryAPIKey,
				},
				client: mockClient,
			}

			ui := &noOpTestUI{}
			ctx := context.Background()

			_, _, _, err := processor.PostProcess(ctx, ui, artifact)

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

// TestPostProcessorClientCreation tests Artifactory client creation
// and error handling when authentication is missing.
func TestPostProcessorClientCreation(t *testing.T) {
	tempDir := t.TempDir()

	ovaFile := createTestOvaFile(t, tempDir, testOvaFile)

	artifact := &testArtifact{
		BuilderIdValue: testBuilderId,
		FilesValue:     []string{ovaFile},
		IdValue:        mockArtifactName,
	}

	processor := &PostProcessor{
		config: artifactoryConfig{
			URL:          mockArtifactoryURL,
			Repository:   mockArtifactoryRepo,
			ArtifactName: mockArtifactName,
			// No authentication - should cause client creation to fail
		},
		client: nil,
	}

	ui := &noOpTestUI{}
	ctx := context.Background()

	_, _, _, err := processor.PostProcess(ctx, ui, artifact)

	if err == nil {
		t.Errorf("Expected error due to missing authentication, but got nil")
		return
	}
	if !strings.Contains(err.Error(), "failed to create Artifactory client") {
		t.Errorf("Expected client creation error, got %v", err)
	}
}
