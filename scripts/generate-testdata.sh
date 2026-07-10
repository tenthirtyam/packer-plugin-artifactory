#!/bin/bash

# Copyright (c) Ryan Johnson
# SPDX-License-Identifier: MPL-2.0

# Setup the test data for the integration tests.
#
# Usage:
#   ./scripts/generate-testdata.sh

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTDATA_DIR="$SCRIPT_DIR/../post-processor/artifactory/testdata"

mkdir -p "$TESTDATA_DIR"
cd "$TESTDATA_DIR"

echo -e "${BLUE}==> Generating Test Data${NC}"

rm -rf ova ovf
mkdir -p ova ovf

if [ ! -f "ovf/sample.ovf" ] && [ "$SKIP_MANUAL_OVF" != "true" ]; then
    cat > ovf/sample.ovf << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<Envelope xmlns="http://schemas.dmtf.org/ovf/envelope/1" xmlns:cim="http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_VirtualSystemSettingData" xmlns:ovf="http://schemas.dmtf.org/ovf/envelope/1" xmlns:rasd="http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_ResourceAllocationSettingData" xmlns:vmw="http://www.vmware.com/schema/ovf" xmlns:vssd="http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_VirtualSystemSettingData" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <References>
    <File ovf:href="sample-disk-0.vmdk" ovf:id="file1" ovf:size="1024"/>
  </References>
  <DiskSection>
    <Info>Virtual disk information</Info>
    <Disk ovf:capacity="1" ovf:diskId="vmdisk1" ovf:fileRef="file1" ovf:format="http://www.vmware.com/interfaces/specifications/vmdk.html#streamOptimized"/>
  </DiskSection>
  <NetworkSection>
    <Info>Logical networks</Info>
    <Network ovf:name="VM Network">
      <Description>VM Network</Description>
    </Network>
  </NetworkSection>
  <VirtualSystem ovf:id="debian-arm64">
    <Info>A virtual machine</Info>
    <Name>debian-arm64</Name>
    <OperatingSystemSection ovf:id="96" vmw:osType="arm-debian-64">
      <Info>Guest Operating System</Info>
      <Description>Debian GNU/Linux 13 (64-bit)</Description>
    </OperatingSystemSection>
    <VirtualHardwareSection>
      <Info>Virtual hardware requirements</Info>
      <System>
        <vssd:ElementName>Virtual Hardware Family</vssd:ElementName>
        <vssd:InstanceID>0</vssd:InstanceID>
        <vssd:VirtualSystemIdentifier>debian-arm64</vssd:VirtualSystemIdentifier>
        <vssd:VirtualSystemType>vmx-13</vssd:VirtualSystemType>
      </System>
      <Item>
        <rasd:AllocationUnits>hertz * 10^6</rasd:AllocationUnits>
        <rasd:Description>Number of Virtual CPUs</rasd:Description>
        <rasd:ElementName>2 virtual CPU(s)</rasd:ElementName>
        <rasd:InstanceID>1</rasd:InstanceID>
        <rasd:ResourceType>3</rasd:ResourceType>
        <rasd:VirtualQuantity>2</rasd:VirtualQuantity>
      </Item>
      <Item>
        <rasd:AllocationUnits>byte * 2^20</rasd:AllocationUnits>
        <rasd:Description>Memory Size</rasd:Description>
        <rasd:ElementName>2048MB of memory</rasd:ElementName>
        <rasd:InstanceID>2</rasd:InstanceID>
        <rasd:ResourceType>4</rasd:ResourceType>
        <rasd:VirtualQuantity>2048</rasd:VirtualQuantity>
      </Item>
      <Item>
        <rasd:Address>0</rasd:Address>
        <rasd:Description>SATA Controller</rasd:Description>
        <rasd:ElementName>SATA controller 0</rasd:ElementName>
        <rasd:InstanceID>3</rasd:InstanceID>
        <rasd:ResourceSubType>AHCI</rasd:ResourceSubType>
        <rasd:ResourceType>20</rasd:ResourceType>
      </Item>
      <Item>
        <rasd:AddressOnParent>0</rasd:AddressOnParent>
        <rasd:ElementName>Hard disk 1</rasd:ElementName>
        <rasd:HostResource>ovf:/disk/vmdisk1</rasd:HostResource>
        <rasd:InstanceID>4</rasd:InstanceID>
        <rasd:Parent>3</rasd:Parent>
        <rasd:ResourceType>17</rasd:ResourceType>
      </Item>
    </VirtualHardwareSection>
  </VirtualSystem>
</Envelope>
EOF
fi

VMCLI="/Applications/VMware Fusion.app/Contents/Library/vmcli"
OVFTOOL="/Applications/VMware Fusion.app/Contents/Library/VMware OVF Tool/ovftool"

if [ ! -f "$VMCLI" ]; then
    echo -e "${RED}==> vmcli not found. This script requires VMware Fusion.${NC}"
    exit 1
fi

if [ ! -f "$OVFTOOL" ]; then
    echo -e "${RED}==> ovftool not found. This script requires VMware Fusion.${NC}"
    exit 1
fi

echo -e "${YELLOW}==> Creating virtual machine...${NC}"

VM_TEMP_DIR=$(mktemp -d)
VM_NAME="debian-arm64"
VMX_PATH="$VM_TEMP_DIR/$VM_NAME.vmx"
VMDK_PATH="$VM_TEMP_DIR/$VM_NAME.vmdk"

"$VMCLI" VM Create -n "$VM_NAME" -d "$VM_TEMP_DIR" -c debian13-64 > /dev/null
rm -f "$VMDK_PATH"
"$VMCLI" "$VMX_PATH" Disk Create -f "$VMDK_PATH" -a lsilogic -s 20480MB -t 0 > /dev/null
"$VMCLI" "$VMX_PATH" Sata SetPresent sata0 TRUE > /dev/null
"$VMCLI" "$VMX_PATH" Disk SetBackingInfo sata0:0 disk "$VMDK_PATH" FALSE > /dev/null
"$VMCLI" "$VMX_PATH" ConfigParams SetEntry sata0:0.present TRUE > /dev/null
"$VMCLI" "$VMX_PATH" ConfigParams SetEntry sata0:0.deviceType disk > /dev/null
"$VMCLI" "$VMX_PATH" ConfigParams SetEntry guestOS arm-debian-64 > /dev/null
"$VMCLI" "$VMX_PATH" ConfigParams SetEntry displayName "$VM_NAME" > /dev/null
"$VMCLI" "$VMX_PATH" ConfigParams SetEntry memsize 2048 > /dev/null
"$VMCLI" "$VMX_PATH" ConfigParams SetEntry numvcpus 2 > /dev/null
"$VMCLI" "$VMX_PATH" Ethernet SetPresent ethernet0 TRUE > /dev/null
"$VMCLI" "$VMX_PATH" Ethernet SetVirtualDevice ethernet0 e1000e > /dev/null
"$VMCLI" "$VMX_PATH" Ethernet SetConnectionType ethernet0 nat > /dev/null

if "$OVFTOOL" "$VMX_PATH" "$TESTDATA_DIR/ovf/sample" > /dev/null 2>&1; then
    echo -e "${GREEN}==> Successfully exported to OVF artifact.${NC}"

    EXPORTED_DIR=$(find "$TESTDATA_DIR/ovf/sample" -name "*.ovf" -exec dirname {} \; | head -1)
    if [ -n "$EXPORTED_DIR" ]; then
        mv "$EXPORTED_DIR"/*.ovf ovf/sample.ovf 2>/dev/null || true
        mv "$EXPORTED_DIR"/*.mf ovf/sample.mf 2>/dev/null || true
        mv "$EXPORTED_DIR"/*.vmdk ovf/sample-disk-0.vmdk 2>/dev/null || true
        rm -rf "$TESTDATA_DIR/ovf/sample"
    fi

    rm -rf "$VM_TEMP_DIR"
    SKIP_MANUAL_OVF=true
else
    echo -e "${YELLOW}==> VM export failed, falling back to manual VMDK creation...${NC}"
    cp "$VMDK_PATH" ovf/sample-disk-0.vmdk 2>/dev/null || true
    rm -rf "$VM_TEMP_DIR"
    echo -e "${GREEN}==> VMDK created from VM.${NC}"
fi

if [ -f "ovf/sample.ovf" ]; then
    sed -i.bak -E 's/ovf:href="[^"]*\.vmdk"/ovf:href="sample-disk-0.vmdk"/' ovf/sample.ovf
    rm -f ovf/sample.ovf.bak
fi

if [ -f "ovf/sample-disk-0.vmdk" ]; then
    ACTUAL_SIZE=$(stat -f%z "ovf/sample-disk-0.vmdk" 2>/dev/null || stat -c%s "ovf/sample-disk-0.vmdk" 2>/dev/null)
    if [ -n "$ACTUAL_SIZE" ]; then
        sed -i.bak "s/ovf:size=\"1024\"/ovf:size=\"$ACTUAL_SIZE\"/" ovf/sample.ovf
        rm -f ovf/sample.ovf.bak
    fi
fi

if [ ! -f "ovf/sample.ovf" ] || [ ! -f "ovf/sample-disk-0.vmdk" ]; then
    echo -e "${RED}==> Missing required files for checksum calculation${NC}"
    exit 1
fi

cat > ovf/sample.mf << EOF
SHA256(sample.ovf)= $(shasum -a 256 ovf/sample.ovf | cut -d' ' -f1)
SHA256(sample-disk-0.vmdk)= $(shasum -a 256 ovf/sample-disk-0.vmdk | cut -d' ' -f1)
EOF

cd ovf
if "$OVFTOOL" sample.ovf ../ova/sample.ova > /dev/null 2>&1; then
    echo -e "${GREEN}==> Successfully created OVA artifact with ovftool.${NC}"
else
    echo -e "${YELLOW}==> Manually creating OVA artifact...${NC}"
    tar -cf ../ova/sample.ova sample.ovf sample.mf sample-disk-0.vmdk
    echo -e "${GREEN}==> Successfully created OVA artifact.${NC}"
fi
cd ..

cat > ovf/sample.cert << 'EOF'
-----BEGIN CERTIFICATE-----
MIICdTCCAd4CAQAwDQYJKoZIhvcNAQEEBQAwgYsxCzAJBgNVBAYTAlVTMRMwEQYD
VQQIEwpDYWxpZm9ybmlhMRYwFAYDVQQHEw1Nb3VudGFpbiBWaWV3MRQwEgYDVQQK
EwtQYXlQYWwgSW5jLjETMBEGA1UECxQKc2FuZGJveF9hcGkxFDASBgNVBAMTC3Nh
bmRib3hfYXBpMB4XDTA0MDEwMTAwMDAwMFoXDTI1MDEwMTAwMDAwMFowgYsxCzAJ
BgNVBAYTAlVTMRMwEQYDVQQIEwpDYWxpZm9ybmlhMRYwFAYDVQQHEw1Nb3VudGFp
biBWaWV3MRQwEgYDVQQKEwtQYXlQYWwgSW5jLjETMBEGA1UECxQKc2FuZGJveF9h
cGkxFDASBgNVBAMTC3NhbmRib3hfYXBpMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCB
iQKBgQC4kqhyafjJ02eqODjw7Vp0vtae4yaQvmtlg5+stM8nUU4IOhXHddc8/Uq5
8B6T3XIiiJTp34un7v2MR4lDHDWoE5A4EzO4DNQxr/tgUhNzwow3DEMb+gg2ID5r
uKWMsqhWPKhWxWXSMn4PnLD7WqMhyUm+WBFc8+aQiLuDQsrxKwIDAQABMA0GCSqG
SIb3DQEBBAUAA4GBAGbVfPgKt4NWtl7uyuMjCb2hF0C0q3v1cwx4ADCK9vMsC0+S
ddHq0xcmrcrHJuhraaJc7zNjzB2RfR+TVkqpQXGfisNUooPYfNHNUhKpwjfDi67L
-----END CERTIFICATE-----
EOF

echo "Sample NVRAM/BIOS settings for test VM" > ovf/sample.nvram
echo "Sample CD-ROM ISO content for testing" > ovf/sample.iso

cat > ovf/sample.mf << EOF
SHA256(sample.ovf)= $(shasum -a 256 ovf/sample.ovf | cut -d' ' -f1)
SHA256(sample-disk-0.vmdk)= $(shasum -a 256 ovf/sample-disk-0.vmdk | cut -d' ' -f1)
SHA256(sample.cert)= $(shasum -a 256 ovf/sample.cert | cut -d' ' -f1)
SHA256(sample.nvram)= $(shasum -a 256 ovf/sample.nvram | cut -d' ' -f1)
SHA256(sample.iso)= $(shasum -a 256 ovf/sample.iso | cut -d' ' -f1)
EOF

echo -e "\n${YELLOW}OVA Package:${NC}"
ls -lh ova/ | tail -n +2 | while read line; do
    echo -e "  ova/${line}"
done
echo -e "\n${YELLOW}OVF Package:${NC}"
ls -lh ovf/ | tail -n +2 | while read line; do
    echo -e "  ovf/${line}"
done
