# Copyright (c) Ryan Johnson
# SPDX-License-Identifier: MPL-2.0

# For full specification on the configuration of this file visit:
# https://github.com/hashicorp/integration-template#metadata-configuration
integration {
  name = "Artifactory"
  description = "A plugin to upload virtual machine artifacts to JFrog Artifactory."
  identifier = "packer/tenthirtyam/artifactory"
  component {
    type = "post-processor"
    name = "Artifactory"
    slug = "artifactory"
  }
}
