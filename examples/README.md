# Packer Artifactory Post-Processor Examples

Build a machine image with the Debian guest operating system, export **OVA or OVF**, and upload the
artifact to Artifactory.

Each directory uses an Artifactory authentication option.

| Example Directory                                  | Authentication                                                          |
|----------------------------------------------------|-------------------------------------------------------------------------|
| [`basic/`](basic/)                                 | Basic Authentication                                                    |
| [`basic-with-properties/`](basic-with-properties/) | Basic Authentication with Custom Properties                             |
| [`api-key/`](api-key/)                             | API Key Authentication                                                  |
| [`access-token/`](access-token/)                   | Access Token Authentication (Not supported on Artifactory open-source)  |

## Interactive Runner

From the repository root:

```bash
./examples/run-example.sh          # Initialize Missing Plugins
./examples/run-example.sh --dev    # Install Development Plugin and Initialize Missing Plugins
./examples/run-example.sh -h       # Help
```

## Manual Build Example (`basic/`)

VMware Fusion on Apple Silicon:

```bash
make dev
eval "$(./scripts/resolve-debian-iso.sh arm64)"
cd examples/basic
packer init .
# OVA (default)
packer build -only=vmware-iso.example .
# OVA
packer build -only=vmware-iso.example \
  -var-file=pkrvars/export-ova.pkrvars.hcl .
# OVF
packer build -only=vmware-iso.example \
  -var-file=pkrvars/export-ovf.pkrvars.hcl .
```

VMware Fusion on Intel / VMware Workstation:

```bash
make dev
eval "$(./scripts/resolve-debian-iso.sh amd64)"
cd examples/basic
packer init .
# OVA
packer build -only=vmware-iso.example \
  -var-file=pkrvars/desktop-amd64.pkrvars.hcl \
  -var-file=pkrvars/export-ova.pkrvars.hcl .
# OVF
packer build -only=vmware-iso.example \
  -var-file=pkrvars/desktop-amd64.pkrvars.hcl \
  -var-file=pkrvars/export-ovf.pkrvars.hcl .
```

VMware vSphere:

```bash
make dev
eval "$(./scripts/resolve-debian-iso.sh amd64)"
cd examples/basic
packer init .
# OVA
packer build -only=vsphere-iso.example \
  -var-file=pkrvars/vsphere.pkrvars.hcl \
  -var-file=pkrvars/export-ova.pkrvars.hcl .
# OVF
packer build -only=vsphere-iso.example \
  -var-file=pkrvars/vsphere.pkrvars.hcl \
  -var-file=pkrvars/export-ovf.pkrvars.hcl .
```
