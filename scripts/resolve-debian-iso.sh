#!/usr/bin/env bash

# Copyright (c) Ryan Johnson
# SPDX-License-Identifier: MPL-2.0

# Resolve the latest stable Debian image URL and checksum file for the given architecture.
#
# Usage:
#   eval "$(./scripts/resolve-debian-iso.sh amd64)"
#   eval "$(./scripts/resolve-debian-iso.sh arm64)"
#   ./scripts/resolve-debian-iso.sh amd64 --hcl

set -euo pipefail

ARCH="${1:-}"
MODE="${2:-export}"

usage() {
  echo "usage: $0 <amd64|arm64> [--export|--hcl]" >&2
  exit 1
}

case "${ARCH}" in
  amd64|arm64) ;;
  *) usage ;;
esac

case "${MODE}" in
  --export|export|"") MODE="export" ;;
  --hcl|hcl) MODE="hcl" ;;
  *) usage ;;
esac

BASE_URL="https://cdimage.debian.org/debian-cd/current/${ARCH}/iso-cd"
SUMS_URL="${BASE_URL}/SHA256SUMS"

printf 'Retrieving latest release for Debian (%s)...\n' "${ARCH}" >&2
sums="$(curl -fsSL --connect-timeout 15 --max-time 60 "${SUMS_URL}")" || {
  echo "error: failed to download ${SUMS_URL}" >&2
  exit 1
}

iso_name="$(
  printf '%s\n' "${sums}" \
    | awk -v arch="${ARCH}" '
        $2 ~ ("^debian-[0-9.]+-" arch "-netinst\\.iso$") { print $2; exit }
      '
)"

if [ -z "${iso_name}" ]; then
  echo "error: could not find debian-*-${ARCH}-netinst.iso in ${SUMS_URL}" >&2
  exit 1
fi

iso_hash="$(
  printf '%s\n' "${sums}" \
    | awk -v name="${iso_name}" '$2 == name { print $1; exit }'
)"

if [ -z "${iso_hash}" ]; then
  echo "error: could not find checksum for ${iso_name} in ${SUMS_URL}" >&2
  exit 1
fi

debian_version="$(
  printf '%s\n' "${iso_name}" \
    | sed -n "s/^debian-\\([0-9.]*\\)-${ARCH}-netinst\\.iso$/\\1/p"
)"

if [ -z "${debian_version}" ]; then
  echo "error: could not parse Debian version from ${iso_name}" >&2
  exit 1
fi

iso_url="${BASE_URL}/${iso_name}"
iso_checksum="sha256:${iso_hash}"

printf 'Using Debian %s (%s).\n' "${debian_version}" "${ARCH}" >&2

case "${MODE}" in
  export)
    printf "export PKR_VAR_iso_url=%q\n" "${iso_url}"
    printf "export PKR_VAR_iso_checksum=%q\n" "${iso_checksum}"
    printf "export PKR_VAR_debian_version=%q\n" "${debian_version}"
    ;;
  hcl)
    printf 'iso_url      = "%s"\n' "${iso_url}"
    printf 'iso_checksum = "%s"\n' "${iso_checksum}"
    ;;
esac
