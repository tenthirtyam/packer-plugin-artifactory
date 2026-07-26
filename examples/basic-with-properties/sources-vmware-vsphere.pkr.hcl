# VMware vSphere

source "vsphere-iso" "example" {
  vcenter_server      = var.vsphere_server
  username            = var.vsphere_username
  password            = var.vsphere_password
  insecure_connection = var.vsphere_insecure

  datacenter = var.vsphere_datacenter
  cluster    = var.vsphere_cluster
  datastore  = var.vsphere_datastore
  folder     = var.vsphere_folder

  vm_name       = local.artifact_name
  guest_os_type = var.vsphere_guest_os_type
  CPUs          = var.cpus
  RAM           = var.memory

  disk_controller_type = var.vsphere_disk_controller_type
  storage {
    disk_size             = var.disk_size
    disk_thin_provisioned = var.vsphere_disk_thin_provisioned
  }

  network_adapters {
    network_card = var.vsphere_network_card
    network      = var.vsphere_network
  }

  iso_url      = var.iso_url
  iso_checksum = var.iso_checksum

  ssh_username     = var.build_username
  ssh_password     = var.build_password
  ssh_timeout      = var.ssh_timeout
  shutdown_command = local.shutdown_command

  http_content = local.data_source_content
  boot_wait    = var.boot_wait
  boot_command = var.vsphere_boot_command

  export {
    force            = var.vsphere_export_force
    output_directory = var.output_directory
    output_format    = var.format
  }
}
