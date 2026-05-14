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
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// DockerClient defines the interface for Docker client operations used by session.
// This interface allows for easier testing by mocking the Docker client.
type DockerClient interface {
	// ContainerCreate creates a new container.
	ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig,
		networkingConfig *network.NetworkingConfig, platform *v1.Platform, name string) (container.CreateResponse, error)

	// ContainerAttach attaches to a container.
	ContainerAttach(ctx context.Context, containerID string, options container.AttachOptions) (types.HijackedResponse, error)

	// ContainerStart starts a container.
	ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error

	// ContainerRemove removes a container.
	ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error

	// ContainerResize resizes a container TTY.
	ContainerResize(ctx context.Context, containerID string, options container.ResizeOptions) error

	// ContainerExecCreate creates an exec instance.
	ContainerExecCreate(ctx context.Context, containerID string, config types.ExecConfig) (types.IDResponse, error)

	// ContainerExecAttach attaches to an exec instance.
	ContainerExecAttach(ctx context.Context, execID string, config types.ExecStartCheck) (types.HijackedResponse, error)

	// ContainerExecResize resizes an exec instance TTY.
	ContainerExecResize(ctx context.Context, execID string, options container.ResizeOptions) error

	// ContainerExecInspect returns information about an exec instance.
	ContainerExecInspect(ctx context.Context, execID string) (types.ContainerExecInspect, error)

	// ContainerWait waits for a container to stop.
	ContainerWait(ctx context.Context, containerID string, condition container.WaitCondition) (<-chan container.WaitResponse, <-chan error)

	// ContainerInspect returns container information.
	ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error)

	// ImageInspectWithRaw returns image information.
	ImageInspectWithRaw(ctx context.Context, imageID string) (types.ImageInspect, []byte, error)

	// ImagePull pulls an image from the registry.
	ImagePull(ctx context.Context, image string, options types.ImagePullOptions) (io.ReadCloser, error)

	// ContainerList returns the list of containers.
	ContainerList(ctx context.Context, options container.ListOptions) ([]types.Container, error)
}

// DockerClientFunc is an adapter to allow using functions as DockerClient.
type DockerClientFunc struct {
	ContainerCreateFunc      func(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *v1.Platform, name string) (container.CreateResponse, error)
	ContainerAttachFunc      func(ctx context.Context, containerID string, options container.AttachOptions) (types.HijackedResponse, error)
	ContainerStartFunc       func(ctx context.Context, containerID string, options container.StartOptions) error
	ContainerRemoveFunc      func(ctx context.Context, containerID string, options container.RemoveOptions) error
	ContainerResizeFunc      func(ctx context.Context, containerID string, options container.ResizeOptions) error
	ContainerExecCreateFunc  func(ctx context.Context, containerID string, config types.ExecConfig) (types.IDResponse, error)
	ContainerExecAttachFunc  func(ctx context.Context, execID string, config types.ExecStartCheck) (types.HijackedResponse, error)
	ContainerExecResizeFunc  func(ctx context.Context, execID string, options container.ResizeOptions) error
	ContainerExecInspectFunc func(ctx context.Context, execID string) (types.ContainerExecInspect, error)
	ContainerWaitFunc        func(ctx context.Context, containerID string, condition container.WaitCondition) (<-chan container.WaitResponse, <-chan error)
	ContainerInspectFunc     func(ctx context.Context, containerID string) (types.ContainerJSON, error)
	ImageInspectWithRawFunc  func(ctx context.Context, imageID string) (types.ImageInspect, []byte, error)
	ImagePullFunc            func(ctx context.Context, image string, options types.ImagePullOptions) (io.ReadCloser, error)
	ContainerListFunc        func(ctx context.Context, options container.ListOptions) ([]types.Container, error)
}

// ContainerCreate implements DockerClient.
func (f *DockerClientFunc) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *v1.Platform, name string) (container.CreateResponse, error) {
	if f.ContainerCreateFunc == nil {
		return container.CreateResponse{}, nil
	}
	return f.ContainerCreateFunc(ctx, config, hostConfig, networkingConfig, platform, name)
}

// ContainerAttach implements DockerClient.
func (f *DockerClientFunc) ContainerAttach(ctx context.Context, containerID string, options container.AttachOptions) (types.HijackedResponse, error) {
	if f.ContainerAttachFunc == nil {
		return types.HijackedResponse{}, nil
	}
	return f.ContainerAttachFunc(ctx, containerID, options)
}

// ContainerStart implements DockerClient.
func (f *DockerClientFunc) ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error {
	if f.ContainerStartFunc == nil {
		return nil
	}
	return f.ContainerStartFunc(ctx, containerID, options)
}

// ContainerRemove implements DockerClient.
func (f *DockerClientFunc) ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error {
	if f.ContainerRemoveFunc == nil {
		return nil
	}
	return f.ContainerRemoveFunc(ctx, containerID, options)
}

// ContainerResize implements DockerClient.
func (f *DockerClientFunc) ContainerResize(ctx context.Context, containerID string, options container.ResizeOptions) error {
	if f.ContainerResizeFunc == nil {
		return nil
	}
	return f.ContainerResizeFunc(ctx, containerID, options)
}

// ContainerExecCreate implements DockerClient.
func (f *DockerClientFunc) ContainerExecCreate(ctx context.Context, containerID string, config types.ExecConfig) (types.IDResponse, error) {
	if f.ContainerExecCreateFunc == nil {
		return types.IDResponse{}, nil
	}
	return f.ContainerExecCreateFunc(ctx, containerID, config)
}

// ContainerExecAttach implements DockerClient.
func (f *DockerClientFunc) ContainerExecAttach(ctx context.Context, execID string, config types.ExecStartCheck) (types.HijackedResponse, error) {
	if f.ContainerExecAttachFunc == nil {
		return types.HijackedResponse{}, nil
	}
	return f.ContainerExecAttachFunc(ctx, execID, config)
}

// ContainerExecResize implements DockerClient.
func (f *DockerClientFunc) ContainerExecResize(ctx context.Context, execID string, options container.ResizeOptions) error {
	if f.ContainerExecResizeFunc == nil {
		return nil
	}
	return f.ContainerExecResizeFunc(ctx, execID, options)
}

// ContainerExecInspect implements DockerClient.
func (f *DockerClientFunc) ContainerExecInspect(ctx context.Context, execID string) (types.ContainerExecInspect, error) {
	if f.ContainerExecInspectFunc == nil {
		return types.ContainerExecInspect{}, nil
	}
	return f.ContainerExecInspectFunc(ctx, execID)
}

// ContainerWait implements DockerClient.
func (f *DockerClientFunc) ContainerWait(ctx context.Context, containerID string, condition container.WaitCondition) (<-chan container.WaitResponse, <-chan error) {
	if f.ContainerWaitFunc == nil {
		return nil, nil
	}
	return f.ContainerWaitFunc(ctx, containerID, condition)
}

// ContainerInspect implements DockerClient.
func (f *DockerClientFunc) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	if f.ContainerInspectFunc == nil {
		return types.ContainerJSON{}, nil
	}
	return f.ContainerInspectFunc(ctx, containerID)
}

// ImageInspectWithRaw implements DockerClient.
func (f *DockerClientFunc) ImageInspectWithRaw(ctx context.Context, imageID string) (types.ImageInspect, []byte, error) {
	if f.ImageInspectWithRawFunc == nil {
		return types.ImageInspect{}, nil, nil
	}
	return f.ImageInspectWithRawFunc(ctx, imageID)
}

// ImagePull implements DockerClient.
func (f *DockerClientFunc) ImagePull(ctx context.Context, image string, options types.ImagePullOptions) (io.ReadCloser, error) {
	if f.ImagePullFunc == nil {
		return nil, nil
	}
	return f.ImagePullFunc(ctx, image, options)
}

// ContainerList implements DockerClient.
func (f *DockerClientFunc) ContainerList(ctx context.Context, options container.ListOptions) ([]types.Container, error) {
	if f.ContainerListFunc == nil {
		return nil, nil
	}
	return f.ContainerListFunc(ctx, options)
}
