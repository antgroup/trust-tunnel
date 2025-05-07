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

FROM golang:1.21 as builder

COPY . /go/src/trust-tunnel

WORKDIR /go/src/trust-tunnel

RUN make trust-tunnel-agent

# Set the base image.
FROM ubuntu

# Install ssh-keygen.
RUN apt-get update && apt-get install -y openssh-client

# Copy files.
COPY --from=builder /go/src/trust-tunnel/out/trust-tunnel-agent /home/trust-tunnel/trust-tunnel-agent
COPY ./config/ /home/trust-tunnel/config/
COPY ./build/trust-tunnel-agent/scripts/entrypoint.sh /home/trust-tunnel/entrypoint.sh
COPY ./build/trust-tunnel-agent/scripts/gen_login_key.sh /home/trust-tunnel/gen_login_key.sh
RUN chmod +x /home/trust-tunnel/entrypoint.sh
RUN chmod +x /home/trust-tunnel/gen_login_key.sh

# Set the default command to execute when creating a new container.
CMD ["/home/trust-tunnel/entrypoint.sh", "/home/trust-tunnel/config/config.toml"]
