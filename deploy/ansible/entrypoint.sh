#!/usr/bin/env sh
set -eu

# Use the Ansible configuration that belongs to this project.
# The file disables host key checking for this local Raspberry Pi lab so password-based SSH works from the ephemeral control container.
export ANSIBLE_CONFIG="/workspace/deploy/ansible/ansible.cfg"
export ANSIBLE_HOST_KEY_CHECKING="False"

playbook="deploy/ansible/playbook.yml"
if [ "$#" -gt 0 ] && [ -f "$1" ]; then
  playbook="$1"
  shift
fi

inventory="deploy/ansible/inventory.ini"
if [ -f "deploy/ansible/secrets/local-inventory.yml" ]; then
  inventory="deploy/ansible/secrets/local-inventory.yml"
fi

exec ansible-playbook -i "$inventory" "$playbook" "$@"
