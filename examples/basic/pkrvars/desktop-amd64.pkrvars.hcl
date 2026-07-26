# Copyright (c) Ryan Johnson
# SPDX-License-Identifier: MPL-2.0

# VMware Fusion / VMware Workstation

guest_os_type        = "debian-64"
network_adapter_type = "e1000"
cdrom_adapter_type   = "ide"
disk_adapter_type    = "lsilogic"
boot_command = [
  "<wait><esc><wait>",
  "auto url=http://{{ .HTTPIP }}:{{ .HTTPPort }}/ks.cfg ",
  "netcfg/get_hostname={{ .Name }}<enter>"
]
