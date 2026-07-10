// Copyright (c) Ryan Johnson
// SPDX-License-Identifier: MPL-2.0

package artifactory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestOvaOvfHandler_CanHandle tests the CanHandle method of the OVA/OVF handler
// with various file combinations to ensure proper artifact type detection.
func TestOvaOvfHandler_CanHandle(t *testing.T) {
	handler := newOvaOvfHandler()

	tests := []struct {
		name     string
		files    []string
		expected bool
	}{
		{
			name:     "single OVA file",
			files:    []string{"/path/to/test.ova"},
			expected: true,
		},
		{
			name:     "OVF package with components",
			files:    []string{"/path/to/test.ovf", "/path/to/test.vmdk", "/path/to/test.mf"},
			expected: true,
		},
		{
			name:     "no OVA or OVF files",
			files:    []string{"/path/to/test.vmdk", "/path/to/test.mf"},
			expected: false,
		},
		{
			name:     "no valid VM files",
			files:    []string{"/path/to/test.txt", "/path/to/test.log"},
			expected: false,
		},
		{
			name:     "empty file list",
			files:    []string{},
			expected: false,
		},
		{
			name:     "mixed valid and invalid files",
			files:    []string{"/path/to/test.ova", "/path/to/test.txt"},
			expected: true,
		},
		{
			name:     "OVF with certificate",
			files:    []string{"/path/to/test.ovf", "/path/to/test.vmdk", "/path/to/test.cert"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			artifact := &testArtifact{FilesValue: tt.files}
			result := handler.CanHandle(artifact)

			if result != tt.expected {
				t.Errorf("CanHandle() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// TestOvaOvfHandler_ProcessArtifact tests the ProcessArtifact method with
// various artifact types and validates proper processing and error handling.
func TestOvaOvfHandler_ProcessArtifact(t *testing.T) {
	handler := newOvaOvfHandler()

	tests := []struct {
		name         string
		files        []string
		expectError  bool
		errorMsg     string
		expectedType string
	}{
		{
			name:         "single OVA file",
			files:        []string{"/path/to/test.ova"},
			expectError:  false,
			expectedType: testArtifactTypeOva,
		},
		{
			name:         "OVF package",
			files:        []string{"/path/to/test.ovf", "/path/to/test.vmdk", "/path/to/test.mf"},
			expectError:  false,
			expectedType: testArtifactTypeOvf,
		},
		{
			name:        "no OVA or OVF files",
			files:       []string{"/path/to/test.vmdk", "/path/to/test.mf"},
			expectError: true,
			errorMsg:    "no OVA or OVF descriptor file found",
		},
		{
			name:        "no valid VM files",
			files:       []string{"/path/to/test.txt", "/path/to/test.log"},
			expectError: true,
			errorMsg:    "no VM files matched the upload allowlist",
		},
		{
			name:        "empty file list",
			files:       []string{},
			expectError: true,
			errorMsg:    "no files found in artifact",
		},
		{
			name:         "mixed valid and invalid files",
			files:        []string{"/path/to/test.ova", "/path/to/test.txt"},
			expectError:  false,
			expectedType: testArtifactTypeOva,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var testFiles []string
			if !tt.expectError && len(tt.files) > 0 {
				tempDir := t.TempDir()
				for _, file := range tt.files {
					filename := filepath.Base(file)
					var tempFile string

					if strings.HasSuffix(file, ".ovf") {
						tempFile = createTestOvfFile(t, tempDir, filename)
					} else {
						tempFile = createTestFileWithContent(t, tempDir, filename, testGenericContent)
					}
					testFiles = append(testFiles, tempFile)
				}
			} else {
				testFiles = tt.files
			}

			artifact := &testArtifact{FilesValue: testFiles}
			result, err := handler.ProcessArtifact(artifact)

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
				if result == nil {
					t.Errorf("Expected result but got nil")
				} else {
					if result.Metadata["type"] != tt.expectedType {
						t.Errorf("Expected type %q, got %q", tt.expectedType, result.Metadata["type"])
					}
				}
			}
		})
	}
}

// TestOvaOvfHandler_OvaExportDropsOvfStaging tests Fusion/Workstation-style artifacts
// that list both a packaged .ova and leftover OVF directory files.
func TestOvaOvfHandler_OvaExportDropsOvfStaging(t *testing.T) {
	handler := newOvaOvfHandler()
	tempDir := t.TempDir()

	ovfDir := filepath.Join(tempDir, "output")
	if err := os.MkdirAll(ovfDir, 0o755); err != nil {
		t.Fatal(err)
	}

	ovfPath := createTestOvfFile(t, ovfDir, "debian.ovf")
	vmdkPath := createTestFileWithContent(t, ovfDir, "debian-disk1.vmdk", testGenericContent)
	mfPath := createTestFileWithContent(t, ovfDir, "debian.mf", testGenericContent)
	ovaPath := createTestFileWithContent(t, ovfDir, "debian.ova", testOvaContent)

	artifact := &testArtifact{FilesValue: []string{ovfPath, vmdkPath, mfPath, ovaPath}}
	result, err := handler.ProcessArtifact(artifact)
	if err != nil {
		t.Fatalf("ProcessArtifact: %v", err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("expected 1 upload file (.ova only), got %d: %v", len(result.Files), result.Files)
	}
	if filepath.Base(result.Files[0]) != "debian.ova" {
		t.Errorf("expected debian.ova, got %s", result.Files[0])
	}
	if result.Metadata["type"] != testArtifactTypeOva {
		t.Errorf("metadata type = %q, want %q", result.Metadata["type"], testArtifactTypeOva)
	}
}

// TestOvaOvfHandler_DefaultOmitsNvram drops .nvram from multi-file uploads unless included explicitly.
func TestOvaOvfHandler_DefaultOmitsNvram(t *testing.T) {
	handler := newOvaOvfHandler()
	tempDir := t.TempDir()

	ovfPath := createTestOvfFile(t, tempDir, "guest.ovf")
	vmdkPath := createTestFileWithContent(t, tempDir, "guest.vmdk", testGenericContent)
	nvramPath := createTestFileWithContent(t, tempDir, "guest.nvram", testGenericContent)

	artifact := &testArtifact{FilesValue: []string{ovfPath, vmdkPath, nvramPath}}
	result, err := handler.ProcessArtifact(artifact)
	if err != nil {
		t.Fatalf("ProcessArtifact: %v", err)
	}
	if len(result.Files) != 2 {
		t.Fatalf("expected ovf+vmdk only (2 files), got %d: %v", len(result.Files), result.Files)
	}
	for _, p := range result.Files {
		if strings.HasSuffix(p, ".nvram") {
			t.Errorf("nvram should be excluded, still have %s", p)
		}
	}
}

// TestOvaOvfHandler_GetHandlerName tests that the handler returns the
// correct name identifier.
func TestOvaOvfHandler_GetHandlerName(t *testing.T) {
	handler := newOvaOvfHandler()
	name := handler.GetHandlerName()

	if name != ovaOvfHandlerName {
		t.Errorf("Expected handler name %q, got %q", ovaOvfHandlerName, name)
	}
}

// TestOvaOvfHandler_ValidateOvfDescriptor tests OVF descriptor validation
// with various XML content scenarios including valid and invalid formats.
func TestOvaOvfHandler_ValidateOvfDescriptor(t *testing.T) {
	handler := newOvaOvfHandler()
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		ovfContent  string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid OVF descriptor",
			ovfContent: `<?xml version="1.0" encoding="UTF-8"?>
<Envelope xmlns="http://schemas.dmtf.org/ovf/envelope/1"
          xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <VirtualSystem ovf:id="vm">
    <Name>Test VM</Name>
  </VirtualSystem>
</Envelope>`,
			expectError: false,
		},
		{
			name:        "invalid XML",
			ovfContent:  `<?xml version="1.0"?><Envelope><unclosed>`,
			expectError: true,
			errorMsg:    "invalid XML structure",
		},
		{
			name:        "empty file",
			ovfContent:  "",
			expectError: true,
			errorMsg:    "invalid XML structure",
		},
		{
			name:        "non-XML content",
			ovfContent:  "This is not XML",
			expectError: true,
			errorMsg:    "invalid XML structure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ovfFile := createTestFileWithContent(t, tempDir, testOvfFile, tt.ovfContent)

			err := handler.validateOvfDescriptor(ovfFile)

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

// TestOvaOvfHandler_EnhancedFileSupport tests the handler's ability to
// process OVF packages with additional file types like NVRAM and ISO files.
func TestOvaOvfHandler_EnhancedFileSupport(t *testing.T) {
	tests := []struct {
		name              string
		files             []string
		included          []string
		canHandle         bool
		expectError       bool
		expectedType      string
		expectMinFileSize int
	}{
		{
			name:              "OVF with NVRAM file (included)",
			files:             []string{"/path/to/test.ovf", "/path/to/test.vmdk", "/path/to/test.nvram"},
			included:          []string{"nvram"},
			canHandle:         true,
			expectError:       false,
			expectedType:      testArtifactTypeOvf,
			expectMinFileSize: 3,
		},
		{
			name:              "OVF with ISO file (included)",
			files:             []string{"/path/to/test.ovf", "/path/to/test.vmdk", "/path/to/test.iso"},
			included:          []string{"iso"},
			canHandle:         true,
			expectError:       false,
			expectedType:      testArtifactTypeOvf,
			expectMinFileSize: 3,
		},
		{
			name:              "OVF with NVRAM and ISO when both included",
			files:             []string{"/path/to/test.ovf", "/path/to/test.vmdk", "/path/to/test.mf", "/path/to/test.cert", "/path/to/test.nvram", "/path/to/test.iso"},
			included:          []string{"nvram", "iso"},
			canHandle:         true,
			expectError:       false,
			expectedType:      testArtifactTypeOvf,
			expectMinFileSize: 6,
		},
		{
			name:      "only NVRAM and ISO files",
			files:     []string{"/path/to/test.nvram", "/path/to/test.iso"},
			included:  nil,
			canHandle: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newOvaOvfHandlerWithIncluded(tt.included)
			artifact := &testArtifact{FilesValue: tt.files}

			canHandle := handler.CanHandle(artifact)
			if canHandle != tt.canHandle {
				t.Errorf("CanHandle() = %v, expected %v", canHandle, tt.canHandle)
			}

			if tt.canHandle {
				tempDir := t.TempDir()
				var tempFiles []string

				for _, file := range tt.files {
					tempFile := filepath.Join(tempDir, filepath.Base(file))
					var content []byte

					// Create valid OVF content for .ovf files
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

					err := os.WriteFile(tempFile, content, 0600)
					if err != nil {
						t.Fatalf("Failed to create temp file: %v", err)
					}
					tempFiles = append(tempFiles, tempFile)
				}

				tempArtifact := &testArtifact{FilesValue: tempFiles}
				result, err := handler.ProcessArtifact(tempArtifact)

				if tt.expectError {
					if err == nil {
						t.Errorf("Expected error but got nil")
					}
				} else {
					if err != nil {
						t.Errorf("Unexpected error: %v", err)
					}
					if result != nil && result.Metadata["type"] != tt.expectedType {
						t.Errorf("Expected type %q, got %q", tt.expectedType, result.Metadata["type"])
					}
					if !tt.expectError && tt.expectMinFileSize > 0 && len(result.Files) < tt.expectMinFileSize {
						t.Errorf("expected at least %d upload files, got %d: %v", tt.expectMinFileSize, len(result.Files), result.Files)
					}
				}
			}
		})
	}
}

// TestOvaOvfHandler_DirectoryStructurePreservation tests that uploads stay on
// the portable allowlist: paths listed in the artifact are kept only when their
// extension matches; unrelated paths (e.g. outside the OVF folder) are omitted.
func TestOvaOvfHandler_DirectoryStructurePreservation(t *testing.T) {
	handler := newOvaOvfHandler()
	tempDir := t.TempDir()

	ovfDir := filepath.Join(tempDir, testDirectoryName)
	err := os.MkdirAll(ovfDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create OVF directory: %v", err)
	}

	ovfFile := createTestOvfFile(t, ovfDir, testOvfFile)

	associatedFiles := []string{testVmdkFile, testManifestFile, "unknown.bin"}
	var allFiles []string
	allFiles = append(allFiles, ovfFile)

	for _, filename := range associatedFiles {
		filePath := createTestFileWithContent(t, ovfDir, filename, testGenericContent)
		allFiles = append(allFiles, filePath)
	}

	outsideFile := createTestFileWithContent(t, tempDir, testOutsideFile, "outside content")
	allFiles = append(allFiles, outsideFile)

	artifact := &testArtifact{FilesValue: allFiles}
	result, err := handler.ProcessArtifact(artifact)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(result.Files) != 3 {
		t.Fatalf("expected ovf+vmdk+mf (3 files), got %d: %v", len(result.Files), result.Files)
	}

	for _, file := range result.Files {
		if strings.Contains(file, "unknown.bin") {
			t.Errorf("unexpected .bin in portable set (use additional_ovf_extensions to allow)")
		}
	}

	foundOutside := false
	for _, file := range result.Files {
		if strings.Contains(file, testOutsideFile) {
			foundOutside = true
			break
		}
	}

	if foundOutside {
		t.Errorf("Expected outside file to not be included")
	}
}
