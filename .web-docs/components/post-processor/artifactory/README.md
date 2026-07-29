Type: `artifactory`

Artifact BuilderId: `packer.post-processor.artifactory`

The Packer Plugin for Artifactory is a post-processor (`artifactory`) that
can be used with [HashiCorp Packer][packer] to upload virtual machine artifacts
to JFrog Artifactory.

## Supported Platforms

This plugin supports the following platforms:

| Platform           | Type                               | Support     |
|--------------------|------------------------------------|-------------|
| VMware vSphere     | Bare Metal Hypervisor              | Supported   |
| VMware Fusion      | Desktop Hypervisor (macOS)         | Supported   |
| VMware Workstation | Desktop Hypervisor (Windows/Linux) | Supported   |
| Other              | Various                            | Best Effort |

**Note:** While the plugin may function with other platforms that generate
OVA/OVF files, official testing and support are currently limited to the VMware by Broadcom ecosystem.

## Supported Artifacts

The plugin supports the following artifacts:

- **OVA**: `.ova` - A single compressed archive package containing all the
  supporting virtual machine files.
- **OVF**: `.ovf` - An array of virtual machine files supporting an Open
  Virtualization Format package.

## Examples

Examples are available in the [`examples`][examples] directory of the GitHub
repository.

## Configuration Reference

The following configuration options are available for the post-processor.

**Required:**

<!-- Code generated from the comments of the artifactoryConfig struct in post-processor/artifactory/config.go; DO NOT EDIT MANUALLY -->

- `url` (string) - The base URL of the Artifactory instance (e.g., https://packages.example.com/artifactory).

- `repository` (string) - The name of the repository to upload the artifact to.

- `artifact_name` (string) - The name of the artifact being uploaded.

<!-- End of code generated from the comments of the artifactoryConfig struct in post-processor/artifactory/config.go; -->


**Optional:**

<!-- Code generated from the comments of the artifactoryConfig struct in post-processor/artifactory/config.go; DO NOT EDIT MANUALLY -->

- `api_key` (string) - An API key to authenticate to the Artifactory instance.

- `access_token` (string) - An access token to authenticate to the Artifactory instance.

- `username` (string) - A username to authenticate to the Artifactory instance.

- `password` (string) - A password to authenticate to the Artifactory instance.

- `artifact_path` (string) - The path within the repository where the artifact will be stored.

- `artifact_path_properties` (bool) - Whether to apply the `properties` to the path of the artifact in addition
  to each uploaded file. (default: `false`)
  
  ~> **Note:** Ignored if `properties` is empty.
  
  ~> **Note:** Ignored if the Artifactory instance does not support setting
  properties on the artifact's repository path. Uploaded files are unaffected
  by this restriction.

- `max_retries` (int) - The number of retry attempts for failed uploads. (default: `3`)

- `timeout_seconds` (int) - The request timeout in seconds. (default: `30`)

- `overwrite` (\*bool) - Whether to overwrite existing artifacts. (default: `false`)
  
  ~> **Note:** The upload will fail if the artifact already exists and the
  overwrite flag is set to `false`.

- `properties` (map[string]string) - Custom properties to attach to artifacts.

- `additional_ovf_extensions` ([]string) - Additional file extensions to upload alongside an OVF package, beyond
  the default set (`.ovf`, `.mf`, `.vmdk`, `.cert`, `.vdi`). Extensions may be
  given with or without a leading dot (_e.g._ `nvram` or `.nvram`).

<!-- End of code generated from the comments of the artifactoryConfig struct in post-processor/artifactory/config.go; -->


## Example Usage

~> **Note:** Examples for VMware vSphere and VMware desktop hypervisors are
available in the repository [`examples`][examples] directory.

### Basic Authentication

```hcl
post-processor "artifactory" {
  url           = var.artifactory_url
  username      = var.artifactory_username
  password      = var.artifactory_password
  repository    = var.artifactory_repository
  artifact_name = local.artifact_name
  artifact_path = local.artifact_path
}
```

Credentials may also be supplied with the `ARTIFACTORY_URL`,
`ARTIFACTORY_USERNAME`, `ARTIFACTORY_PASSWORD`, and `ARTIFACTORY_REPOSITORY`
environment variables.

### Basic Authentication with Properties

```hcl
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
    "os.version"      = local.guest_version
    "os.arch"         = local.guest_arch
  }
}
```

### API Key Authentication

```hcl
post-processor "artifactory" {
  url             = var.artifactory_url
  api_key         = var.artifactory_api_key
  repository      = var.artifactory_repository
  artifact_name   = local.artifact_name
  artifact_path   = local.artifact_path
  overwrite       = true
  max_retries     = 5
  timeout_seconds = 60
}
```

The API key may also be supplied with `ARTIFACTORY_API_KEY`.

### Access Token Authentication

```hcl
post-processor "artifactory" {
  url             = var.artifactory_url
  access_token    = var.artifactory_token
  repository      = var.artifactory_repository
  artifact_name   = local.artifact_name
  artifact_path   = local.artifact_path
  overwrite       = true
  max_retries     = 5
  timeout_seconds = 60
}
```

The access token may also be supplied with `ARTIFACTORY_TOKEN`.

[packer]: https://www.packer.io/
[examples]: https://github.com/tenthirtyam/packer-plugin-artifactory/tree/main/examples/
