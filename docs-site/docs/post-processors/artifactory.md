---
icon: material/package-variant
title: Artifactory
---

# Artifactory Post-Processor

Type: `artifactory`

Artifact BuilderId: `packer.post-processor.artifactory`

The Packer Plugin for Artifactory is a post-processor (`artifactory`) that can
be used with [HashiCorp Packer][packer] to upload virtual machine artifacts to
JFrog Artifactory.

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

- **OVA**: `.ova` — A single compressed archive package containing all the
  supporting virtual machine files.
- **OVF**: `.ovf` — An array of virtual machine files supporting an Open
  Virtualization Format package.

## Configuration Reference

The following configuration options are available for the post-processor.

### Required

- `url` (string) — The base URL of the Artifactory instance (e.g.,
  `https://packages.example.com/artifactory`).

- `repository` (string) — The name of the repository to upload the artifact to.

- `artifact_name` (string) — The name of the artifact being uploaded.

### Optional

- `api_key` (string) — An API key to authenticate to the Artifactory instance.
  Environment variable: `ARTIFACTORY_API_KEY`.

- `access_token` (string) — An access token to authenticate to the Artifactory
  instance. Environment variable: `ARTIFACTORY_TOKEN`.

- `username` (string) — A username to authenticate to the Artifactory instance.
  Environment variable: `ARTIFACTORY_USERNAME`.

- `password` (string) — A password to authenticate to the Artifactory instance.
  Environment variable: `ARTIFACTORY_PASSWORD`.

- `artifact_path` (string) — The path within the repository where the artifact
  will be stored.

- `artifact_path_properties` (bool) — Whether to apply the `properties` to the
  path of the artifact in addition to each uploaded file. (default: `false`)

    !!! note
        Ignored if `properties` is empty.

    !!! note
        Ignored if the Artifactory instance does not support setting properties
        on the artifact's repository path. Uploaded files are unaffected by
        this restriction.

- `max_retries` (int) — The number of retry attempts for failed uploads.
  (default: `3`)

- `timeout_seconds` (int) — The request timeout in seconds. (default: `30`)

- `overwrite` (*bool) — Whether to overwrite existing artifacts.
  (default: `false`)

    !!! note
        The upload will fail if the artifact already exists and the overwrite
        flag is set to `false`.

- `properties` (map[string]string) — Custom properties to attach to artifacts.

- `additional_ovf_extensions` ([]string) — Additional file extensions to upload
  alongside an OVF package, beyond the default set (`.ovf`, `.mf`, `.vmdk`,
  `.cert`, `.vdi`). Extensions may be given with or without a leading dot
  (e.g. `nvram` or `.nvram`).

!!! tip
    Provide exactly one authentication method: API key, access token, or
    username and password (via configuration or environment variables).

## Example Usage

!!! note
    Complete examples for VMware vSphere and VMware desktop hypervisors are
    available in the repository [`examples`][examples] directory. The files
    below are included from those examples.

### Basic Authentication

???+ note "Post-Processor Configuration"

    ```hcl
    --8<-- "examples/basic/build.pkr.hcl"
    ```

??? note "Additional Configuration"

    Versions:

    ```hcl
    --8<-- "examples/basic/versions.pkr.hcl"
    ```

    Variables:

    ```hcl
    --8<-- "examples/basic/variables.pkr.hcl"
    ```

    Sources:

    ```hcl
    --8<-- "examples/basic/sources-vmware-vsphere.pkr.hcl"
    ```

    ```hcl
    --8<-- "examples/basic/sources-vmware-desktop.pkr.hcl"
    ```

### Basic Authentication with Properties

???+ note "`build.pkr.hcl`"
    ```hcl
    --8<-- "examples/basic-with-properties/build.pkr.hcl"
    ```

??? note "`variables.pkr.hcl`"
    ```hcl
    --8<-- "examples/basic-with-properties/variables.pkr.hcl"
    ```

??? note "`sources-vmware-desktop.pkr.hcl`"
    ```hcl
    --8<-- "examples/basic-with-properties/sources-vmware-desktop.pkr.hcl"
    ```

??? note "`sources-vmware-vsphere.pkr.hcl`"
    ```hcl
    --8<-- "examples/basic-with-properties/sources-vmware-vsphere.pkr.hcl"
    ```

??? note "`versions.pkr.hcl`"
    ```hcl
    --8<-- "examples/basic-with-properties/versions.pkr.hcl"
    ```

### API Key Authentication

???+ note "`build.pkr.hcl`"
    ```hcl
    --8<-- "examples/api-key/build.pkr.hcl"
    ```

??? note "`variables.pkr.hcl`"
    ```hcl
    --8<-- "examples/api-key/variables.pkr.hcl"
    ```

??? note "`sources-vmware-desktop.pkr.hcl`"
    ```hcl
    --8<-- "examples/api-key/sources-vmware-desktop.pkr.hcl"
    ```

??? note "`sources-vmware-vsphere.pkr.hcl`"
    ```hcl
    --8<-- "examples/api-key/sources-vmware-vsphere.pkr.hcl"
    ```

??? note "`versions.pkr.hcl`"
    ```hcl
    --8<-- "examples/api-key/versions.pkr.hcl"
    ```

### Access Token Authentication

???+ note "`build.pkr.hcl`"
    ```hcl
    --8<-- "examples/access-token/build.pkr.hcl"
    ```

??? note "`variables.pkr.hcl"
    ```hcl
    --8<-- "examples/access-token/variables.pkr.hcl"
    ```

??? note "sources-vmware-desktop.pkr.hcl"
    ```hcl
    --8<-- "examples/access-token/sources-vmware-desktop.pkr.hcl"
    ```

??? note "sources-vmware-vsphere.pkr.hcl"
    ```hcl
    --8<-- "examples/access-token/sources-vmware-vsphere.pkr.hcl"
    ```

??? note "versions.pkr.hcl"
    ```hcl
    --8<-- "examples/access-token/versions.pkr.hcl"
    ```

!!! note
    Access token authentication is not supported on Artifactory open-source.

[packer]: https://www.packer.io/
[examples]: https://github.com/tenthirtyam/packer-plugin-artifactory/tree/main/examples/
