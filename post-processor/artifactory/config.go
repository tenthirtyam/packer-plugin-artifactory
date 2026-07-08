// Copyright (c) Ryan Johnson
// SPDX-License-Identifier: MPL-2.0

package artifactory

import (
	"fmt"
	"os"
)

//go:generate packer-sdc struct-markdown
//go:generate packer-sdc mapstructure-to-hcl2 -type artifactoryConfig

const (
	// defaultRetries is the default number of retry attempts for failed
	// uploads.
	defaultRetries = 3
	// defaultTimeoutSeconds is the default request timeout in seconds.
	defaultTimeoutSeconds = 30
)

// artifactoryConfig contains configuration for the Artifactory post-processor.
type artifactoryConfig struct {
	// The base URL of the Artifactory instance (e.g., https://packages.example.com/artifactory).
	URL string `mapstructure:"url" required:"true"`
	// An API key to authenticate to the Artifactory instance.
	APIKey string `mapstructure:"api_key"`
	// An access token to authenticate to the Artifactory instance.
	AccessToken string `mapstructure:"access_token"`
	// A username to authenticate to the Artifactory instance.
	Username string `mapstructure:"username"`
	// A password to authenticate to the Artifactory instance.
	Password string `mapstructure:"password"`
	// The name of the repository to upload the artifact to.
	Repository string `mapstructure:"repository" required:"true"`
	// The name of the artifact being uploaded.
	ArtifactName string `mapstructure:"artifact_name" required:"true"`
	// The path within the repository where the artifact will be stored.
	ArtifactPath string `mapstructure:"artifact_path"`
	// Whether to apply the `properties` to the path of the artifact in addition
	// to each uploaded file. (default: `false`)
	//
	// ~> **Note:** Ignored if `properties` is empty.
	//
	// ~> **Note:** Ignored if the Artifactory instance does not support setting
	// properties on the artifact's repository path. Uploaded files are unaffected
	// by this restriction.
	ArtifactPathProperties bool `mapstructure:"artifact_path_properties"`
	// The number of retry attempts for failed uploads. (default: `3`)
	MaxRetries int `mapstructure:"max_retries"`
	// The request timeout in seconds. (default: `30`)
	TimeoutSeconds int `mapstructure:"timeout_seconds"`
	// Whether to overwrite existing artifacts. (default: `false`)
	//
	// ~> **Note:** The upload will fail if the artifact already exists and the
	// overwrite flag is set to `false`.
	Overwrite *bool `mapstructure:"overwrite"`
	// Custom properties to attach to artifacts.
	Properties map[string]string `mapstructure:"properties"`
	// Additional file extensions to upload alongside an OVF package, beyond
	// the default set (`.ovf`, `.mf`, `.vmdk`, `.cert`, `.vdi`). Extensions may be
	// given with or without a leading dot (_e.g._ `nvram` or `.nvram`).
	AdditionalOvfExtensions []string `mapstructure:"additional_ovf_extensions"`
}

// Validate checks the configuration for required fields and sets defaults.
func (c *artifactoryConfig) Validate() error {
	var errs []error

	if c.URL == "" {
		errs = append(errs, fmt.Errorf("url is required"))
	}

	if c.Repository == "" {
		errs = append(errs, fmt.Errorf("repository is required"))
	}

	if c.ArtifactName == "" {
		errs = append(errs, fmt.Errorf("artifact_name is required"))
	}

	if c.APIKey == "" {
		if envAPIKey := os.Getenv("ARTIFACTORY_API_KEY"); envAPIKey != "" {
			c.APIKey = envAPIKey
		}
	}

	if c.AccessToken == "" {
		if envToken := os.Getenv("ARTIFACTORY_TOKEN"); envToken != "" {
			c.AccessToken = envToken
		}
	}

	if c.Username == "" {
		if envUsername := os.Getenv("ARTIFACTORY_USERNAME"); envUsername != "" {
			c.Username = envUsername
		}
	}

	if c.Password == "" {
		if envPassword := os.Getenv("ARTIFACTORY_PASSWORD"); envPassword != "" {
			c.Password = envPassword
		}
	}

	authMethods := 0
	var providedMethods []string

	if c.APIKey != "" {
		authMethods++
		providedMethods = append(providedMethods, "api_key")
	}

	if c.AccessToken != "" {
		authMethods++
		providedMethods = append(providedMethods, "access_token")
	}

	if c.Username != "" && c.Password != "" {
		authMethods++
		providedMethods = append(providedMethods, "username/password")
	}

	if authMethods == 0 {
		errs = append(errs, fmt.Errorf("authentication is required: provide exactly one of api_key, access_token, or username/password via artifactoryConfig or environment variables"))
	} else if authMethods > 1 {
		errs = append(errs, fmt.Errorf("multiple authentication methods provided (%v): use only one authentication method", providedMethods))
	}

	if (c.Username != "" && c.Password == "") || (c.Username == "" && c.Password != "") {
		errs = append(errs, fmt.Errorf("username and password must be provided together"))
	}

	if c.MaxRetries == 0 {
		c.MaxRetries = defaultRetries
	}

	if c.Overwrite == nil {
		overwrite := false
		c.Overwrite = &overwrite
	}

	if c.TimeoutSeconds == 0 {
		c.TimeoutSeconds = defaultTimeoutSeconds
	}

	if len(errs) > 0 {
		return fmt.Errorf("configuration validation failed: %v", errs)
	}

	return nil
}

// shouldOverwrite returns the overwrite setting, defaulting to false if not
// set.
func (c *artifactoryConfig) shouldOverwrite() bool {
	if c.Overwrite == nil {
		return false
	}
	return *c.Overwrite
}
