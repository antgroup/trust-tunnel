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
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"trust-tunnel/pkg/common/sessionutil"
	"trust-tunnel/pkg/trust-tunnel-agent/sidecar"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

// stdType is the type of standard stream
// a writer can multiplex to.
type stdType byte

const (
	// stdin represents standard input stream type.
	stdin stdType = iota
	// stdout represents standard output stream type.
	stdout
	// stderr represents standard error steam type.
	stderr
)

// ContainerRuntime defines the container runtime.
type ContainerRuntime string

const (
	Docker     ContainerRuntime = "docker"
	Containerd ContainerRuntime = "containerd"
	bufferSize                  = 4096
)

const (
	stdWriterPrefixLen = 8
	stdWriterFdIndex   = 0
	stdWriterSizeIndex = 4

	// DefaultCPUs defines the default cpu resource limitation.
	DefaultCPUs = 1 // 1 CPU

	// DefaultMemoryMB defines the default memory resource limitation.
	DefaultMemoryMB = 512 // 512MB
)

type dockerSession struct {
	ctx       context.Context
	client    client.CommonAPIClient
	respID    string
	isExec    bool
	conn      net.Conn
	reader    *bufio.Reader
	tty       bool
	stdoutCh  chan io.Reader
	stderrCh  chan io.Reader
	sidecarID string

	stdoutDone chan struct{}
	stderrDone chan struct{}

	lock sync.Mutex
}

func (s *dockerSession) NextStdin() (io.WriteCloser, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.conn == nil {
		return nil, io.EOF
	}

	return s.conn, nil
}

func (s *dockerSession) NextStdout() (io.Reader, error) {
	r, ok := <-s.stdoutCh
	if !ok {
		return nil, io.EOF
	}

	return r, nil
}

func (s *dockerSession) NextStderr() (io.Reader, error) {
	r, ok := <-s.stderrCh
	if !ok {
		return nil, io.EOF
	}

	return r, nil
}

func (s *dockerSession) StderrDone() error {
	s.stderrDone <- struct{}{}

	return nil
}

func (s *dockerSession) StdoutDone() error {
	s.stdoutDone <- struct{}{}

	return nil
}

func (s *dockerSession) Clean() error {
	s.lock.Lock()
	s.conn.Close()
	s.conn = nil
	s.lock.Unlock()

	err := s.cleanLegacyProcess(s.isExec)
	if err != nil && !strings.Contains(err.Error(), "process already finished") {
		logger.Errorf("kill legacy process err:%v", err)
	}

	if !s.isExec {
		// Remove sidecar container.
		err := s.client.ContainerRemove(context.Background(), s.respID, container.RemoveOptions{Force: true})
		if err != nil {
			logger.WithField("container", s.respID).Errorf("remove container error: %v", err)

			return err
		}

		logger.WithField("container", s.respID).Infof("remove container done")
	}

	return nil
}

func (s *dockerSession) Resize(h, w int) error {
	logger.Debugf("resize to %d*%d", h, w)

	if s.isExec {
		return s.client.ContainerExecResize(s.ctx, s.respID, container.ResizeOptions{
			Height: uint(h),
			Width:  uint(w),
		})
	}

	return s.client.ContainerResize(s.ctx, s.respID, container.ResizeOptions{
		Height: uint(h),
		Width:  uint(w),
	})
}

func (s *dockerSession) ExitCode() int {
	<-s.stdoutDone
	<-s.stderrDone

	ctx := context.Background()

	if s.isExec {
		inspect, err := s.client.ContainerExecInspect(ctx, s.respID)
		if err != nil {
			logger.WithError(err).Errorf("failed to wait container %s", s.respID)

			return 0
		}

		return inspect.ExitCode
	}

	statusCode, err := waitContainer(s.client, s.respID)
	if err != nil {
		logger.Errorf("wait container error: %s", err.Error())

		return 0
	}

	return statusCode
}

// establishDockerSession creates a new Docker session based on the given configuration.
func establishDockerSession(c *Config, containerClient client.CommonAPIClient) (*dockerSession, error) {
	if containerClient == nil {
		return nil, fmt.Errorf("container Client is nil")
	}

	var s *dockerSession

	var loginDir string

	var err error

	if c.LoginName != "" {
		_, _, loginDir, err = sessionutil.GetUserInfo(c.LoginName, c.RootfsPrefix+"/etc/passwd")
		if err != nil {
			return nil, fmt.Errorf(sessionutil.WrapContainerError(err.Error(), c.ContainerID))
		}
	}

	if len(c.Cmd) > 0 {
		c.Cmd[len(c.Cmd)-1] = "cd " + loginDir + ";" + c.Cmd[len(c.Cmd)-1]
	}

	// If clean mode is disabled, exec into the container directly.
	if c.DisableCleanMode {
		logger.WithFields(logrus.Fields{"disable-clean-mode": c.DisableCleanMode}).
			Infof("exec into container %s directly", c.ContainerID)

		s, err = execContainer(c, containerClient)
	} else {
		// Otherwise, attach a sidecar to the container and execute the command using nsenter inside it.
		logger.WithFields(logrus.Fields{"disable-clean-mode": c.DisableCleanMode}).
			Infof("attach sidecar to container %s", c.ContainerID)

		s, err = attachSidecar(c, containerClient)
	}

	if err != nil {
		return nil, fmt.Errorf(sessionutil.WrapContainerError(err.Error(), c.ContainerID))
	}

	go s.handleStreamOutput(!c.DisableCleanMode)

	return s, nil
}

// attachSidecar attaches a sidecar container to the given container and returns a new Docker session.
func attachSidecar(c *Config, apiClient client.CommonAPIClient) (*dockerSession, error) {
	ctx := context.Background()

	// Pull the sidecar image if it's not already present.
	image, err := sidecar.PullMissingImage(c.SidecarImage, c.ImageHubAuth, false, apiClient)
	if err != nil {
		return nil, err
	}

	if c.LoginName == "" {
		return nil, fmt.Errorf("empty login name isn't allowed")
	}

	// Build the command to execute inside the sidecar container.
	cmd := []string{"/superman.sh", "-u", c.LoginName}
	if c.LoginGroup != "" {
		cmd = append(cmd, "-g", c.LoginGroup)
	}

	cmd = append(cmd, c.Cmd...)

	// Configure the container to run the command inside the sidecar.
	contConfig := &container.Config{
		AttachStderr: true,
		AttachStdin:  true,
		AttachStdout: true,
		Cmd:          cmd,
		Env:          []string{"RequestedIP=0.0.0.0", "HOME=/home/" + c.LoginName},
		Entrypoint:   nil,
		Image:        image,
		OpenStdin:    c.Interactive,
		StdinOnce:    c.Interactive,
		Tty:          c.Tty,
	}
	logger.Infof("entering container with command: %v", contConfig.Cmd)

	// Validating the resource values.
	if c.Cpus <= 0 {
		c.Cpus = DefaultCPUs
	}

	if c.MemoryMB <= 0 {
		c.MemoryMB = DefaultMemoryMB
	}

	// Configure the host to run the sidecar container.
	hostConfig := &container.HostConfig{
		AutoRemove:  false,
		PidMode:     container.PidMode("container:" + c.ContainerID),
		NetworkMode: container.NetworkMode("container:" + c.ContainerID),
		Privileged:  true,
		Resources: container.Resources{
			CPUPeriod: 100000,
			CPUQuota:  int64(c.Cpus * 100000),
			Memory:    int64(c.MemoryMB) * 1024 * 1024,
		},
	}

	// Configure the container to run the command inside the sidecar.
	netConfig := &network.NetworkingConfig{}
	cname := ""

	// Create the sidecar container.
	createResp, err := apiClient.ContainerCreate(ctx, contConfig, hostConfig, netConfig, nil, cname)
	if err != nil {
		return nil, fmt.Errorf("create container exec error: %v", err)
	}

	attachOptions := container.AttachOptions{
		Stream: true,
		Stdin:  contConfig.AttachStdin,
		Stdout: contConfig.AttachStdout,
		Stderr: contConfig.AttachStderr,
	}
	// Attach to the sidecar container.
	resp, err := apiClient.ContainerAttach(ctx, createResp.ID, attachOptions)
	if err != nil {
		return nil, fmt.Errorf("attach to container error: %v", err)
	}

	// Start the sidecar container.
	if err = apiClient.ContainerStart(ctx, createResp.ID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("start container error: %v", err)
	}

	// Return a new Docker session for the sidecar container.
	return &dockerSession{
		ctx:        ctx,
		client:     apiClient,
		respID:     createResp.ID,
		isExec:     false,
		conn:       resp.Conn,
		reader:     resp.Reader,
		tty:        c.Tty,
		stdoutCh:   make(chan io.Reader, 64),
		stderrCh:   make(chan io.Reader, 64),
		stdoutDone: make(chan struct{}, 1),
		stderrDone: make(chan struct{}, 1),
		sidecarID:  createResp.ID,
	}, nil
}

// execContainer executes the given command inside the given container using the way of 'docker exec',
// returns a new Docker session.
func execContainer(c *Config, apiClient client.CommonAPIClient) (*dockerSession, error) {
	ctx := context.Background()

	// Configure the exec config.
	createExecConfig := types.ExecConfig{
		Cmd:          c.Cmd,
		Tty:          c.Tty,
		AttachStderr: true,
		AttachStdout: true,
		AttachStdin:  c.Interactive,
		User:         c.LoginName,
	}

	createResp, err := apiClient.ContainerExecCreate(ctx, c.ContainerID, createExecConfig)
	if err != nil {
		return nil, fmt.Errorf("create container exec error: %v", err)
	}

	attachResp, err := apiClient.ContainerExecAttach(ctx, createResp.ID, types.ExecStartCheck{Tty: c.Tty})
	if err != nil {
		return nil, fmt.Errorf("start container exec error: %v", err)
	}

	return &dockerSession{
		ctx:        ctx,
		client:     apiClient,
		respID:     createResp.ID,
		isExec:     true,
		conn:       attachResp.Conn,
		reader:     attachResp.Reader,
		tty:        c.Tty,
		stdoutCh:   make(chan io.Reader, 64),
		stderrCh:   make(chan io.Reader, 64),
		stdoutDone: make(chan struct{}, 1),
		stderrDone: make(chan struct{}, 1),
	}, nil
}

// handleStreamOutput handles the output streaming of the session depending on whether it has a tty or is exec.
func (s *dockerSession) handleStreamOutput(exec bool) {
	// TTY case.
	if s.tty {
		s.streamUnifiedOutput()
	} else if exec {
		s.streamSplitOutput()
	} else {
		s.streamUnifiedOutput()
	}
}

// streamUnifiedOutput reads the output stream directly and sends it without distinguishing between stdout and stderr.
func (s *dockerSession) streamUnifiedOutput() {
	// The reader can be used directly.
	for {
		buf := make([]byte, bufferSize)

		n, err := s.reader.Read(buf)
		if n > 0 {
			reader := bytes.NewReader(buf[:n])
			s.stdoutCh <- reader
		}

		if err != nil {
			if err != io.EOF &&
				!strings.Contains(err.Error(), "use of closed network connection") {
				// connection is closed.
				logger.WithField("container", s.respID).Warnf("read container tty error: %v", err)
			}

			close(s.stdoutCh)

			close(s.stderrCh)

			return
		}
	}
}

// streamSplitOutput first reads and parses the header of the output,
// then sends the data to the corresponding channel based on the frame type (stdout or stderr).
func (s *dockerSession) streamSplitOutput() {
	for {
		var (
			metadata []byte
			err      error
		)
		// Peek will block until reader got some data or error occurs.
		metadata, err = s.reader.Peek(stdWriterPrefixLen)
		if err != nil {
			// Connection is closed.
			close(s.stdoutCh)
			close(s.stderrCh)

			return
		}

		s.reader.Discard(stdWriterPrefixLen)

		stream := stdType(metadata[stdWriterFdIndex])
		frameSize := int(binary.BigEndian.Uint32(metadata[stdWriterSizeIndex : stdWriterSizeIndex+4]))

		// Bytes that are already read.
		nr := 0

		for {
			var buffer []byte

			left := frameSize - nr
			if left <= 0 {
				break
			} else if left < bufferSize {
				buffer = make([]byte, left)
			} else {
				buffer = make([]byte, bufferSize)
			}

			n, err := io.ReadFull(s.reader, buffer)
			if err != nil {
				logger.WithField("container", s.respID).Errorf("pollout error: %v", err)

				return
			}

			if n <= 0 {
				continue
			}

			nr += n
			reader := bytes.NewReader(buffer[:n])
			// Check the first byte to know where to write.
			switch stream {
			case stdin:
				logger.WithField("container", s.respID).Errorf("got stdin output from exec connection")

				return
			case stdout:
				// Write on stdout.
				s.stdoutCh <- reader
			case stderr:
				// Write on stderr.
				s.stderrCh <- reader
			default:
				logger.WithField("container", s.respID).Errorf("Unrecognized input header: %d", stream)

				return
			}
		}
	}
}

// cleanLegacyProcess clean the legacy processes before session disconnects.
func (s *dockerSession) cleanLegacyProcess(isExec bool) error {
	if isExec {
		// Now clean legacy process only support sidecar scene.
		return nil
	}
	// Support the sidecar legacy process to kill.
	cid := s.sidecarID

	cont, err := s.client.ContainerInspect(context.Background(), cid)
	if err != nil {
		return err
	}

	pid := cont.State.Pid
	// Kill the children processes first.
	err = sessionutil.KillProcessGroup(pid, "/superman.sh", true)
	if err != nil && !strings.Contains(err.Error(), "process already finished") {
		return err
	}

	// Kill the process itself.
	return sessionutil.KillProcess(pid)
}

// waitContainer waits for the container to stop running and returns its exit status code.
func waitContainer(cli client.CommonAPIClient, containerID string) (int, error) {
	statusCh, errCh := cli.ContainerWait(context.Background(), containerID, container.WaitConditionNotRunning)

	for {
		select {
		case err := <-errCh:
			if err != nil {
				return 0, err
			}
		case status := <-statusCh:
			return int(status.StatusCode), nil
		}
	}
}
