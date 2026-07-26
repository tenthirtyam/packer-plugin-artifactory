# VMware Fusion / VMware Workstation

source "vmware-iso" "example" {
  iso_url              = var.iso_url
  iso_checksum         = var.iso_checksum
  vm_name              = local.artifact_name
  guest_os_type        = var.guest_os_type
  memory               = var.memory
  cpus                 = var.cpus
  disk_size            = var.disk_size
  ssh_username         = var.build_username
  ssh_password         = var.build_password
  ssh_timeout          = var.ssh_timeout
  shutdown_command     = local.shutdown_command
  output_directory     = var.output_directory
  format               = var.format
  ovftool_options      = var.ovftool_options
  network_adapter_type = var.network_adapter_type
  cdrom_adapter_type   = var.cdrom_adapter_type
  disk_adapter_type    = var.disk_adapter_type
  boot_wait            = var.boot_wait
  boot_command         = var.boot_command
  http_content         = local.data_source_content
  vmx_data             = var.vmx_data
}
