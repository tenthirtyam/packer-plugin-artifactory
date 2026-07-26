# Basic Authentication with Custom Properties.

build {
  sources = [
    "source.vmware-iso.example",
    "source.vsphere-iso.example",
  ]

  provisioner "shell" {
    inline = [
      "sudo apt-get update",
      "sudo apt-get install -y open-vm-tools"
    ]
  }

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
}
