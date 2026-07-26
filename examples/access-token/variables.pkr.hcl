# ---------------------------------------------------------------------------
# Artifactory Access Token Authentication
# ---------------------------------------------------------------------------

variable "artifactory_url" {
  type        = string
  description = "URL of the Artifactory instance to use for the build."
  default     = env("ARTIFACTORY_URL") != "" ? env("ARTIFACTORY_URL") : "http://localhost:8081/artifactory"
}

variable "artifactory_token" {
  type        = string
  description = "Access token to authenticate to the Artifactory instance."
  default     = env("ARTIFACTORY_TOKEN")
  sensitive   = true
}

variable "artifactory_repository" {
  type        = string
  description = "Repository in the Artifactory instance to use for the build."
  default     = env("ARTIFACTORY_REPOSITORY") != "" ? env("ARTIFACTORY_REPOSITORY") : "example-repo-local"
}

# ---------------------------------------------------------------------------
# ISO
# Note: Set using eval "$(./scripts/resolve-debian-iso.sh arm64|amd64)"
# ---------------------------------------------------------------------------

variable "iso_url" {
  type        = string
  description = "URL of the Debian .iso file to use for the build."
}

variable "iso_checksum" {
  type        = string
  description = "Checksum of the Debian .iso file to use for the build."
}

# ---------------------------------------------------------------------------
# Guest Identity
# ---------------------------------------------------------------------------

variable "build_username" {
  type        = string
  description = "Username to authenticate to the guest operating system."
  default     = "packer"
}

variable "build_password" {
  type        = string
  description = "Password to authenticate to the guest operating system."
  default     = "packer"
  sensitive   = true
}

variable "vm_guest_os_language" {
  type        = string
  description = "Language for the guest operating system."
  default     = "en"
}

variable "vm_guest_os_country" {
  type        = string
  description = "Country for the guest operating system."
  default     = "US"
}

variable "vm_guest_os_locale" {
  type        = string
  description = "Locale for the guest operating system."
  default     = "en_US.UTF-8"
}

variable "vm_guest_os_keyboard" {
  type        = string
  description = "Keyboard layout for the guest operating system."
  default     = "us"
}

variable "vm_guest_os_timezone" {
  type        = string
  description = "Timezone for the guest operating system."
  default     = "UTC"
}

variable "debian_mirror_hostname" {
  type        = string
  description = "Hostname of the Debian mirror to use for the build."
  default     = "deb.debian.org"
}

variable "ssh_timeout" {
  type        = string
  description = "Timeout for the SSH connection."
  default     = "30m"
}

variable "shutdown_command" {
  type        = string
  description = "Command to use to shut down the guest operating system."
  default     = null
}

# ---------------------------------------------------------------------------
# Sizing / Boot
# ---------------------------------------------------------------------------

variable "vm_name" {
  type        = string
  description = "Name of the virtual machine."
  default     = null
}

variable "cpus" {
  type        = number
  description = "Number of CPUs for the virtual machine."
  default     = 2
}

variable "memory" {
  type        = number
  description = "Memory for the virtual machine."
  default     = 2048
}

variable "disk_size" {
  type        = number
  description = "Disk size for the virtual machine."
  default     = 20480
}

variable "boot_wait" {
  type        = string
  description = "Time to wait for the virtual machine to boot."
  default     = "5s"
}

# ---------------------------------------------------------------------------
# Desktop (Defaults for VMware Fusion on Apple Silicon)
# ---------------------------------------------------------------------------

variable "guest_os_type" {
  type        = string
  description = "Guest operating system type for the virtual machine."
  default     = "arm-debian-64"
}

variable "network_adapter_type" {
  type        = string
  description = "Network adapter type for the virtual machine."
  default     = "e1000e"
}

variable "cdrom_adapter_type" {
  type        = string
  description = "CD-ROM adapter type for the virtual machine."
  default     = "sata"
}

variable "disk_adapter_type" {
  type        = string
  description = "Disk adapter type for the virtual machine."
  default     = "sata"
}

variable "format" {
  type        = string
  description = "Export format for desktop (vmware-iso) and vSphere (export.output_format): ova or ovf."
  default     = "ova"
  validation {
    condition     = contains(["ova", "ovf"], var.format)
    error_message = "Format must be \"ova\" or \"ovf\"."
  }
}

variable "ovftool_options" {
  type        = list(string)
  description = "Extra ovftool arguments for desktop (vmware-iso) export."
  default     = []
}

variable "vmx_data" {
  type        = map(string)
  description = "Extra VMX data for the virtual machine."
  default = {
    "usb_xhci.present" = "true"
  }
}

variable "boot_command" {
  type        = list(string)
  description = "Boot command for the virtual machine."
  default = [
    "<wait><up>e<wait>",
    "<down><down><down>",
    "<right><right><right><right><right><right><right><right><right><right>",
    "<right><right><right><right><right><right><right><right><right><right>",
    "<right><right><right><right><right><right><right><right><right><right>",
    "<right><right><right><right><right><right><wait>",
    "install <wait>",
    " preseed/url=http://{{ .HTTPIP }}:{{ .HTTPPort }}/ks.cfg <wait>",
    "debian-installer=en_US.UTF-8 <wait>auto <wait>locale=en_US.UTF-8 <wait>",
    "kbd-chooser/method=us <wait>keyboard-configuration/xkb-keymap=us <wait>",
    "netcfg/get_hostname={{ .Name }} <wait>netcfg/get_domain={{ .Name }} <wait>",
    "fb=false <wait>debconf/frontend=noninteractive <wait>",
    "console-setup/ask_detect=false <wait>console-keymaps-at/keymap=us <wait>",
    "grub-installer/bootdev=/dev/sda <wait>",
    "<f10><wait>"
  ]
}

# ---------------------------------------------------------------------------
# vSphere
# ---------------------------------------------------------------------------

variable "vsphere_server" {
  type        = string
  description = "Fully qualified domain name of the vCenter instance to use for the build."
  default     = env("VSPHERE_SERVER") != "" ? env("VSPHERE_SERVER") : "vc01.example.com"
}

variable "vsphere_username" {
  type        = string
  description = "Username to authenticate to the vSphere instance."
  default     = env("VSPHERE_USERNAME") != "" ? env("VSPHERE_USERNAME") : "administrator@vsphere.local"
  sensitive   = true
}

variable "vsphere_password" {
  type        = string
  description = "Password to authenticate to the vSphere instance."
  default     = env("VSPHERE_PASSWORD") != "" ? env("VSPHERE_PASSWORD") : "VMw@re1!"
  sensitive   = true
}

variable "vsphere_datacenter" {
  type        = string
  description = "Datacenter in the vSphere inventory to use for the build."
  default     = env("VSPHERE_DATACENTER") != "" ? env("VSPHERE_DATACENTER") : "dc01"
}

variable "vsphere_cluster" {
  type        = string
  description = "Cluster in the vSphere inventory to use for the build."
  default     = env("VSPHERE_CLUSTER") != "" ? env("VSPHERE_CLUSTER") : "cl01"
}

variable "vsphere_datastore" {
  type        = string
  description = "Datastore in the vSphere inventory to use for the build."
  default     = env("VSPHERE_DATASTORE") != "" ? env("VSPHERE_DATASTORE") : "nfs"
}

variable "vsphere_network" {
  type        = string
  description = "Network in the vSphere inventory to use for the build."
  default     = env("VSPHERE_NETWORK") != "" ? env("VSPHERE_NETWORK") : "VM Network"
}

variable "vsphere_folder" {
  type        = string
  description = "Virtual machine folder in the vSphere inventory to use for the build."
  default     = env("VSPHERE_FOLDER") != "" ? env("VSPHERE_FOLDER") : "templates"
}

variable "vsphere_insecure" {
  type        = bool
  description = "Whether to use an insecure connection to the vSphere instance."
  default     = true
}

variable "vsphere_guest_os_type" {
  type        = string
  description = "Guest operating system type for the virtual machine."
  default     = "debian12_64Guest"
}

variable "vsphere_disk_controller_type" {
  type        = list(string)
  description = "Disk controller type for the virtual machine."
  default     = ["pvscsi"]
}

variable "vsphere_disk_thin_provisioned" {
  type        = bool
  description = "Whether to use thin provisioning for the virtual machine."
  default     = true
}

variable "vsphere_network_card" {
  type        = string
  description = "Network card type for the virtual machine."
  default     = "vmxnet3"
}

variable "vsphere_boot_command" {
  type        = list(string)
  description = "Boot command for the virtual machine."
  default = [
    "<wait><esc><wait>",
    "auto url=http://{{ .HTTPIP }}:{{ .HTTPPort }}/ks.cfg ",
    "netcfg/get_hostname={{ .Name }}<enter>"
  ]
}

variable "vsphere_export_force" {
  type        = bool
  description = "Whether to force the export of the virtual machine."
  default     = true
}

variable "output_directory" {
  type        = string
  description = "Directory for build artifacts (desktop output and vSphere export)."
  default     = "output"
}

# ---------------------------------------------------------------------------
# Locals
# ---------------------------------------------------------------------------

locals {
  guest         = regex("debian-[0-9.]+-(?:amd64|arm64)", var.iso_url)
  artifact_name = coalesce(var.vm_name, local.guest)
  artifact_path = "${local.guest}/${formatdate("YYYYMMDDhhmmss", timestamp())}"
  shutdown_command = coalesce(
    var.shutdown_command,
    "echo '${var.build_password}' | sudo -S shutdown -P now"
  )

  data_source_content = {
    "/ks.cfg" = templatefile("${abspath(path.root)}/data/ks.pkrtpl.hcl", {
      build_username         = var.build_username
      build_password         = var.build_password
      vm_guest_os_language   = var.vm_guest_os_language
      vm_guest_os_country    = var.vm_guest_os_country
      vm_guest_os_locale     = var.vm_guest_os_locale
      vm_guest_os_keyboard   = var.vm_guest_os_keyboard
      vm_guest_os_timezone   = var.vm_guest_os_timezone
      debian_mirror_hostname = var.debian_mirror_hostname
    })
  }
}
