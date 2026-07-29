# Packer Plugin for Artifactory

The Packer Plugin for Artifactory is a post-processor (`artifactory`) that can be used with [HashiCorp Packer][packer]
to upload virtual machine artifacts to JFrog Artifactory.

## Installation

To install this plugin add this code into your Packer configuration and run
[`packer init`](/packer/docs/commands/init).

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
packer plugins install github.com/tenthirtyam/artifactory@v0.1.0
```

## Supported Platforms

This plugin supports the following platforms:

| Platform           | Type                               | Support     |
|--------------------|------------------------------------|-------------|
| VMware vSphere     | Bare Metal Hypervisor              | Supported   |
| VMware Fusion      | Desktop Hypervisor (macOS)         | Supported   |
| VMware Workstation | Desktop Hypervisor (Windows/Linux) | Supported   |
| Other              | Various                            | Best Effort |

~> **Note:** While the plugin may function with other platforms that generate OVA/OVF files,
official testing and support are currently limited to the VMware by Broadcom ecosystem.

## Supported Artifacts

The plugin supports the following artifacts:

- **OVA**: `.ova` - A single compressed archive package containing the
  supporting virtual machine files.
- **OVF**: `.ovf` - An array of virtual machine files supporting an Open
  Virtualization Format package.

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

## Example

```hcl
build {
  # ...
  post-processor "artifactory" {
    url                      = var.artifactory_url
    username                 = var.artifactory_username
    password                 = var.artifactory_password
    repository               = var.artifactory_repository
    artifact_name            = local.artifact_name
    artifact_path            = local.artifact_path
    artifact_path_properties = true
    properties = {
      "version.number"  = "1.0.0"
      "build.number"    = "12345678"
      "release.channel" = "stable"
      "os.family"       = "linux"
      "os.vendor"       = "debian"
      "os.version"      = "13.6.0"
      "os.arch"         = "amd64"
    }
  }
}
```

## Reference

For detailed configuration options, see the [documentation][documentation].

## Examples

Examples are available in the [`examples`][examples] directory of the GitHub
repository.

[packer]: https://www.packer.io/
[documentation]: https://developer.hashicorp.com/packer/integrations/tenthirtyam/artifactory
[examples]: https://github.com/tenthirtyam/packer-plugin-artifactory/tree/main/examples/
