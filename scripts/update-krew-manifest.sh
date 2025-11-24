#!/bin/bash
# SPDX-License-Identifier: Apache-2.0
# Copyright Authors of kubectl-mft
#
# This script updates the krew manifest file with the correct version and checksums.
# Usage: ./scripts/update-krew-manifest.sh <version>
# Example: ./scripts/update-krew-manifest.sh v0.2.0

set -euo pipefail

VERSION="${1:-}"

if [[ -z "$VERSION" ]]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 v0.2.0"
    exit 1
fi

# Remove 'v' prefix for version number (e.g., v0.1.0 -> 0.1.0)
VERSION_NUM="${VERSION#v}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
TEMPLATE_FILE="${PROJECT_ROOT}/plugins/mft.yaml.tmpl"
OUTPUT_FILE="${PROJECT_ROOT}/plugins/mft.yaml"
CHECKSUMS_URL="https://github.com/chez-shanpu/kubectl-mft/releases/download/${VERSION}/checksums.txt"

echo "Updating krew manifest for version ${VERSION}..."

# Download checksums
echo "Downloading checksums from ${CHECKSUMS_URL}..."
CHECKSUMS=$(curl -sL "${CHECKSUMS_URL}")

if [[ -z "$CHECKSUMS" ]]; then
    echo "Error: Failed to download checksums"
    exit 1
fi

# Extract checksums for each platform
get_checksum() {
    local pattern="$1"
    echo "$CHECKSUMS" | grep "$pattern" | awk '{print $1}'
}

SHA256_DARWIN_AMD64=$(get_checksum "darwin_amd64.tar.gz")
SHA256_DARWIN_ARM64=$(get_checksum "darwin_arm64.tar.gz")
SHA256_LINUX_AMD64=$(get_checksum "linux_amd64.tar.gz")
SHA256_LINUX_ARM64=$(get_checksum "linux_arm64.tar.gz")
SHA256_WINDOWS_AMD64=$(get_checksum "windows_amd64.zip")
SHA256_WINDOWS_ARM64=$(get_checksum "windows_arm64.zip")

# Verify all checksums were found
for var in SHA256_DARWIN_AMD64 SHA256_DARWIN_ARM64 SHA256_LINUX_AMD64 SHA256_LINUX_ARM64 SHA256_WINDOWS_AMD64 SHA256_WINDOWS_ARM64; do
    if [[ -z "${!var}" ]]; then
        echo "Error: Failed to extract checksum for ${var}"
        exit 1
    fi
done

echo "Checksums extracted:"
echo "  darwin_amd64: ${SHA256_DARWIN_AMD64}"
echo "  darwin_arm64: ${SHA256_DARWIN_ARM64}"
echo "  linux_amd64:  ${SHA256_LINUX_AMD64}"
echo "  linux_arm64:  ${SHA256_LINUX_ARM64}"
echo "  windows_amd64: ${SHA256_WINDOWS_AMD64}"
echo "  windows_arm64: ${SHA256_WINDOWS_ARM64}"

# Generate manifest from template
echo "Generating ${OUTPUT_FILE}..."

sed -e "s/\${VERSION}/${VERSION}/g" \
    -e "s/\${VERSION_NUM}/${VERSION_NUM}/g" \
    -e "s/\${SHA256_DARWIN_AMD64}/${SHA256_DARWIN_AMD64}/g" \
    -e "s/\${SHA256_DARWIN_ARM64}/${SHA256_DARWIN_ARM64}/g" \
    -e "s/\${SHA256_LINUX_AMD64}/${SHA256_LINUX_AMD64}/g" \
    -e "s/\${SHA256_LINUX_ARM64}/${SHA256_LINUX_ARM64}/g" \
    -e "s/\${SHA256_WINDOWS_AMD64}/${SHA256_WINDOWS_AMD64}/g" \
    -e "s/\${SHA256_WINDOWS_ARM64}/${SHA256_WINDOWS_ARM64}/g" \
    "${TEMPLATE_FILE}" > "${OUTPUT_FILE}"

echo "Successfully updated ${OUTPUT_FILE}"
