<!-- markdownlint-disable first-line-h1 no-inline-html -->

<img src=".github/mascot-300px.png" alt="Packer Plugin for Artifactory">

# Packer Plugin for Artifactory

The Packer Plugin for Artifactory is a post-processor (`artifactory`) that can be used with
HashiCorp Packer to upload virtual machine artifacts to JFrog Artifactory.

## Installation

To install this plugin add this code into your Packer configuration and run [`packer init`](/packer/docs/commands/init).

```hcl
packer {
  required_plugins {
    artifactory = {
      source  = "github.com/tenthirtyam/artifactory"
      version = ">=0.1.1"
    }
  }
}
```

Alternatively, you can use `packer plugins install` to manage installation of this plugin.

```sh
packer plugins install github.com/tenthirtyam/artifactory@v0.1.1
```

## Supported Platforms

This plugin supports the following platforms:

| Platform           | Type                               | Support     |
|--------------------|------------------------------------|-------------|
| VMware vSphere     | Bare Metal Hypervisor              | Supported   |
| VMware Fusion      | Desktop Hypervisor (macOS)         | Supported   |
| VMware Workstation | Desktop Hypervisor (Windows/Linux) | Supported   |
| Other              | Various                            | Best Effort |

> [!NOTE]
> While the plugin may function with other platforms that generate OVA/OVF files, testing and
> support are limited to the VMware by Broadcom ecosystem.

## Supported Artifacts

The plugin supports the following artifacts:

- **OVA**: `.ova` - A single compressed archive package containing all the supporting virtual
  machine files.
- **OVF**: `.ovf` - An array of virtual machine files supporting an Open Virtualization Format
  package.

## Documentation

- Project documentation: [Packer Plugin for Artifactory][documentation]
- HashiCorp Integrations registry: [tenthirtyam/artifactory][hashicorp-docs]

## Examples

Refer to the [`examples/`][examples] folder for usage scenarios.

### Trademark Notice

All product names and brands referenced in this project are the property of their respective owners.
Use of these names does not imply any affiliation with or endorsement by those owners.

## License

Copyright © 2026 [Ryan Johnson][author].

Licensed under the [Mozilla Public License 2.0][license].

See [NOTICE][notice] for additional attribution.

## Sponsor

[![Sponsor](https://img.shields.io/badge/Sponsor-EA4AAA?style=for-the-badge&logo=githubsponsors&logoColor=white)][sponsor]&nbsp;&nbsp;
[![Buy Me a Coffee](https://img.shields.io/badge/Buy%20Me%20a%20Coffee-FFDD00?style=for-the-badge&logo=buymeacoffee&logoColor=white)][buy-me-a-coffee]

[author]: https://github.com/tenthirtyam
[packer]: https://www.packer.io/
[documentation]: https://tenthirtyam.github.io/packer-plugin-artifactory/latest/
[hashicorp-docs]: https://developer.hashicorp.com/packer/integrations/tenthirtyam/artifactory
[examples]: https://github.com/tenthirtyam/packer-plugin-artifactory/tree/main/examples/
[discussions]: https://github.com/tenthirtyam/packer-plugin-artifactory/discussions
[license]: LICENSE
[notice]: NOTICE
[sponsor]: https://github.com/sponsors/tenthirtyam
[buy-me-a-coffee]: https://www.buymeacoffee.com/tenthirtyam
