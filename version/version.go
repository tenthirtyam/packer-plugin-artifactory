// Copyright (c) Ryan Johnson
// SPDX-License-Identifier: MPL-2.0

// Package version defines the plugin version information.
package version

import sdkversion "github.com/hashicorp/packer-plugin-sdk/version"

var (
	// number is the main version number.
	number = "0.1.0"
	// channel is the release channel for the version.
	channel = ""
	// metadata is build metadata for the version.
	metadata = ""
	// PluginVersion is the complete version information for the plugin.
	PluginVersion = sdkversion.NewPluginVersion(number, channel, metadata)
)
