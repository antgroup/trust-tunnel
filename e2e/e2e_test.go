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
	"os/exec"
	"testing"
	"time"
	"trust-tunnel/pkg/common/sessionutil"

	"github.com/docker/docker/api/types/container"
)

const (
	host             = "localhost"
	port             = "5006"
	dockerAPIVersion = "1.40"
)

type Config struct {
	Host             string
	Port             string
	TargetType       string
	ContainerID      string
	LoginName        string
	DisableCleanMode string
	Cmd              string
}

func TestExecCmdInPhy(t *testing.T) {
	cli, err := sessionutil.CreateDockerClient("unix:///var/run/docker.sock", dockerAPIVersion)
	if err != nil {
		t.Fatalf("Failed to create Docker client: %v", err)
	}
	cid, err := startTrustTunnelAgent(cli, "./config/config.toml")
	if err != nil {
		t.Fatalf("Failed to run trust-tunnel-agent: %v", err)
	}
	defer func() {
		if err := cli.ContainerRemove(context.Background(), cid, container.RemoveOptions{Force: true}); err != nil {
			t.Fatalf("Failed to remove container: %v", err)
		}
	}()
	time.Sleep(10 * time.Second)

	tests := []struct {
		name             string
		host             string
		port             string
		targetType       string
		loginName        string
		disableCleanMode string
		cmd              string
	}{
		{
			name:             "Test with clean mode",
			host:             host,
			port:             port,
			targetType:       "phys",
			loginName:        "root",
			disableCleanMode: "false",
			cmd:              "ls -l",
		},
		{
			name:             "Test with disable clean mode",
			host:             host,
			port:             port,
			targetType:       "phys",
			loginName:        "root",
			disableCleanMode: "true",
			cmd:              "ls -l",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Config{
				Host:             tt.host,
				Port:             tt.port,
				TargetType:       tt.targetType,
				LoginName:        tt.loginName,
				DisableCleanMode: tt.disableCleanMode,
				Cmd:              tt.cmd,
			}

			output, err := execCmdWithClient(c)
			if err != nil {
				t.Fatalf("exec cmd in host error: %v", err)
			}
			if output == "" {
				t.Fatalf("Failed to run trust-tunnel-client: %v", err)
			}
		})
	}
}

func TestExecCmdInContainer(t *testing.T) {
	cli, err := sessionutil.CreateDockerClient("unix:///var/run/docker.sock", "1.40")
	if err != nil {
		t.Fatalf("Failed to create Docker client: %v", err)
	}
	agentContainerCid, err := startTrustTunnelAgent(cli, "./config/config.toml")
	if err != nil {
		t.Fatalf("Failed to run trust-tunnel-agent: %v", err)
	}

	targetContainerCid, err := startTargetContainer(cli)
	if err != nil {
		t.Fatalf("failed to run target container: %v", err)
	}

	defer func() {
		if err := cli.ContainerRemove(context.Background(), agentContainerCid, container.RemoveOptions{Force: true}); err != nil {
			t.Fatalf("Failed to remove container: %v", err)
		}
		if err := cli.ContainerRemove(context.Background(), targetContainerCid, container.RemoveOptions{Force: true}); err != nil {
			t.Fatalf("Failed to remove container: %v", err)
		}
	}()
	time.Sleep(10 * time.Second)

	tests := []struct {
		name             string
		host             string
		port             string
		targetType       string
		cid              string
		loginName        string
		disableCleanMode string
		cmd              string
	}{
		{
			name:             "Test with clean mode",
			host:             host,
			port:             port,
			targetType:       "container",
			cid:              targetContainerCid,
			loginName:        "root",
			disableCleanMode: "false",
			cmd:              "ls -l",
		},
		{
			name:             "Test with disable clean mode",
			host:             host,
			port:             port,
			targetType:       "container",
			cid:              targetContainerCid,
			loginName:        "root",
			disableCleanMode: "true",
			cmd:              "ls -l",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Config{
				Host:             tt.host,
				Port:             tt.port,
				TargetType:       tt.targetType,
				ContainerID:      tt.cid,
				LoginName:        tt.loginName,
				DisableCleanMode: tt.disableCleanMode,
				Cmd:              tt.cmd,
			}

			output, err := execCmdWithClient(c)
			if err != nil {
				t.Fatalf("exec cmd in container error: %v", err)
			}
			if output == "" {
				t.Fatalf("Failed to run trust-tunnel-client: %v", err)
			}
		})
	}
}

func execCmdWithClient(c Config) (string, error) {
	var clientCmd *exec.Cmd
	disableCleanModeFlag := "--disable-clean-mode" + "=" + c.DisableCleanMode
	if c.ContainerID != "" {
		clientCmd = exec.Command("../out/trust-tunnel-client", "--host", c.Host, "--port", c.Port, "--type", c.TargetType,
			"--login-name", c.LoginName, "--cid", c.ContainerID, disableCleanModeFlag, "sh", "-c", c.Cmd)
	} else {
		clientCmd = exec.Command("../out/trust-tunnel-client", "--host", c.Host, "--port", c.Port, "--type", c.TargetType,
			"--login-name", c.LoginName, disableCleanModeFlag, "sh", "-c", c.Cmd)
	}

	stdin, err := clientCmd.StdinPipe()
	if err != nil {
		return "", err
	}
	defer stdin.Close()

	clientOutput, err := clientCmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return string(clientOutput), nil
}
