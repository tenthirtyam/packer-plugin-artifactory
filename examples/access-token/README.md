# Access Token Authentication

Build a machine image with the Debian guest operating system, export **OVA or OVF**, and
upload the artifact to Artifactory using access token authentication.

Not supported on Artifactory open-source (including the local Docker OSS image). Use
[`basic/`](../basic/) or [`api-key/`](../api-key/) there.

Set `ARTIFACTORY_TOKEN` (the interactive runner prompts when it is unset). Point
`ARTIFACTORY_URL` and `ARTIFACTORY_REPOSITORY` at a token-capable instance.

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
export ARTIFACTORY_URL="https://artifactory.example.com/artifactory"
export ARTIFACTORY_TOKEN="..."
export ARTIFACTORY_REPOSITORY="your-repo"
cd examples/access-token
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
export ARTIFACTORY_URL="https://artifactory.example.com/artifactory"
export ARTIFACTORY_TOKEN="..."
export ARTIFACTORY_REPOSITORY="your-repo"
cd examples/access-token
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
export ARTIFACTORY_URL="https://artifactory.example.com/artifactory"
export ARTIFACTORY_TOKEN="..."
export ARTIFACTORY_REPOSITORY="your-repo"
cd examples/access-token
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
