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

//go:build linux
// +build linux

package e2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

const (
	targetContainerImage = "docker.m.daocloud.io/ubuntu:latest"
	targetContainerName  = "trust-tunnel-target-test"
	agentImage           = "trust-tunnel-agent"
	agentContainerName   = "trust-tunnel-agent-test"
)

// removeContainerIfExists removes a container with the given name if it exists.
func removeContainerIfExists(cli *client.Client, containerName string) error {
	containers, err := cli.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		return fmt.Errorf("failed to list containers: %v", err)
	}

	for _, c := range containers {
		if c.Names[0] == "/"+containerName {
			if err := cli.ContainerRemove(context.Background(), c.ID, container.RemoveOptions{Force: true}); err != nil {
				return fmt.Errorf("failed to remove container: %v", err)
			}
			break
		}
	}

	return nil
}

// startTrustTunnelAgent starts a trust-tunnel-agent container.
func startTrustTunnelAgent(cli *client.Client, configFile string) (string, error) {
	containerConfig := &container.Config{
		Image: agentImage,
		Tty:   false,
	}

	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("error getting the working directory: %v", err)
	}

	configFilePath := filepath.Join(dir, configFile)
	configFileBind := configFilePath + ":" + "/home/trust-tunnel/config/config.toml"

	hostConfig := &container.HostConfig{
		RestartPolicy: container.RestartPolicy{Name: "on-failure"},
		Privileged:    true,
		PidMode:       "host",
		NetworkMode:   "host",
		Binds: []string{
			"/var/run:/var/run-mount",
			"/home:/rootfs/home",
			"/root:/rootfs/root",
			"/etc/passwd:/rootfs/etc/passwd",
			configFileBind,
		},
	}

	// If a container with the same name exists, remove it.
	if err := removeContainerIfExists(cli, agentContainerName); err != nil {
		return "", fmt.Errorf("failed to remove container: %v", err)
	}

	resp, err := cli.ContainerCreate(context.Background(), containerConfig, hostConfig, nil, nil, agentContainerName)
	if err != nil {
		return "", fmt.Errorf("create container err:%v", err)
	}

	if err := cli.ContainerStart(context.Background(), resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start container: %v", err)
	}
	return resp.ID, nil
}

// startTargetContainer starts a target container.
func startTargetContainer(cli *client.Client) (string, error) {
	_, err := cli.ImagePull(context.Background(), targetContainerImage, image.PullOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to pull image: %v", err)
	}

	err = removeContainerIfExists(cli, targetContainerName)
	if err != nil {
		return "", fmt.Errorf("failed to remove container: %v", err)
	}

	containerConfig := &container.Config{
		Image: targetContainerImage,
		Tty:   false,
		Cmd:   []string{"sleep", "36000"},
	}

	hostConfig := &container.HostConfig{}

	// Create the container.
	resp, err := cli.ContainerCreate(context.Background(), containerConfig, hostConfig, nil, nil, targetContainerName)
	if err != nil {
		return "", fmt.Errorf("create container err:%v", err)
	}

	// Start the created container.
	if err := cli.ContainerStart(context.Background(), resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start container: %v", err)
	}

	// Return the ID of the successfully started container.
	return resp.ID, nil
}
