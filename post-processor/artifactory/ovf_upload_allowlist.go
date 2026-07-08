// Copyright (c) Ryan Johnson
// SPDX-License-Identifier: MPL-2.0

package artifactory

import (
	"path/filepath"
	"strings"
)

// portableOvfUploadExtensions is the default allowlist for multi-file OVF uploads
// (paths in artifact.Files). It favors portable templates: descriptor, manifest,
// certificates, and virtual disks only. NVRAM, ISO attachments, and other extras
// require additional_ovf_extensions.
var portableOvfUploadExtensions = map[string]struct{}{
	".ovf":  {},
	".mf":   {},
	".vmdk": {},
	".cert": {},
	".vdi":  {},
}

func clonePortableOvfUploadExtensions() map[string]struct{} {
	m := make(map[string]struct{}, len(portableOvfUploadExtensions)+4)
	for k := range portableOvfUploadExtensions {
		m[k] = struct{}{}
	}
	return m
}

// mergeUploadAllowlist returns portable ∪ normalizeExtensionList(included).
func mergeUploadAllowlist(included []string) map[string]struct{} {
	m := clonePortableOvfUploadExtensions()
	for k := range normalizeExtensionList(included) {
		m[k] = struct{}{}
	}
	return m
}

// normalizeExtensionList maps lowercased extensions with a leading dot.
func normalizeExtensionList(in []string) map[string]struct{} {
	out := make(map[string]struct{})
	for _, e := range in {
		e = strings.ToLower(strings.TrimSpace(e))
		if e == "" {
			continue
		}
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		out[e] = struct{}{}
	}
	return out
}

func extensionAllowedForUpload(path string, allow map[string]struct{}) bool {
	_, ok := allow[strings.ToLower(filepath.Ext(path))]
	return ok
}
