# Basic Authentication

Build a machine image with the Debian guest operating system, export **OVA or OVF**, and
upload the artifact to Artifactory using basic (username / password) authentication.

URL, username, password, and repository default to the local Docker Artifactory values.
Override with `ARTIFACTORY_*` or Packer `-var` / `-var-file`.

## Interactive Runner

From the repository root:

```bash
./examples/run-example.sh          # Initialize Missing Plugins
./examples/run-example.sh --dev    # Install Development Plugin and Initialize Missing Plugins
./examples/run-example.sh -h       # Help
```

## Manual Build

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
