// Copyright (c) Ryan Johnson
// SPDX-License-Identifier: MPL-2.0

package artifactory

import (
	"github.com/hashicorp/packer-plugin-sdk/packer"
)

// processedArtifact represents the result of processing an artifact through a
// handler.
type processedArtifact struct {
	Files    []string          // List of file paths to upload.
	Metadata map[string]string // Additional metadata for the artifact.
}

// artifactHandler defines the interface for processing different types of
// artifacts.
type artifactHandler interface {
	// CanHandle determines if this handler can process the given artifact.
	CanHandle(artifact packer.Artifact) bool

	// ProcessArtifact extracts and validates artifacts for upload.
	ProcessArtifact(artifact packer.Artifact) (*processedArtifact, error)

	// GetHandlerName returns a unique identifier for this handler.
	GetHandlerName() string
}
