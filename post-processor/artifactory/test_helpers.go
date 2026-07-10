// Copyright (c) Ryan Johnson
// SPDX-License-Identifier: MPL-2.0

package artifactory

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
)

const (
	// mockArtifactoryURL is the example URL for Artifactory.
	mockArtifactoryURL = "https://packages.example.com/artifactory/"
	// mockArtifactoryURLNoSlash is the example URL for Artifactory without
	// a trailing slash used in URL normalization tests.
	mockArtifactoryURLNoSlash = "https://packages.example.com/artifactory"
	// mockArtifactoryUsername is the example username used in mock tests.
	mockArtifactoryUsername = "test-user"
	// mockArtifactoryPassword is the example password used in mock tests.
	mockArtifactoryPassword = "test-password"
	// mockArtifactoryRepo is the example repository name used in mock tests.
	mockArtifactoryRepo = "test-repo"
	// mockArtifactName is the example artifact name used in mock tests.
	mockArtifactName = "test-artifact"
	// mockArtifactoryAPIKey is the example API key used in mock tests.
	mockArtifactoryAPIKey = "test-api-key"
	// mockArtifactoryAccessToken is the example access token used in mock tests.
	mockArtifactoryAccessToken = "test-access-token"

	// Test artifact types.
	testArtifactTypeOva = "ova"
	testArtifactTypeOvf = "ovf"

	// Test builder IDs.
	testBuilderId = "test-builder"

	// Test file names.
	testOvaFile      = "test.ova"
	testOvfFile      = "test.ovf"
	testVmdkFile     = "test.vmdk"
	testManifestFile = "test.mf"
	testCertFile     = "test.cert"
	testNvramFile    = "test.nvram"
	testIsoFile      = "test.iso"
	testLogFile      = "test.log"
	testTextFile     = "test.txt"
	testOutsideFile  = "outside.txt"

	// Test paths and directories.
	testArtifactPath  = "builds/test"
	testNestedPath    = "team/project/version/1.0.0"
	testDirectoryName = "vm-package"

	// Test timestamps and IDs.
	testTimestamp = "1234567890"
	testID        = "test-id"

	// Test descriptions.
	testDescription = "Test description."

	// Test handler names.
	testHandlerName   = "test-handler"
	ovaOvfHandlerName = "ova-ovf"
	firstHandlerName  = "first-handler"
	secondHandlerName = "second-handler"
	newHandlerName    = "new-handler"

	// testOvfContent is a valid OVF XML descriptor for testing.
	testOvfContent = `<?xml version="1.0"?>
<Envelope xmlns="http://schemas.dmtf.org/ovf/envelope/1">
  <VirtualSystem ovf:id="vm">
    <Name>test</Name>
  </VirtualSystem>
</Envelope>`

	// testOvaContent is sample OVA file content for testing.
	testOvaContent = "test ova content"
	// testGenericContent is generic test file content.
	testGenericContent = "test content"
)

// Variables that use environment variables with mock defaults.
var (
	testArtifactoryURL = getEnvOrDefault("ARTIFACTORY_URL", mockArtifactoryURL)
	testUsername       = getEnvOrDefault("ARTIFACTORY_USERNAME", mockArtifactoryUsername)
	testPassword       = resolveArtifactoryPassword()
	testRepository     = getEnvOrDefault("ARTIFACTORY_REPOSITORY", mockArtifactoryRepo)
)

// exampleProperties returns a fresh copy of the properties block from
// docs/README.md (same keys, same order, same values), so property-handling
// tests exercise a realistic, documented configuration rather than ad hoc
// keys.
func exampleProperties() map[string]string {
	return map[string]string{
		"version.number":  "1.0.0",
		"build.number":    "12345678",
		"release.channel": "stable",
		"os.family":       "linux",
		"os.vendor":       "debian",
		"os.version":      "13.6.0",
		"os.arch":         "amd64",
	}
}

// getEnvOrDefault returns the environment variable value or a default if not set.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// resolveArtifactoryPassword returns password auth for integration tests.
// Prefers ARTIFACTORY_PASSWORD, then API key, then access token.
func resolveArtifactoryPassword() string {
	if value := os.Getenv("ARTIFACTORY_PASSWORD"); value != "" {
		return value
	}
	if value := os.Getenv("ARTIFACTORY_API_KEY"); value != "" {
		return value
	}
	if value := os.Getenv("ARTIFACTORY_TOKEN"); value != "" {
		return value
	}
	return mockArtifactoryPassword
}

// captureLogOutput redirects the standard "log" package output to a buffer
// for the duration of the test (restoring it on cleanup), for asserting on
// [WARN]-style messages that artifactoryClient.warnOnce logs instead of
// surfacing in the build UI.
func captureLogOutput(t *testing.T) *bytes.Buffer {
	var buf bytes.Buffer
	original := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() {
		log.SetOutput(original)
	})
	return &buf
}

// clearArtifactoryEnvVars clears all Artifactory-related environment variables for testing.
func clearArtifactoryEnvVars(t *testing.T) {
	envVars := []string{
		"ARTIFACTORY_API_KEY",
		"ARTIFACTORY_TOKEN",
		"ARTIFACTORY_USERNAME",
		"ARTIFACTORY_PASSWORD",
		"ARTIFACTORY_URL",
		"ARTIFACTORY_REPOSITORY",
	}

	for _, envVar := range envVars {
		if err := os.Unsetenv(envVar); err != nil {
			t.Logf("Failed to unset %s: %v", envVar, err)
		}
	}
}

// testArtifact is a consolidated mock implementation of the packer.Artifact
// interface for all testing scenarios.
type testArtifact struct {
	BuilderIdValue string
	FilesValue     []string
	IdValue        string
	StringValue    string
}

// BuilderId returns the builder ID for the test artifact.
func (a *testArtifact) BuilderId() string {
	return a.BuilderIdValue
}

// Files returns the list of files associated with the test artifact.
func (a *testArtifact) Files() []string {
	return a.FilesValue
}

// Id returns the unique identifier for the test artifact.
func (a *testArtifact) Id() string {
	if a.IdValue != "" {
		return a.IdValue
	}
	return testID
}

// String returns a string representation of the test artifact.
func (a *testArtifact) String() string {
	if a.StringValue != "" {
		return a.StringValue
	}
	return a.Id()
}

// State returns the state value for the given name (always returns nil for test).
func (a *testArtifact) State(_ string) any {
	return nil
}

// Destroy cleans up the test artifact (no-op for testing).
func (a *testArtifact) Destroy() error {
	return nil
}

// testHandler is a consolidated mock implementation of the artifactHandler
// interface for testing with configurable behavior.
type testHandler struct {
	Name              string
	CanHandleArtifact bool
	ShouldError       bool
	ReturnType        string
}

// CanHandle returns whether this test handler can process the given artifact.
func (h *testHandler) CanHandle(_ packer.Artifact) bool {
	return h.CanHandleArtifact
}

// ProcessArtifact processes the given artifact and returns a processedArtifact
// or an error based on the test configuration.
func (h *testHandler) ProcessArtifact(_ packer.Artifact) (*processedArtifact, error) {
	if h.ShouldError {
		return nil, fmt.Errorf("mock error")
	}

	artifactType := h.ReturnType
	if artifactType == "" {
		artifactType = testArtifactTypeOva
	}

	return &processedArtifact{
		Files:    []string{testOvaFile},
		Metadata: map[string]string{"type": artifactType},
	}, nil
}

// GetHandlerName returns the name of this test handler.
func (h *testHandler) GetHandlerName() string {
	if h.Name != "" {
		return h.Name
	}
	return testHandlerName
}

// testArtifactoryClient is a consolidated mock implementation of the
// artifactoryClient interface for testing upload functionality.
type testArtifactoryClient struct {
	UploadFileError        error
	UploadMetadataError    error
	SetPathPropertiesError error
	UploadedFiles          []string
	UploadedMetadata       []artifactMetadata
	SetPathPropertiesCalls int
}

// UploadFile simulates file upload to Artifactory, returning configured error
// or recording the uploaded filename.
func (c *testArtifactoryClient) UploadFile(_ context.Context, _, filename string) error {
	if c.UploadFileError != nil {
		return c.UploadFileError
	}
	c.UploadedFiles = append(c.UploadedFiles, filename)
	return nil
}

// UploadMetadata simulates metadata upload to Artifactory, returning configured
// error or recording the uploaded metadata.
func (c *testArtifactoryClient) UploadMetadata(_ context.Context, metadata artifactMetadata) error {
	if c.UploadMetadataError != nil {
		return c.UploadMetadataError
	}
	c.UploadedMetadata = append(c.UploadedMetadata, metadata)
	return nil
}

// SetPathProperties simulates setting properties on the artifact_path folder,
// returning a configured error or recording that it was called.
func (c *testArtifactoryClient) SetPathProperties(_ context.Context) error {
	c.SetPathPropertiesCalls++
	if c.SetPathPropertiesError != nil {
		return c.SetPathPropertiesError
	}
	return nil
}

// mockServicesManager is a minimal mock implementation of the
// artifactory.ArtifactoryServicesManager interface, embedding the SDK's
// EmptyArtifactoryServicesManager (whose methods panic if called) and
// overriding only SetProps, which is all artifactoryClient.SetPathProperties
// needs.
type mockServicesManager struct {
	artifactory.EmptyArtifactoryServicesManager
	SetPropsFunc  func(services.PropsParams) (int, error)
	SetPropsCalls []services.PropsParams
}

// SetProps records the call and returns the configured SetPropsFunc result,
// defaulting to a single successful update.
func (m *mockServicesManager) SetProps(params services.PropsParams) (int, error) {
	m.SetPropsCalls = append(m.SetPropsCalls, params)
	if m.SetPropsFunc != nil {
		return m.SetPropsFunc(params)
	}
	return 1, nil
}

// noOpTestUI is a consolidated mock implementation of the packer.Ui interface
// for tests that don't need to capture UI interactions.
type noOpTestUI struct{}

// Ask prompts the user for input (no-op implementation returns empty string).
func (ui *noOpTestUI) Ask(_ string) (string, error) { return "", nil }

// Askf prompts the user for input with formatting (no-op implementation).
func (ui *noOpTestUI) Askf(_ string, _ ...any) (string, error) {
	return "", nil
}

// Error displays an error message (no-op implementation does nothing).
func (ui *noOpTestUI) Error(_ string) {}

// Errorf displays a formatted error message (no-op implementation does nothing).
func (ui *noOpTestUI) Errorf(_ string, _ ...any) {}

// Machine outputs machine-readable data (no-op implementation does nothing).
func (ui *noOpTestUI) Machine(_ string, _ ...string) {}

// Message displays a message (no-op implementation does nothing).
func (ui *noOpTestUI) Message(_ string) {}

// Say displays a message (no-op implementation does nothing).
func (ui *noOpTestUI) Say(_ string) {}

// Sayf displays a formatted message (no-op implementation does nothing).
func (ui *noOpTestUI) Sayf(_ string, _ ...any) {}

// TrackProgress tracks file transfer progress (no-op implementation returns
// the stream unchanged).
func (ui *noOpTestUI) TrackProgress(_ string, _, _ int64, stream io.ReadCloser) io.ReadCloser {
	return stream
}

// capturingTestUI is a consolidated mock implementation of the packer.Ui
// interface for tests that need to capture and verify UI interactions.
type capturingTestUI struct {
	Messages []string
}

// Ask prompts the user for input (capturing implementation returns empty string).
func (ui *capturingTestUI) Ask(_ string) (string, error) { return "", nil }

// Askf prompts the user for input with formatting (capturing implementation).
func (ui *capturingTestUI) Askf(_ string, _ ...any) (string, error) {
	return "", nil
}

// Error displays an error message and records it in the messages slice.
func (ui *capturingTestUI) Error(message string) {
	ui.Messages = append(ui.Messages, "ERROR: "+message)
}

// Errorf displays a formatted error message and records it in the messages slice.
func (ui *capturingTestUI) Errorf(format string, args ...any) {
	ui.Messages = append(ui.Messages, "ERRORF: "+fmt.Sprintf(format, args...))
}

// Machine outputs machine-readable data and records it in the messages slice.
func (ui *capturingTestUI) Machine(t string, _ ...string) {
	ui.Messages = append(ui.Messages, "MACHINE: "+t)
}

// Message displays a message and records it in the messages slice.
func (ui *capturingTestUI) Message(message string) {
	ui.Messages = append(ui.Messages, "MESSAGE: "+message)
}

// Say displays a message and records it in the messages slice.
func (ui *capturingTestUI) Say(message string) {
	ui.Messages = append(ui.Messages, "SAY: "+message)
}

// Sayf displays a formatted message and records it in the messages slice.
func (ui *capturingTestUI) Sayf(format string, args ...any) {
	ui.Messages = append(ui.Messages, "SAYF: "+fmt.Sprintf(format, args...))
}

// TrackProgress tracks file transfer progress and records it in the messages slice.
func (ui *capturingTestUI) TrackProgress(src string, _, _ int64, stream io.ReadCloser) io.ReadCloser {
	ui.Messages = append(ui.Messages, "TRACK_PROGRESS: "+src)
	return stream
}

// createTestOvaFile creates a test OVA file in the specified directory and
// returns the full file path.
func createTestOvaFile(t *testing.T, dir, filename string) string {
	if filename == "" {
		filename = testOvaFile
	}
	filePath := filepath.Join(dir, filename)
	if err := os.WriteFile(filePath, []byte(testOvaContent), 0600); err != nil {
		t.Fatalf("Failed to create test OVA file: %v", err)
	}
	return filePath
}

// createTestOvfFile creates a test OVF file with valid XML content in the
// specified directory and returns the full file path.
func createTestOvfFile(t *testing.T, dir, filename string) string {
	if filename == "" {
		filename = testOvfFile
	}
	filePath := filepath.Join(dir, filename)
	if err := os.WriteFile(filePath, []byte(testOvfContent), 0600); err != nil {
		t.Fatalf("Failed to create test OVF file: %v", err)
	}
	return filePath
}

// createTestFileWithContent creates a test file with custom content in the
// specified directory and returns the full file path.
func createTestFileWithContent(t *testing.T, dir, filename, content string) string {
	filePath := filepath.Join(dir, filename)
	if err := os.WriteFile(filepath.Clean(filePath), []byte(content), 0600); err != nil { //nolint:gosec
		t.Fatalf("Failed to create test file %s: %v", filename, err)
	}
	return filePath
}
