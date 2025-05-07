#!/bin/sh
# Copyright The TrustTunnel Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

ROOTFS_DIR=$1

# Remove old ssh key.
rm -f /root/.ssh/id_rsa*

# Generate new ssh key.
ssh-keygen -t rsa -f /root/.ssh/id_rsa_trust_tunnel_agent -N "" -C "trust-tunnel-agent" -q

# Update authorized_keys.
update_authorized_keys() {
    user_dir=$1
    if [ -f "${ROOTFS_DIR}${user_dir}/.ssh/authorized_keys" ]; then
        sed -i "/trust-tunnel-agent/d" "${ROOTFS_DIR}${user_dir}/.ssh/authorized_keys"
        cat /root/.ssh/id_rsa_trust_tunnel_agent.pub >> "${ROOTFS_DIR}${user_dir}/.ssh/authorized_keys"
    fi
}

# Update authorized_keys for root, log and admin.
update_authorized_keys "/root"
update_authorized_keys "/home/log"
update_authorized_keys "/home/admin"

