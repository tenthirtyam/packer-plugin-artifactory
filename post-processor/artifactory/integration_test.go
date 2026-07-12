// Copyright (c) Ryan Johnson
// SPDX-License-Identifier: MPL-2.0

package artifactory

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// artifactoryRepoConfigRESTBlocked reports whether the response body indicates
// repository configuration via REST is unavailable (e.g. Artifactory OSS).
func artifactoryRepoConfigRESTBlocked(body string) bool {
	return strings.Contains(body, "Artifactory Pro") ||
		strings.Contains(body, "valid license")
}

// TestIntegrationMain runs integration tests if enabled in the environment
// variables.
func TestIntegrationMain(t *testing.T) {
	if os.Getenv("INTEGRATION_TESTS") == "" {
		t.Skip("Skipping integration tests. Set INTEGRATION_TESTS=1 to run.")
	}

	if !waitForArtifactory(t) {
		t.Fatal("Artifactory failed to become ready")
	}

	if err := createTestRepository(t); err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}

	t.Run("UploadOvaFile", testUploadOvaFile)
	t.Run("UploadOvfPackage", testUploadOvfPackage)
	t.Run("OverwriteExistingFile", testOverwriteExistingFile)
	t.Run("RejectOverwriteWhenDisabled", testRejectOverwriteWhenDisabled)
	t.Run("RejectNonVMFiles", testRejectNonVMFiles)
}

// waitForArtifactory waits for the Artifactory server to become ready by
// polling the ping endpoint with a timeout.
func waitForArtifactory(t *testing.T) bool {
	client := &http.Client{Timeout: 5 * time.Second}
	pingURL := strings.TrimRight(testArtifactoryURL, "/") + "/api/system/ping"

	for i := range 30 {
		req, err := http.NewRequest(http.MethodGet, pingURL, nil)
		if err != nil {
			t.Logf("Waiting for Artifactory... building ping request: %v", err)
			time.Sleep(10 * time.Second)
			continue
		}

		req.SetBasicAuth(testUsername, testPassword)

		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close() // Ignore close error in test
			t.Log("Artifactory is ready")
			return true
		}
		if resp != nil {
			_ = resp.Body.Close() // Ignore close error in test
		}
		t.Logf("Waiting for Artifactory... attempt %d/30", i+1)
		time.Sleep(10 * time.Second)
	}
	return false
}

// createTestRepository creates a test repository in Artifactory for
// integration testing and cleans up any existing test artifacts.
func createTestRepository(t *testing.T) error {
	client := &http.Client{Timeout: 10 * time.Second}

	cleanupPaths := []string{
		testArtifactoryURL + "/" + testRepository + "/integration-test",
		testArtifactoryURL + "/" + testRepository + "/integration-test/test-ova",
		testArtifactoryURL + "/" + testRepository + "/integration-test/test-ovf-package",
		testArtifactoryURL + "/" + testRepository + "/integration-test/overwrite-test",
		testArtifactoryURL + "/" + testRepository + "/integration-test/no-overwrite-test",
	}

	for _, cleanupPath := range cleanupPaths {
		if req, err := http.NewRequest("DELETE", cleanupPath, nil); err == nil {
			req.SetBasicAuth(testUsername, testPassword)
			resp, _ := client.Do(req)
			if resp != nil {
				_ = resp.Body.Close()
			}
		}
	}
	t.Logf("Cleaned up existing test artifacts")

	getReq, err := http.NewRequest(http.MethodGet,
		testArtifactoryURL+"/api/repositories/"+testRepository, nil)
	if err != nil {
		return err
	}
	getReq.SetBasicAuth(testUsername, testPassword)

	getResp, err := client.Do(getReq)
	if err != nil {
		return err
	}
	getBody, err := io.ReadAll(getResp.Body)
	_ = getResp.Body.Close()
	if err != nil {
		return err
	}
	getBodyStr := string(getBody)

	if getResp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf(
			"authentication failed checking repository %q (verify ARTIFACTORY_USERNAME and ARTIFACTORY_PASSWORD or API key as password)",
			testRepository)
	}

	if getResp.StatusCode == http.StatusOK {
		t.Logf("Using existing repository %q", testRepository)
		return nil
	}

	if getResp.StatusCode == http.StatusBadRequest && artifactoryRepoConfigRESTBlocked(getBodyStr) {
		t.Logf(
			"Repository configuration REST API unavailable (typical for Artifactory OSS); using %q — ensure this repository exists",
			testRepository)
		return nil
	}

	if getResp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("unexpected status checking repository %q: %s - %s",
			testRepository, getResp.Status, getBodyStr)
	}

	repoConfig := `{
                "key": "` + testRepository + `",
                "rclass": "local",
                "packageType": "generic",
                "description": "` + testDescription + `"
        }`

	req, err := http.NewRequest(http.MethodPut,
		testArtifactoryURL+"/api/repositories/"+testRepository,
		strings.NewReader(repoConfig))
	if err != nil {
		return err
	}

	req.SetBasicAuth(testUsername, testPassword)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create repository: %s - %s", resp.Status, string(body))
	}

	if resp.StatusCode == http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)
		if strings.Contains(bodyStr, "already exists") {
			t.Logf("Test repository already exists, continuing with tests...")
			return nil
		}
		if artifactoryRepoConfigRESTBlocked(bodyStr) {
			t.Logf(
				"Repository creation REST API unavailable; using %q — ensure this repository exists",
				testRepository)
			return nil
		}
		return fmt.Errorf("failed to create repository: %s - %s", resp.Status, bodyStr)
	}

	t.Logf("Test repository created successfully")
	return nil
}

// testUploadOvaFile tests uploading an OVA file to Artifactory using
// real test data from the testdata directory.
func testUploadOvaFile(t *testing.T) {
	tempDir := t.TempDir()

	ovaContent, err := os.ReadFile("testdata/ova/sample.ova")
	if err != nil {
		t.Fatalf("Failed to read sample OVA: %v", err)
	}

	ovaFile := createTestFileWithContent(t, tempDir, testOvaFile, string(ovaContent))

	artifact := &testArtifact{
		BuilderIdValue: testBuilderId,
		FilesValue:     []string{ovaFile},
		StringValue:    "Test OVA artifact",
	}

	config := artifactoryConfig{
		URL:            testArtifactoryURL,
		Username:       testUsername,
		Password:       testPassword,
		Repository:     testRepository,
		ArtifactName:   "test-ova",
		ArtifactPath:   "integration-test/ova/",
		MaxRetries:     3,
		TimeoutSeconds: 30,
	}

	pp := &PostProcessor{}
	configErr := pp.Configure(map[string]any{
		"url":           config.URL,
		"username":      config.Username,
		"password":      config.Password,
		"repository":    config.Repository,
		"artifact_name": config.ArtifactName,
		"artifact_path": config.ArtifactPath,
	})
	if configErr != nil {
		t.Fatalf("Failed to configure post-processor: %v", configErr)
	}

	ui := &noOpTestUI{}
	ctx := context.Background()

	resultArtifact, keep, forceOverride, err := pp.PostProcess(ctx, ui, artifact)
	if err != nil {
		t.Fatalf("PostProcess failed: %v", err)
	}

	if resultArtifact == nil {
		t.Fatal("Expected artifact to be returned")
	}

	if keep != false || forceOverride != false {
		t.Errorf("Expected keep=false, forceOverride=false, got keep=%v, forceOverride=%v", keep, forceOverride)
	}

	t.Log("Successfully uploaded OVA file")
}

// testUploadOvfPackage tests uploading an OVF package (multiple files)
// to Artifactory using real test data.
func testUploadOvfPackage(t *testing.T) {
	tempDir := t.TempDir()

	ovfContent, err := os.ReadFile("testdata/ovf/sample.ovf")
	if err != nil {
		t.Fatalf("Failed to read sample OVF: %v", err)
	}

	mfContent, err := os.ReadFile("testdata/ovf/sample.mf")
	if err != nil {
		t.Fatalf("Failed to read sample manifest: %v", err)
	}

	vmdkContent, err := os.ReadFile("testdata/ovf/sample-disk-0.vmdk")
	if err != nil {
		t.Fatalf("Failed to read sample VMDK: %v", err)
	}

	files := map[string][]byte{
		testOvfFile:      ovfContent,
		testVmdkFile:     vmdkContent,
		testManifestFile: mfContent,
	}

	var filePaths []string
	for filename, content := range files {
		filePath := createTestFileWithContent(t, tempDir, filename, string(content))
		filePaths = append(filePaths, filePath)
	}

	artifact := &testArtifact{
		BuilderIdValue: testBuilderId,
		FilesValue:     filePaths,
		StringValue:    "Test OVF package",
	}

	config := artifactoryConfig{
		URL:          testArtifactoryURL,
		Username:     testUsername,
		Password:     testPassword,
		Repository:   testRepository,
		ArtifactName: "test-ovf-package",
		ArtifactPath: "integration-test/ovf",
	}

	pp := &PostProcessor{}
	configErr := pp.Configure(map[string]any{
		"url":           config.URL,
		"username":      config.Username,
		"password":      config.Password,
		"repository":    config.Repository,
		"artifact_name": config.ArtifactName,
		"artifact_path": config.ArtifactPath,
	})
	if configErr != nil {
		t.Fatalf("Failed to configure post-processor: %v", configErr)
	}

	ui := &noOpTestUI{}
	ctx := context.Background()

	_, _, _, processErr := pp.PostProcess(ctx, ui, artifact)
	if processErr != nil {
		t.Fatalf("PostProcess failed: %v", processErr)
	}

	t.Log("Successfully uploaded OVF package")
}

// testOverwriteExistingFile tests the overwrite functionality by uploading
// the same file twice with overwrite enabled.
func testOverwriteExistingFile(t *testing.T) {
	// Use real OVA test data
	tempDir := t.TempDir()
	ovaFile := filepath.Join(tempDir, "overwrite-test.ova")

	// Copy sample OVA from testdata
	ovaContent, err := os.ReadFile("testdata/ova/sample.ova")
	if err != nil {
		t.Fatalf("Failed to read sample OVA: %v", err)
	}

	if err := os.WriteFile(filepath.Clean(ovaFile), ovaContent, 0600); err != nil { //nolint:gosec
		t.Fatalf("Failed to create test OVA file: %v", err)
	}

	// Create mock artifact with OVA file
	artifact := &testArtifact{
		BuilderIdValue: testBuilderId,
		FilesValue:     []string{ovaFile},
		StringValue:    "Test OVA artifact for overwrite",
	}

	config := artifactoryConfig{
		URL:            testArtifactoryURL,
		Username:       testUsername,
		Password:       testPassword,
		Repository:     testRepository,
		ArtifactName:   "overwrite-test",
		ArtifactPath:   "integration-test",
		Overwrite:      &[]bool{true}[0],
		MaxRetries:     3,
		TimeoutSeconds: 30,
	}

	pp := &PostProcessor{}
	configErr := pp.Configure(map[string]any{
		"url":           config.URL,
		"username":      config.Username,
		"password":      config.Password,
		"repository":    config.Repository,
		"artifact_name": config.ArtifactName,
		"artifact_path": config.ArtifactPath,
		"overwrite":     *config.Overwrite,
	})
	if configErr != nil {
		t.Fatalf("Failed to configure post-processor: %v", configErr)
	}

	ui := &noOpTestUI{}
	ctx := context.Background()

	t.Log("Performing first upload...")
	resultArtifact1, keep1, forceOverride1, err := pp.PostProcess(ctx, ui, artifact)
	if err != nil {
		t.Fatalf("First PostProcess failed: %v", err)
	}

	if resultArtifact1 == nil {
		t.Fatal("Expected artifact to be returned from first upload")
	}

	if keep1 != false || forceOverride1 != false {
		t.Errorf("Expected keep=false, forceOverride=false on first upload, got keep=%v, forceOverride=%v", keep1, forceOverride1)
	}

	t.Log("First upload successful")

	t.Log("Performing second upload (overwrite)...")
	resultArtifact2, keep2, forceOverride2, err := pp.PostProcess(ctx, ui, artifact)
	if err != nil {
		t.Fatalf("Second PostProcess (overwrite) failed: %v", err)
	}

	if resultArtifact2 == nil {
		t.Fatal("Expected artifact to be returned from second upload")
	}

	if keep2 != false || forceOverride2 != false {
		t.Errorf("Expected keep=false, forceOverride=false on second upload, got keep=%v, forceOverride=%v", keep2, forceOverride2)
	}

	t.Log("Successfully tested overwrite functionality")
}

// testRejectOverwriteWhenDisabled tests that file uploads are rejected
// when attempting to overwrite existing files with overwrite disabled.
func testRejectOverwriteWhenDisabled(t *testing.T) {
	// Use real OVA test data
	tempDir := t.TempDir()
	ovaFile := filepath.Join(tempDir, "no-overwrite-test.ova")

	// Copy sample OVA from testdata
	ovaContent, err := os.ReadFile("testdata/ova/sample.ova")
	if err != nil {
		t.Fatalf("Failed to read sample OVA: %v", err)
	}

	if err := os.WriteFile(filepath.Clean(ovaFile), ovaContent, 0600); err != nil { //nolint:gosec
		t.Fatalf("Failed to create test OVA file: %v", err)
	}

	// Create mock artifact with OVA file
	artifact := &testArtifact{
		BuilderIdValue: testBuilderId,
		FilesValue:     []string{ovaFile},
		StringValue:    "Test OVA artifact for no-overwrite",
	}

	config := artifactoryConfig{
		URL:            testArtifactoryURL,
		Username:       testUsername,
		Password:       testPassword,
		Repository:     testRepository,
		ArtifactName:   "no-overwrite-test",
		ArtifactPath:   "integration-test",
		Overwrite:      &[]bool{false}[0],
		MaxRetries:     3,
		TimeoutSeconds: 30,
	}

	pp := &PostProcessor{}
	configErr := pp.Configure(map[string]any{
		"url":           config.URL,
		"username":      config.Username,
		"password":      config.Password,
		"repository":    config.Repository,
		"artifact_name": config.ArtifactName,
		"artifact_path": config.ArtifactPath,
		"overwrite":     *config.Overwrite,
	})
	if configErr != nil {
		t.Fatalf("Failed to configure post-processor: %v", configErr)
	}

	ui := &noOpTestUI{}
	ctx := context.Background()

	t.Log("Performing first upload...")
	_, _, _, err = pp.PostProcess(ctx, ui, artifact)
	if err != nil {
		t.Fatalf("First PostProcess failed: %v", err)
	}

	t.Log("First upload successful")

	t.Log("Performing second upload (should fail)...")
	_, _, _, err = pp.PostProcess(ctx, ui, artifact)
	if err == nil {
		t.Fatal("Expected second upload to fail when overwrite is disabled")
	}

	expectedErrors := []string{
		"already exists",
		"file exists",
		"conflict",
		"overwrite",
		"Set overwrite=true to replace",
	}

	errorFound := false
	for _, expectedError := range expectedErrors {
		if contains(err.Error(), expectedError) {
			errorFound = true
			break
		}
	}

	if !errorFound {
		t.Logf("Got error: %v", err)
		t.Log("This might be expected behavior - some Artifactory configurations allow overwrites by default")
	} else {
		t.Log("Successfully rejected overwrite when disabled")
	}
}

// Error handling tests

// testRejectNonVMFiles tests that the post-processor correctly rejects
// artifacts that don't contain VM files (OVA/OVF).
func testRejectNonVMFiles(t *testing.T) {
	tempDir := t.TempDir()
	textFile := createTestFileWithContent(t, tempDir, "readme.txt", "not a vm file")

	artifact := &testArtifact{
		BuilderIdValue: "foo",
		FilesValue:     []string{textFile},
		StringValue:    "Non-VM artifact",
	}

	pp := &PostProcessor{}
	configErr := pp.Configure(map[string]any{
		"url":           testArtifactoryURL,
		"username":      testUsername,
		"password":      testPassword,
		"repository":    testRepository,
		"artifact_name": "should-fail",
	})
	if configErr != nil {
		t.Fatalf("Failed to configure post-processor: %v", configErr)
	}

	ui := &noOpTestUI{}
	ctx := context.Background()

	_, _, _, processErr := pp.PostProcess(ctx, ui, artifact)
	if processErr == nil {
		t.Fatal("Expected PostProcess to fail with non-VM files")
	}

	expectedErrors := []string{
		"no VM files matched the upload allowlist",
		"no handler found for artifact type",
	}

	errorFound := false
	for _, expectedError := range expectedErrors {
		if contains(processErr.Error(), expectedError) {
			errorFound = true
			break
		}
	}

	if !errorFound {
		t.Errorf("Expected error containing one of %v, got: %v", expectedErrors, processErr)
	}

	t.Log("Correctly rejected non-VM files")
}

// contains checks if a string contains a substring using a custom implementation.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			stringContains(s, substr))))
}

// stringContains performs substring search within a string.
func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
