// Copyright (c) Ryan Johnson
// SPDX-License-Identifier: MPL-2.0

package artifactory

import (
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/packer"
)

// artifactProcessor orchestrates artifact processing using registered handlers.
type artifactProcessor struct {
	handlers []artifactHandler
}

// newArtifactProcessor creates a new artifactProcessor with default handlers.
func newArtifactProcessor(additionalOvfExtensions []string) *artifactProcessor {
	return &artifactProcessor{
		handlers: []artifactHandler{
			newOvaOvfHandlerWithIncluded(additionalOvfExtensions),
		},
	}
}

// processArtifact processes the given artifact using the first compatible
// handler.
func (ap *artifactProcessor) processArtifact(artifact packer.Artifact) (*processedArtifact, error) {
	for _, handler := range ap.handlers {
		if handler.CanHandle(artifact) {
			return handler.ProcessArtifact(artifact)
		}
	}

	return nil, fmt.Errorf("no handler found for artifact type")
}

// registerHandler adds a new handler to the processor.
func (ap *artifactProcessor) registerHandler(handler artifactHandler) {
	ap.handlers = append(ap.handlers, handler)
}
