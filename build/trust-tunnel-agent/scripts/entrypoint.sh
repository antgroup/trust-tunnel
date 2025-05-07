#! /bin/bash
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

set -e
set -x

config_file="$1"
if [[ ! -f "$config_file" ]]; then
    echo "Config file not found: $config_file"
    exit 1
fi


# Parse config.toml,get the value of rootfs_prefix.
ROOT_FS=$(grep rootfs_prefix "$config_file" | awk -F '=' '{print $2}' | sed 's/ //g' | sed 's/\"//g')

# Generate ssh key for ssh physical tunnel.
/home/trust-tunnel/gen_login_key.sh "$ROOT_FS"

# Start trust-tunnel-agent.
/home/trust-tunnel/trust-tunnel-agent -c "$config_file"
