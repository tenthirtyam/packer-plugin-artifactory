// Copyright (c) Ryan Johnson
// SPDX-License-Identifier: MPL-2.0

// Package main implements a Packer post-processor plugin for uploading
// OVA and OVF artifacts to JFrog Artifactory.
package main

import (
	"github.com/hashicorp/packer-plugin-sdk/plugin"
	"github.com/tenthirtyam/packer-plugin-artifactory/post-processor/artifactory"
	"github.com/tenthirtyam/packer-plugin-artifactory/version"
)

// main initializes and runs the Packer plugin.
func main() {
	pps := plugin.NewSet()
	pps.RegisterPostProcessor(plugin.DEFAULT_NAME, new(artifactory.PostProcessor))
	pps.SetVersion(version.PluginVersion)
	err := pps.Run()
	if err != nil {
		panic(err)
	}
}
