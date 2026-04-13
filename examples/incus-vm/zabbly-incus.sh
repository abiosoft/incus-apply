#!/usr/bin/env bash


KERNEL_DONE_FILE="/var/lib/zabbly-kernel-done"
DONE_FILE="/var/lib/zabbly-incus-done"

# check if kernel is done, if not, exit with an error message
if [ ! -f "$KERNEL_DONE_FILE" ]; then
    echo "Zabbly Kernel provisioning not completed yet. Please run zabbly-kernel.sh first."
    exit 0 # Exit with 0 to allow cloud-init to continue normally.
fi

# check if a custom file exists that marks it as done, if not, run the script
if [ -f "$DONE_FILE" ]; then
    echo "Zabbly Incus provisioning already completed. Skipping."
    exit 0
fi

set -euo pipefail

# Ensure apt doesn't block waiting for user input during provisioning
export DEBIAN_FRONTEND=noninteractive

# Install necessary packages, add Zabbly's GPG key and repository, then install Incus.
apt update && apt install -y curl gpg debian-keyring debian-archive-keyring apt-transport-https
mkdir -p /etc/apt/keyrings/
curl -fsSL https://pkgs.zabbly.com/key.asc -o /etc/apt/keyrings/zabbly.asc
sh -c 'cat <<EOF > /etc/apt/sources.list.d/zabbly-incus-stable.sources
Enabled: yes
Types: deb
URIs: https://pkgs.zabbly.com/incus/stable
Suites: $(. /etc/os-release && echo ${VERSION_CODENAME})
Components: main
Architectures: $(dpkg --print-architecture)
Signed-By: /etc/apt/keyrings/zabbly.asc

EOF'
apt update
apt install -y incus incus-extra incus-ui-canonical

# Configure Incus with ZFS storage and trust the client certificate
incus admin init --auto --storage-backend=zfs --storage-pool=default --storage-create-loop=40
incus config trust add-certificate /root/install/client.crt && rm /root/install/client.crt
incus config set core.https_address :8443


# Mark the provisioning as done by creating a file
touch "$DONE_FILE"
echo "Zabbly Incus provisioning completed successfully."
