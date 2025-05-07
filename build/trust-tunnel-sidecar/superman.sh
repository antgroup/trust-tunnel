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

set -e

usage() {
    echo "Usage:"
    echo "superman.sh -u USER [-g GROUP] command"
    echo "Description:"
    echo "USER: user name for the command."
    echo "GROUP: group name for the command."
    exit 255
}

# Parse options.
while getopts 'u:g:' OPT; do
    case $OPT in
        u) user="$OPTARG";;
        g) group="$OPTARG";;
        ?) usage;;
    esac
done

# Shift to get left part as command.
shift $((OPTIND - 1))

# User must set, print usage and exit.
[ "$user"x = ""x ] && usage

# If group isn't set from user's input.
if [ "$group"x = ""x ]
then
	if [ "$user"x = "log"x ]
	then
		# Set group to "admin" for user "log".
		group="admin"
	else
		group="$user"
	fi
fi

# Get uid and gid from user name.
uid=$(nsenter -t 1 -m id -u "$user")
gid=$(nsenter -t 1 -m getent group "${group}" | cut -d: -f3)

# Execute command with user's uid and gid.
nsenter -t 1 -m -u -i -n -p -S "$uid" -G "$gid" "$@"
