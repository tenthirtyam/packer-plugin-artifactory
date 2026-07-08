// Copyright (c) Ryan Johnson
// SPDX-License-Identifier: MPL-2.0

package artifactory

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"maps"
	"os"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/auth"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
	rtutils "github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	jfrogConfig "github.com/jfrog/jfrog-client-go/config"
	"github.com/jfrog/jfrog-client-go/utils/io/content"
)

// artifactoryClient provides methods to interact with JFrog Artifactory.
type artifactoryClient struct {
	config          artifactoryConfig
	servicesManager artifactory.ArtifactoryServicesManager
	retryLogic      *retryLogic
	ui              packer.Ui

	warnedOnce map[string]struct{}
}

// artifactMetadata contains metadata about VM artifacts uploaded to Artifactory.
type artifactMetadata struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	BuilderId   string            `json:"builder_id"`
	Files       []string          `json:"files,omitempty"`
	Timestamp   string            `json:"timestamp"`
	Description string            `json:"description"`
	Properties  map[string]string `json:"properties,omitempty"`
}

// newArtifactoryClient creates a new Artifactory client with the given
// configuration.
func newArtifactoryClient(config artifactoryConfig, ui packer.Ui) (*artifactoryClient, error) {
	rtDetails := auth.NewArtifactoryDetails()
	rtDetails.SetUrl(strings.TrimRight(config.URL, "/") + "/")

	if config.APIKey != "" {
		rtDetails.SetApiKey(config.APIKey)
		if config.Username != "" {
			rtDetails.SetUser(config.Username)
		}
	} else if config.AccessToken != "" {
		rtDetails.SetAccessToken(config.AccessToken)
	} else if config.Username != "" && config.Password != "" {
		rtDetails.SetUser(config.Username)
		rtDetails.SetPassword(config.Password)
	} else {
		return nil, fmt.Errorf("no valid authentication method provided (API key, access_token, or username/password)")
	}

	serviceConfig, err := jfrogConfig.NewConfigBuilder().
		SetServiceDetails(rtDetails).
		SetDryRun(false).
		SetThreads(1).
		Build()
	if err != nil {
		return nil, fmt.Errorf("failed to create service artifactoryConfig: %w", err)
	}

	servicesManager, err := artifactory.New(serviceConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create artifactory services manager: %w", err)
	}

	retryLogic := newRetryLogic(config.MaxRetries, config.TimeoutSeconds, ui)

	return &artifactoryClient{
		config:          config,
		servicesManager: servicesManager,
		retryLogic:      retryLogic,
		ui:              ui,
	}, nil
}

// resolvedProperties returns the configured properties, dropping any entry
// whose value renders empty (e.g. an optional "{{ env `VAR` }}" with VAR
// unset). Artifactory's upload API rejects a property with no value
// ("Invalid property format: key= - format should be key=val1,val2,..."),
// so an unset optional property must be omitted rather than sent empty.
func (c *artifactoryClient) resolvedProperties() map[string]string {
	resolved := make(map[string]string, len(c.config.Properties))
	for key, value := range c.config.Properties {
		if value == "" {
			c.warnEmptyProperty(key)
			continue
		}
		resolved[key] = value
	}
	return resolved
}

// warnEmptyProperty notifies the user (once per key) that a property was
// skipped because its value was empty after interpolation.
func (c *artifactoryClient) warnEmptyProperty(key string) {
	c.warnOnce("empty-property:"+key, fmt.Sprintf(
		"Skipping property %q: value is empty after interpolation (e.g. an unset env var)", key))
}

// warnOnce logs message as a [WARN] entry (visible with PACKER_LOG=1) at
// most once per unique key, rather than surfacing it in the build UI.
func (c *artifactoryClient) warnOnce(key, message string) {
	if c.warnedOnce == nil {
		c.warnedOnce = make(map[string]struct{})
	}
	if _, warned := c.warnedOnce[key]; warned {
		return
	}
	c.warnedOnce[key] = struct{}{}

	log.Printf("[WARN] %s", message)
}

// cleanArtifactPath normalizes the configured ArtifactPath by stripping
// path-traversal sequences and leading/trailing slashes.
func (c *artifactoryClient) cleanArtifactPath() string {
	cleanPath := strings.ReplaceAll(c.config.ArtifactPath, "..", "")
	return strings.Trim(cleanPath, "/")
}

// isLicenseRequiredError reports whether err indicates that the requested
// Artifactory REST API is gated behind a Pro/Enterprise license, rather than
// a transient failure or misconfiguration. Artifactory returns this as a 400
// with a message like "This REST API is available only in Artifactory Pro".
func isLicenseRequiredError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Artifactory Pro") || strings.Contains(msg, "valid license")
}

// SetPathProperties sets the resolved properties on the configured
// ArtifactPath folder itself (rather than the files within it), when
// artifact_path_properties is enabled. It is a no-op if the option is
// disabled, if artifact_path is empty (there is no dedicated folder to
// tag), or if there are no resolved properties to set.
func (c *artifactoryClient) SetPathProperties(ctx context.Context) error {
	if !c.config.ArtifactPathProperties {
		return nil
	}

	cleanPath := c.cleanArtifactPath()
	if cleanPath == "" {
		c.warnOnce("artifact_path_properties:no-path",
			"artifact_path_properties is enabled but artifact_path is not set; skipping folder properties")
		return nil
	}

	props := c.resolvedProperties()
	if len(props) == 0 {
		c.warnOnce("artifact_path_properties:no-props",
			"artifact_path_properties is enabled but no properties are configured; skipping folder properties")
		return nil
	}

	propPairs := make([]string, 0, len(props))
	for key, value := range props {
		propPairs = append(propPairs, fmt.Sprintf("%s=%s", key, value))
	}

	writer, err := content.NewContentWriter(content.DefaultKey, true, false)
	if err != nil {
		return fmt.Errorf("failed to create folder properties writer: %w", err)
	}
	writer.Write(rtutils.ResultItem{
		Repo: c.config.Repository,
		Path: ".",
		Name: cleanPath,
		Type: string(rtutils.Folder),
	})
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to write folder properties record: %w", err)
	}
	defer func(name string) {
		_ = os.Remove(name)
	}(writer.GetFilePath())

	reader := content.NewContentReader(writer.GetFilePath(), content.DefaultKey)
	defer func(r *content.ContentReader) {
		_ = r.Close()
	}(reader)

	setPropsOperation := func() error {
		reader.Reset()
		_, err := c.servicesManager.SetProps(services.PropsParams{
			Reader: reader,
			Props:  strings.Join(propPairs, ";"),
		})
		if err != nil {
			if isLicenseRequiredError(err) {
				c.warnOnce("artifact_path_properties:pro-required",
					"artifact_path_properties requires the Set Item Properties REST API, "+
						"which is only available in Artifactory Pro/Enterprise; skipping folder properties")
				return nil
			}
			return fmt.Errorf("failed to set properties on folder %s: %w", cleanPath, err)
		}
		return nil
	}

	operationName := fmt.Sprintf("Set properties on folder %s", cleanPath)
	return c.retryLogic.executeWithRetry(ctx, setPropsOperation, operationName)
}

// UploadMetadata uploads artifact metadata as a JSON file to Artifactory.
func (c *artifactoryClient) UploadMetadata(ctx context.Context, metadata artifactMetadata) error {
	metadataOperation := func() error {
		if metadata.Properties == nil {
			metadata.Properties = make(map[string]string)
		}

		maps.Copy(metadata.Properties, c.resolvedProperties())

		jsonData, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}

		tempFile, err := os.CreateTemp("", fmt.Sprintf("%s-metadata-*.json", metadata.Name))
		if err != nil {
			return fmt.Errorf("failed to create temporary metadata file: %w", err)
		}
		defer func(name string) {
			_ = os.Remove(name)
		}(tempFile.Name())

		if _, err := tempFile.Write(jsonData); err != nil {
			_ = tempFile.Close()
			return fmt.Errorf("failed to write metadata to temporary file: %w", err)
		}
		if err := tempFile.Close(); err != nil {
			return fmt.Errorf("failed to close metadata file: %w", err)
		}

		metadataFilename := fmt.Sprintf("%s-metadata.json", metadata.Name)
		return c.UploadFile(ctx, tempFile.Name(), metadataFilename)
	}

	operationName := fmt.Sprintf("Upload metadata for %s", metadata.Name)
	return c.retryLogic.executeWithRetry(ctx, metadataOperation, operationName)
}

// UploadFile uploads a single file to Artifactory at the configured repository
// and path.
func (c *artifactoryClient) UploadFile(ctx context.Context, filePath, filename string) error {
	artifactPath := c.buildArtifactPath(filename)
	targetPath := fmt.Sprintf("%s/%s", c.config.Repository, artifactPath)

	if !c.config.shouldOverwrite() {
		exists, err := c.checkArtifactExists(targetPath)
		if err != nil {
			return fmt.Errorf("failed to check if artifact exists: %w", err)
		}
		if exists {
			return fmt.Errorf("artifact already exists at %s. set overwrite=true to replace", targetPath)
		}
	}

	uploadOperation := func() error {
		params := services.NewUploadParams()
		params.Pattern = filePath
		params.Target = targetPath
		params.Flat = true

		allProperties := c.resolvedProperties()

		if len(allProperties) > 0 {
			var buildProps []string
			for key, value := range allProperties {
				buildProps = append(buildProps, fmt.Sprintf("%s=%s", key, value))
			}
			params.BuildProps = strings.Join(buildProps, ";")
		}

		uploadServiceOptions := artifactory.UploadServiceOptions{
			FailFast: false,
		}

		successCount, failedCount, err := c.servicesManager.UploadFiles(uploadServiceOptions, params)
		if err != nil {
			return fmt.Errorf("upload failed: %w", err)
		}

		if failedCount > 0 {
			return fmt.Errorf("upload failed: %d files failed to upload", failedCount)
		}

		if successCount == 0 {
			return fmt.Errorf("no files were uploaded")
		}

		return nil
	}

	operationName := fmt.Sprintf("Upload file %s", filename)
	return c.retryLogic.executeWithRetry(ctx, uploadOperation, operationName)
}

// checkArtifactExists checks if an artifact already exists at the given path.
func (c *artifactoryClient) checkArtifactExists(artifactPath string) (bool, error) {
	searchParams := services.NewSearchParams()
	searchParams.Pattern = artifactPath
	searchParams.Limit = 1

	reader, err := c.servicesManager.SearchFiles(searchParams)
	if err != nil {
		return false, fmt.Errorf("search files failed: %w", err)
	}
	defer func(reader *content.ContentReader) {
		_ = reader.Close()
	}(reader)

	// Use a proper struct to unmarshal the search result
	var result struct {
		Repo string `json:"repo,omitempty"`
		Path string `json:"path,omitempty"`
		Name string `json:"name,omitempty"`
	}

	err = reader.NextRecord(&result)
	if err != nil {
		// "Empty" error means no results found, which means artifact doesn't
		// exist in Artifactory.
		if err.Error() == "Empty" {
			return false, nil
		}
		return false, fmt.Errorf("failed to read search result: %w", err)
	}

	// If a record is read the artifact exists.
	return result.Name != "", nil
}

// buildArtifactPath creates a consistent artifact path using the configured
// ArtifactPath.
func (c *artifactoryClient) buildArtifactPath(filename string) string {
	if cleanPath := c.cleanArtifactPath(); cleanPath != "" {
		return fmt.Sprintf("%s/%s", cleanPath, filename)
	}
	return filename
}
