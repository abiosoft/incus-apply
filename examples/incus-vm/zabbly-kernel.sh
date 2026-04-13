#!/usr/bin/env bash

# ==============================================================================
# Zabbly Kernel & ZFS Provisioning Script
# Target: Debian 13 (Trixie)
# Sources: github.com/zabbly/linux & github.com/zabbly/zfs
# ==============================================================================

# Exit immediately if a command exits with a non-zero status
set -euo pipefail

# Define a file that indicates kernel provisioning is complete
KERNEL_DONE_FILE="/var/lib/zabbly-kernel-done"

# Ensure apt doesn't block waiting for user input during provisioning
export DEBIAN_FRONTEND=noninteractive

echo "==========================================================="
echo " Starting Zabbly Kernel & ZFS Installation"
echo "==========================================================="

echo "-> [1/5] Installing prerequisite packages..."
# Ensure cache is fresh and we have the tools needed to fetch the GPG key
apt-get update -y
apt-get install -y curl ca-certificates

echo "-> [2/5] Fetching Zabbly repository GPG key..."
# Create the keyrings directory if it doesn't already exist and fetch the key
mkdir -p /etc/apt/keyrings
curl -fsSL https://pkgs.zabbly.com/key.asc -o /etc/apt/keyrings/zabbly.asc

echo "-> [3/5] Configuring APT sources for Zabbly..."
# Dynamically fetch the OS codename (e.g., 'trixie' for Debian 13)
OS_CODENAME=$(. /etc/os-release && echo "${VERSION_CODENAME}")
SYSTEM_ARCH=$(dpkg --print-architecture)

# Create a single .sources file for both the mainline kernel and ZFS components
cat <<EOF > /etc/apt/sources.list.d/zabbly-kernel-stable.sources
Enabled: yes
Types: deb
URIs: https://pkgs.zabbly.com/kernel/stable
Suites: ${OS_CODENAME}
Components: main zfs
Architectures: ${SYSTEM_ARCH}
Signed-By: /etc/apt/keyrings/zabbly.asc
EOF

echo "-> [4/5] Updating APT package index with Zabbly sources..."
apt-get update -y

echo "-> [5/5] Installing Zabbly Kernel, OpenZFS and Btrfs..."
# Install the kernel metapackage alongside the Zabbly-specific OpenZFS packages
apt-get install -y \
    linux-zabbly \
    openzfs-zfsutils \
    openzfs-zfs-dkms \
    openzfs-zfs-initramfs \
    btrfs-progs

# Mark the kernel provisioning as done by creating a file
touch "$KERNEL_DONE_FILE"

echo "==========================================================="
echo " Installation completed successfully."
echo " A system reboot is required to boot into the new kernel."
echo "==========================================================="
