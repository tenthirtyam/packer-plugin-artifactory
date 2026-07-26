# Access Token Authentication.

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
    url             = var.artifactory_url
    access_token    = var.artifactory_token
    repository      = var.artifactory_repository
    artifact_name   = local.artifact_name
    artifact_path   = local.artifact_path
    overwrite       = true
    max_retries     = 5
    timeout_seconds = 60
  }
}
