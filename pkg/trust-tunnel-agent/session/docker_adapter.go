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

package session

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	dockerClient "github.com/docker/docker/client"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// DockerClientAdapter wraps docker client to implement DockerClient interface.
type DockerClientAdapter struct {
	client dockerClient.CommonAPIClient
}

// NewDockerClientAdapter creates a new DockerClientAdapter.
func NewDockerClientAdapter(client dockerClient.CommonAPIClient) DockerClient {
	return &DockerClientAdapter{client: client}
}

// ContainerCreate implements DockerClient.
func (d *DockerClientAdapter) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig,
	networkingConfig *network.NetworkingConfig, platform *v1.Platform, name string,
) (container.CreateResponse, error) {
	return d.client.ContainerCreate(ctx, config, hostConfig, networkingConfig, platform, name)
}

// ContainerAttach implements DockerClient.
func (d *DockerClientAdapter) ContainerAttach(ctx context.Context, containerID string, options container.AttachOptions) (types.HijackedResponse, error) {
	return d.client.ContainerAttach(ctx, containerID, options)
}

// ContainerStart implements DockerClient.
func (d *DockerClientAdapter) ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error {
	return d.client.ContainerStart(ctx, containerID, options)
}

// ContainerRemove implements DockerClient.
func (d *DockerClientAdapter) ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error {
	return d.client.ContainerRemove(ctx, containerID, options)
}

// ContainerResize implements DockerClient.
func (d *DockerClientAdapter) ContainerResize(ctx context.Context, containerID string, options container.ResizeOptions) error {
	return d.client.ContainerResize(ctx, containerID, options)
}

// ContainerExecCreate implements DockerClient.
func (d *DockerClientAdapter) ContainerExecCreate(ctx context.Context, containerID string, config types.ExecConfig) (types.IDResponse, error) {
	return d.client.ContainerExecCreate(ctx, containerID, config)
}

// ContainerExecAttach implements DockerClient.
func (d *DockerClientAdapter) ContainerExecAttach(ctx context.Context, execID string, config types.ExecStartCheck) (types.HijackedResponse, error) {
	return d.client.ContainerExecAttach(ctx, execID, config)
}

// ContainerExecResize implements DockerClient.
func (d *DockerClientAdapter) ContainerExecResize(ctx context.Context, execID string, options container.ResizeOptions) error {
	return d.client.ContainerExecResize(ctx, execID, options)
}

// ContainerExecInspect implements DockerClient.
func (d *DockerClientAdapter) ContainerExecInspect(ctx context.Context, execID string) (types.ContainerExecInspect, error) {
	return d.client.ContainerExecInspect(ctx, execID)
}

// ContainerWait implements DockerClient.
func (d *DockerClientAdapter) ContainerWait(ctx context.Context, containerID string, condition container.WaitCondition) (<-chan container.WaitResponse, <-chan error) {
	return d.client.ContainerWait(ctx, containerID, condition)
}

// ContainerInspect implements DockerClient.
func (d *DockerClientAdapter) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	return d.client.ContainerInspect(ctx, containerID)
}

// ImageInspectWithRaw implements DockerClient.
func (d *DockerClientAdapter) ImageInspectWithRaw(ctx context.Context, imageID string) (types.ImageInspect, []byte, error) {
	return d.client.ImageInspectWithRaw(ctx, imageID)
}

// ImagePull implements DockerClient.
func (d *DockerClientAdapter) ImagePull(ctx context.Context, image string, options types.ImagePullOptions) (io.ReadCloser, error) {
	return d.client.ImagePull(ctx, image, options)
}

// ContainerList implements DockerClient.
func (d *DockerClientAdapter) ContainerList(ctx context.Context, options container.ListOptions) ([]types.Container, error) {
	return d.client.ContainerList(ctx, options)
}
