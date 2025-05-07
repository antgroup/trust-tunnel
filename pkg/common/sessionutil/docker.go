// Copyright The TrustTunnel Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sessionutil

import dockerClient "github.com/docker/docker/client"

// CreateDockerClient Creates a Docker client based on the given socket endpoint and docker api version.
func CreateDockerClient(endpoint string, apiVersion string) (*dockerClient.Client, error) {
	cli, err := dockerClient.NewClientWithOpts(dockerClient.WithHost(endpoint), dockerClient.WithVersion(apiVersion))
	if err != nil {
		return nil, err
	}

	return cli, nil
}
