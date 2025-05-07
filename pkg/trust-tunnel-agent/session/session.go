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
	"io"
	"trust-tunnel/pkg/common/logutil"

	dockerClient "github.com/docker/docker/client"
	client "trust-tunnel/pkg/trust-tunnel-client"

	"github.com/containerd/containerd"
)

var logger = logutil.GetLogger("trust-tunnel-agent-session")

// Config defines the configuration for establishing a session.
type Config struct {
	// TargetType specifies the type of target, which can be a container or a physical host.
	TargetType client.TargetType

	// RootfsPrefix specifies the prefix of the root file system mounted in the container.
	RootfsPrefix string

	// ContainerID specifies the ID of the target container.
	ContainerID string

	// SidecarImage specifies the image of the sidecar container.
	SidecarImage string

	// ImageHubAuth specifies the authentication information for the image hub.
	ImageHubAuth string

	// UserName specifies the username for the user's identity.
	UserName string

	// LoginName specifies the login name for the target to connect.
	LoginName string

	// LoginGroup specifies the login group for the target to connect.
	LoginGroup string

	// Cmd specifies the commands to be executed in the target.
	Cmd []string

	// Tty specifies whether the session should be a TTY session.
	Tty bool

	// Interactive specifies whether the session should be an interactive session.
	Interactive bool

	// PhysTunnel specifies the physical tunnel to be used for the session,'SSH' or 'nsenter'.
	PhysTunnel string

	// Disable clean mode means remote cmd will be executed via "docker exec" for container,
	// and "ssh" for physical host.
	DisableCleanMode bool

	// Cpus specifies the limit of CPUs to be used for the sidecar container.
	Cpus float64

	// MemoryMB specifies the limit of memory to be used for the sidecar container in megabytes.
	MemoryMB int

	// ContainerNamespace specifies the namespace of the container.
	// It is used in containerd session when get container info.
	ContainerNamespace string
}

type Session interface {
	// NextStdin returns the next standard input stream.
	NextStdin() (io.WriteCloser, error)

	// NextStdout returns the next standard output stream.
	NextStdout() (io.Reader, error)

	// NextStderr returns the next standard error stream.
	NextStderr() (io.Reader, error)

	// StdoutDone signals that the standard output stream is done.
	StdoutDone() error

	// StderrDone signals that the standard error stream is done.
	StderrDone() error

	// Clean cleans up the resources used by the session.
	Clean() error

	// Resize resizes the console.
	Resize(h, w int) error

	// ExitCode returns the exit code of the session.
	ExitCode() int
}

// ContainerConfig represents the configuration structure for container services.
// It includes various configuration details pertinent to the container runtime environment.
type ContainerConfig struct {
	// Endpoint is the API endpoint address of the container service.
	// This is used for communication with the container service.
	Endpoint string `toml:"endpoint"`

	// DockerAPIVersion is the version number compatible with the Docker API.
	// This ensures compatibility in communication with the Docker engine.
	DockerAPIVersion string `toml:"docker_api_version"`

	// RootfsPrefix specifies the prefix of the root file system mounted in the container.
	RootfsPrefix string `toml:"rootfs_prefix"`

	// ContainerRuntime specifies the container runtime being used.
	// Supported runtimes include Docker, Containerd, etc.
	ContainerRuntime ContainerRuntime `toml:"container_runtime"`

	// Namespace is the namespace for the container runtime.
	// This is used in containerd when getting the container info.
	Namespace string `toml:"namespace"`
}

// EstablishSession establishes a session based on targetType in the config,
// returns a physical session or a container session.
func EstablishSession(config *Config, apiClient dockerClient.CommonAPIClient, containerdClient *containerd.Client, containerRuntime ContainerRuntime) (Session, error) {
	if config.TargetType == client.TargetPhys {
		return establishPhysSession(config)
	}

	return establishContainerSession(config, apiClient, containerdClient, containerRuntime)
}

// establishPhysSession establishes a physical session and returns the session and an error if any.
func establishPhysSession(config *Config) (Session, error) {
	if config.PhysTunnel == "nsenter" && !config.DisableCleanMode {
		return establishNsenterSession(config)
	}

	// Default to use sshd.
	return establishSSHSession(config)
}

// establishContainerSession establishes a container session and returns the session and an error if any.
func establishContainerSession(config *Config, apiClient dockerClient.CommonAPIClient, containerdClient *containerd.Client, containerRuntime ContainerRuntime) (Session, error) {
	if containerRuntime == Docker {
		return establishDockerSession(config, apiClient)
	}

	return establishContainerdSession(config, containerdClient)
}
