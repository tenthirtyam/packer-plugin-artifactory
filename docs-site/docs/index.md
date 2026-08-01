---
icon: octicons/home-16
---

# Packer Plugin for Artifactory

The Packer Plugin for Artifactory is a post-processor (`artifactory`) that can
be used with [HashiCorp Packer][packer] to upload virtual machine artifacts to
JFrog Artifactory.

## Installation

To install this plugin, add this code to your Packer configuration and run
[`packer init`](https://developer.hashicorp.com/packer/docs/commands/init).

```hcl
packer {
  required_plugins {
    artifactory = {
      source  = "github.com/tenthirtyam/artifactory"
      version = ">=0.1.0"
    }
  }
}
```

Alternatively, you can use `packer plugins install` to manage installation of
this plugin.

```sh
packer plugins install github.com/tenthirtyam/artifactory
```

## Supported Platforms

This plugin supports the following platforms:

| Platform           | Type                               | Support     |
|--------------------|------------------------------------|-------------|
| VMware vSphere     | Bare Metal Hypervisor              | Supported   |
| VMware Fusion      | Desktop Hypervisor (macOS)         | Supported   |
| VMware Workstation | Desktop Hypervisor (Windows/Linux) | Supported   |
| Other              | Various                            | Best Effort |

!!! note
    While the plugin may function with other platforms that generate OVA/OVF
    files, official testing and support are currently limited to the VMware by
    Broadcom ecosystem.

## Supported Artifacts

The plugin supports the following artifacts:

- **OVA**: `.ova` — A single compressed archive package containing the
  supporting virtual machine files.
- **OVF**: `.ovf` — An array of virtual machine files supporting an Open
  Virtualization Format package.

## Components

### Post-Processors

- [`artifactory`](post-processors/artifactory.md) — Upload OVA and OVF
  artifacts to JFrog Artifactory.

## Authentication

The plugin supports the following Artifactory authentication methods:

1. Username and Password
2. API Key
3. Access Token

### Environment Variables

```bash
# Option 1: Username and Password
export ARTIFACTORY_USERNAME="your-username"
export ARTIFACTORY_PASSWORD="your-password"

# Option 2: API key
export ARTIFACTORY_API_KEY="your-api-key"

# Option 3: Access Token
export ARTIFACTORY_TOKEN="your-token"
```

## Examples

Examples are available in the [`examples`][examples] directory of the GitHub
repository.

[packer]: https://www.packer.io/
[examples]: https://github.com/tenthirtyam/packer-plugin-artifactory/tree/main/examples/
