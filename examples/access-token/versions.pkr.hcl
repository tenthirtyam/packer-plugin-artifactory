packer {
  required_plugins {
    artifactory = {
      version = ">= 0.1.0"
      source  = "github.com/tenthirtyam/artifactory"
    }
    vmware = {
      version = ">= 2.0.0"
      source  = "github.com/vmware/vmware"
    }
    vsphere = {
      version = ">= 2.0.0"
      source  = "github.com/vmware/vsphere"
    }
  }
}
