// Copyright (c) Ryan Johnson
// SPDX-License-Identifier: MPL-2.0

package artifactory

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/packer"
)

// allowedOvaOvfExtensions defines extensions used for CanHandle detection (artifact
// “looks like” a VM export). Upload selection uses uploadAllowlist ∪ {.ova}.
var allowedOvaOvfExtensions = map[string]bool{
	".ova":   true,
	".ovf":   true,
	".mf":    true,
	".cert":  true,
	".vmdk":  true,
	".vdi":   true,
	".nvram": true,
	".iso":   true,
}

// ovfEnvelope represents the root element of an OVF descriptor.
type ovfEnvelope struct {
	XMLName xml.Name `xml:"Envelope"`
}

// ovaOvfHandler handles OVA and OVF artifacts.
type ovaOvfHandler struct {
	uploadAllowlist map[string]struct{} // portable OVF ∪ additional_ovf_extensions (never includes .ova)
}

// newOvaOvfHandler returns a handler with only the portable OVF allowlist (for tests).
func newOvaOvfHandler() *ovaOvfHandler {
	return newOvaOvfHandlerWithIncluded(nil)
}

// newOvaOvfHandlerWithIncluded merges additional_ovf_extensions into the portable OVF set.
func newOvaOvfHandlerWithIncluded(additional []string) *ovaOvfHandler {
	return &ovaOvfHandler{uploadAllowlist: mergeUploadAllowlist(additional)}
}

// CanHandle determines if this handler can process the given artifact.
func (h *ovaOvfHandler) CanHandle(artifact packer.Artifact) bool {
	files := artifact.Files()
	if len(files) == 0 {
		return false
	}

	hasOvaOrOvf := false
	hasValidFiles := false

	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file))
		if allowedOvaOvfExtensions[ext] {
			hasValidFiles = true
			if ext == ".ova" || ext == ".ovf" {
				hasOvaOrOvf = true
			}
		}
	}

	return hasValidFiles && hasOvaOrOvf
}

// ProcessArtifact extracts and validates artifacts for upload.
func (h *ovaOvfHandler) ProcessArtifact(artifact packer.Artifact) (*processedArtifact, error) {
	allFiles := artifact.Files()
	if len(allFiles) == 0 {
		return nil, fmt.Errorf("no files found in artifact")
	}

	var validFiles []string
	var ovfFiles []string
	hasOvaOrOvf := false

	for _, file := range allFiles {
		ext := strings.ToLower(filepath.Ext(file))

		if ext == ".ova" {
			validFiles = append(validFiles, file)
			hasOvaOrOvf = true
			continue
		}

		if ext == ".ovf" {
			hasOvaOrOvf = true
		}

		if extensionAllowedForUpload(file, h.uploadAllowlist) {
			validFiles = append(validFiles, file)
			if ext == ".ovf" {
				ovfFiles = append(ovfFiles, file)
			}
		}
	}

	// VMware Fusion/Workstation (and similar flows) export OVA by first materializing
	// an OVF directory, then packaging it into a .ova. Packer's artifact often lists
	// both the final .ova and the leftover OVF bundle files. Upload only the OVA.
	if ovaPaths := filesWithExtension(validFiles, ".ova"); len(ovaPaths) > 0 {
		validFiles = ovaPaths
		ovfFiles = nil
	}

	for _, ovfFile := range ovfFiles {
		if err := h.validateOvfDescriptor(ovfFile); err != nil {
			return nil, fmt.Errorf("invalid OVF descriptor %s: %w", ovfFile, err)
		}
	}

	if len(validFiles) == 0 {
		return nil, fmt.Errorf("no VM files matched the upload allowlist (portable OVF: .ovf, .mf, .vmdk, .cert, .vdi; plus .ova; set additional_ovf_extensions for e.g. nvram or iso)")
	}

	if !hasOvaOrOvf {
		return nil, fmt.Errorf("no OVA or OVF descriptor file found; at least one .ova or .ovf file is required")
	}

	artifactType := "ovf"
	if len(validFiles) == 1 {
		ext := strings.ToLower(filepath.Ext(validFiles[0]))
		switch ext {
		case ".ova":
			artifactType = "ova"
		case ".ovf":
			artifactType = "ovf"
		}
	} else if pathsAllHaveExtension(validFiles, ".ova") {
		artifactType = "ova"
	}

	var fileNames []string
	for _, file := range validFiles {
		fileNames = append(fileNames, filepath.Base(file))
	}

	metadata := map[string]string{
		"type":       artifactType,
		"builder_id": artifact.BuilderId(),
		"files":      strings.Join(fileNames, ","),
	}

	return &processedArtifact{
		Files:    validFiles,
		Metadata: metadata,
	}, nil
}

// validateOvfDescriptor validates that an OVF file contains valid XML structure.
func (h *ovaOvfHandler) validateOvfDescriptor(ovfPath string) error {
	file, err := os.Open(ovfPath)
	if err != nil {
		return fmt.Errorf("cannot open OVF file: %w", err)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	var envelope ovfEnvelope
	decoder := xml.NewDecoder(file)

	if err := decoder.Decode(&envelope); err != nil {
		return fmt.Errorf("invalid XML structure: %w", err)
	}

	return nil
}

// GetHandlerName returns a unique identifier for this handler.
func (h *ovaOvfHandler) GetHandlerName() string {
	return ovaOvfHandlerName
}

// filesWithExtension returns paths in order whose extension matches ext (e.g. ".ova").
func filesWithExtension(paths []string, ext string) []string {
	want := strings.ToLower(ext)
	if !strings.HasPrefix(want, ".") {
		want = "." + want
	}
	var out []string
	for _, p := range paths {
		if strings.ToLower(filepath.Ext(p)) == want {
			out = append(out, p)
		}
	}
	return out
}

func pathsAllHaveExtension(paths []string, ext string) bool {
	if len(paths) == 0 {
		return false
	}
	want := strings.ToLower(ext)
	if !strings.HasPrefix(want, ".") {
		want = "." + want
	}
	for _, p := range paths {
		if strings.ToLower(filepath.Ext(p)) != want {
			return false
		}
	}
	return true
}
