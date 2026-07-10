// Copyright (c) Ryan Johnson
// SPDX-License-Identifier: MPL-2.0

package artifactory

import (
	"strings"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/packer"
)

// TestArtifactProcessor_ProcessArtifact tests artifact processing with various
// handler scenarios.
func TestArtifactProcessor_ProcessArtifact(t *testing.T) {
	tests := []struct {
		name        string
		handlers    []artifactHandler
		artifact    packer.Artifact
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful processing with matching handler",
			handlers: []artifactHandler{
				&testHandler{Name: testHandlerName, CanHandleArtifact: true, ShouldError: false},
			},
			artifact:    &testArtifact{FilesValue: []string{testOvaFile}},
			expectError: false,
		},
		{
			name: "no matching handler",
			handlers: []artifactHandler{
				&testHandler{Name: testHandlerName, CanHandleArtifact: false, ShouldError: false},
			},
			artifact:    &testArtifact{FilesValue: []string{testTextFile}},
			expectError: true,
			errorMsg:    "no handler found for artifact type",
		},
		{
			name: "handler processing error",
			handlers: []artifactHandler{
				&testHandler{Name: testHandlerName, CanHandleArtifact: true, ShouldError: true},
			},
			artifact:    &testArtifact{FilesValue: []string{testOvaFile}},
			expectError: true,
			errorMsg:    "mock error",
		},
		{
			name: "first matching handler is used",
			handlers: []artifactHandler{
				&testHandler{Name: firstHandlerName, CanHandleArtifact: true, ShouldError: false},
				&testHandler{Name: secondHandlerName, CanHandleArtifact: true, ShouldError: false},
			},
			artifact:    &testArtifact{FilesValue: []string{testOvaFile}},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := &artifactProcessor{handlers: tt.handlers}

			result, err := processor.processArtifact(tt.artifact)

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
				}
			}
		})
	}
}

// TestArtifactProcessor_RegisterHandler tests handler registration
// functionality.
func TestArtifactProcessor_RegisterHandler(t *testing.T) {
	processor := newArtifactProcessor(nil)
	initialCount := len(processor.handlers)

	newHandler := &testHandler{Name: newHandlerName, CanHandleArtifact: true}
	processor.registerHandler(newHandler)

	if len(processor.handlers) != initialCount+1 {
		t.Errorf("Expected %d handlers, got %d", initialCount+1, len(processor.handlers))
	}

	// Verify the handler was added.
	found := false
	for _, handler := range processor.handlers {
		if handler.GetHandlerName() == newHandlerName {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("New handler was not found in the processor")
	}
}

// TestNewArtifactProcessor tests processor creation and default handler
// registration.
func TestNewArtifactProcessor(t *testing.T) {
	processor := newArtifactProcessor(nil)

	if processor == nil {
		t.Errorf("Expected processor but got nil")
		return
	}

	if len(processor.handlers) == 0 {
		t.Errorf("Expected default handlers to be registered")
	}

	// Verify OVA/OVF handler is registered by default.
	found := false
	for _, handler := range processor.handlers {
		if handler.GetHandlerName() == ovaOvfHandlerName {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected OVA/OVF handler to be registered by default")
	}
}
