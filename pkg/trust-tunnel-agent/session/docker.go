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
)

const (
	// bufferSize is the buffer size for reading container output.
	bufferSize = 4096

	// streamChannelSize is the buffer size for stdout/stderr channels.
	streamChannelSize = 64
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

func (ds *dockerSession) NextStdin() (io.WriteCloser, error) {
	ds.lock.Lock()
	defer ds.lock.Unlock()

	if ds.conn == nil {
		return nil, io.EOF
	}

	return ds.conn, nil
}

func (ds *dockerSession) NextStdout() (io.Reader, error) {
	r, ok := <-ds.stdoutCh
	if !ok {
		return nil, io.EOF
	}

	return r, nil
}

func (ds *dockerSession) NextStderr() (io.Reader, error) {
	r, ok := <-ds.stderrCh
	if !ok {
		return nil, io.EOF
	}

	return r, nil
}

func (ds *dockerSession) StderrDone() error {
	ds.stderrDone <- struct{}{}

	return nil
}

func (ds *dockerSession) StdoutDone() error {
	ds.stdoutDone <- struct{}{}

	return nil
}

func (ds *dockerSession) Clean() error {
	ds.lock.Lock()
	ds.conn.Close()
	ds.conn = nil
	ds.lock.Unlock()

	err := ds.cleanLegacyProcess(ds.isExec)
	if err != nil && !strings.Contains(err.Error(), "process already finished") {
		logger.Errorf("kill legacy process err:%v", err)
	}

	if !ds.isExec {
		// Remove sidecar container.
		err := ds.client.ContainerRemove(context.Background(), ds.respID, container.RemoveOptions{Force: true})
		if err != nil {
			logger.WithField("container", ds.respID).Errorf("remove container error: %v", err)

			return err
		}

		logger.WithField("container", ds.respID).Infof("remove container done")
	}

	return nil
}

func (ds *dockerSession) Resize(h, w int) error {
	logger.Debugf("resize to %d*%d", h, w)

	if ds.isExec {
		return ds.client.ContainerExecResize(ds.ctx, ds.respID, container.ResizeOptions{
			Height: uint(h),
			Width:  uint(w),
		})
	}

	return ds.client.ContainerResize(ds.ctx, ds.respID, container.ResizeOptions{
		Height: uint(h),
		Width:  uint(w),
	})
}

func (ds *dockerSession) ExitCode() int {
	<-ds.stdoutDone
	<-ds.stderrDone

	ctx := context.Background()

	if ds.isExec {
		inspect, err := ds.client.ContainerExecInspect(ctx, ds.respID)
		if err != nil {
			logger.WithError(err).Errorf("failed to wait container %s", ds.respID)

			return 0
		}

		return inspect.ExitCode
	}

	statusCode, err := waitContainer(ds.client, ds.respID)
	if err != nil {
		logger.Errorf("wait container error: %s", err.Error())

		return 0
	}

	return statusCode
}

// establishDockerSession creates a new Docker session based on the given configuration.
func establishDockerSession(config *Config, containerClient client.CommonAPIClient) (*dockerSession, error) {
	if containerClient == nil {
		return nil, fmt.Errorf("container Client is nil")
	}

	var session *dockerSession
	var loginDir string
	var err error

	if config.LoginName != "" {
		_, _, loginDir, err = sessionutil.GetUserInfo(config.LoginName, config.RootfsPrefix+"/etc/passwd")
		if err != nil {
			return nil, fmt.Errorf("%s", sessionutil.WrapContainerError(err.Error(), config.ContainerID))
		}
	}

	if len(config.Cmd) > 0 {
		config.Cmd[len(config.Cmd)-1] = "cd " + loginDir + ";" + config.Cmd[len(config.Cmd)-1]
	}

	// If clean mode is disabled, exec into the container directly.
	if config.DisableCleanMode {
		logger.WithFields(logrus.Fields{"disable-clean-mode": config.DisableCleanMode}).
			Infof("exec into container %s directly", config.ContainerID)

		session, err = execContainer(config, containerClient)
	} else {
		// Otherwise, attach a sidecar to the container and execute the command using nsenter inside it.
		logger.WithFields(logrus.Fields{"disable-clean-mode": config.DisableCleanMode}).
			Infof("attach sidecar to container %s", config.ContainerID)

		session, err = attachSidecar(config, containerClient)
	}

	if err != nil {
		return nil, fmt.Errorf("%s", sessionutil.WrapContainerError(err.Error(), config.ContainerID))
	}

	go session.handleStreamOutput(!config.DisableCleanMode)

	return session, nil
}

// attachSidecar attaches a sidecar container to the given container and returns a new Docker session.
func attachSidecar(config *Config, apiClient client.CommonAPIClient) (*dockerSession, error) {
	ctx := context.Background()

	// Pull the sidecar image if it's not already present.
	image, err := sidecar.PullMissingImage(config.SidecarImage, config.ImageHubAuth, false, apiClient)
	if err != nil {
		return nil, err
	}

	if config.LoginName == "" {
		return nil, fmt.Errorf("empty login name isn't allowed")
	}

	// Build the command to execute inside the sidecar container.
	cmd := []string{"/superman.sh", "-u", config.LoginName}
	if config.LoginGroup != "" {
		cmd = append(cmd, "-g", config.LoginGroup)
	}

	cmd = append(cmd, config.Cmd...)

	// Configure the container to run the command inside the sidecar.
	contConfig := &container.Config{
		AttachStderr: true,
		AttachStdin:  true,
		AttachStdout: true,
		Cmd:          cmd,
		Env:          []string{"RequestedIP=0.0.0.0", "HOME=/home/" + config.LoginName},
		Entrypoint:   nil,
		Image:        image,
		OpenStdin:    config.Interactive,
		StdinOnce:    config.Interactive,
		Tty:          config.Tty,
	}
	logger.Infof("entering container with command: %v", contConfig.Cmd)

	// Validating the resource values.
	if config.Cpus <= 0 {
		config.Cpus = DefaultCPUs
	}

	if config.MemoryMB <= 0 {
		config.MemoryMB = DefaultMemoryMB
	}

	// Configure the host to run the sidecar container.
	hostConfig := &container.HostConfig{
		AutoRemove:  false,
		PidMode:     container.PidMode("container:" + config.ContainerID),
		NetworkMode: container.NetworkMode("container:" + config.ContainerID),
		Privileged:  true,
		Resources: container.Resources{
			CPUPeriod: 100000,
			CPUQuota:  int64(config.Cpus * 100000),
			Memory:    int64(config.MemoryMB) * 1024 * 1024,
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
		tty:        config.Tty,
		stdoutCh:   make(chan io.Reader, streamChannelSize),
		stderrCh:   make(chan io.Reader, streamChannelSize),
		stdoutDone: make(chan struct{}, 1),
		stderrDone: make(chan struct{}, 1),
		sidecarID:  createResp.ID,
	}, nil
}

// execContainer executes the given command inside the given container using the way of 'docker exec',
// returns a new Docker session.
func execContainer(config *Config, apiClient client.CommonAPIClient) (*dockerSession, error) {
	ctx := context.Background()

	// Configure the exec config.
	createExecConfig := types.ExecConfig{
		Cmd:          config.Cmd,
		Tty:          config.Tty,
		AttachStderr: true,
		AttachStdout: true,
		AttachStdin:  config.Interactive,
		User:         config.LoginName,
	}

	createResp, err := apiClient.ContainerExecCreate(ctx, config.ContainerID, createExecConfig)
	if err != nil {
		return nil, fmt.Errorf("create container exec error: %v", err)
	}

	attachResp, err := apiClient.ContainerExecAttach(ctx, createResp.ID, types.ExecStartCheck{Tty: config.Tty})
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
		tty:        config.Tty,
		stdoutCh:   make(chan io.Reader, streamChannelSize),
		stderrCh:   make(chan io.Reader, streamChannelSize),
		stdoutDone: make(chan struct{}, 1),
		stderrDone: make(chan struct{}, 1),
	}, nil
}

// handleStreamOutput handles the output streaming of the session depending on whether it has a tty or is exec.
func (ds *dockerSession) handleStreamOutput(exec bool) {
	// TTY case.
	if ds.tty {
		ds.streamUnifiedOutput()
	} else if exec {
		ds.streamSplitOutput()
	} else {
		ds.streamUnifiedOutput()
	}
}

// streamUnifiedOutput reads the output stream directly and sends it without distinguishing between stdout and stderr.
func (ds *dockerSession) streamUnifiedOutput() {
	// The reader can be used directly.
	for {
		buf := make([]byte, bufferSize)

		n, err := ds.reader.Read(buf)
		if n > 0 {
			reader := bytes.NewReader(buf[:n])
			ds.stdoutCh <- reader
		}

		if err != nil {
			if err != io.EOF &&
				!strings.Contains(err.Error(), "use of closed network connection") {
				// connection is closed.
				logger.WithField("container", ds.respID).Warnf("read container tty error: %v", err)
			}

			close(ds.stdoutCh)

			close(ds.stderrCh)

			return
		}
	}
}

// streamSplitOutput first reads and parses the header of the output,
// then sends the data to the corresponding channel based on the frame type (stdout or stderr).
func (ds *dockerSession) streamSplitOutput() {
	for {
		var (
			metadata []byte
			err      error
		)
		// Peek will block until reader got some data or error occurs.
		metadata, err = ds.reader.Peek(stdWriterPrefixLen)
		if err != nil {
			// Connection is closed.
			close(ds.stdoutCh)
			close(ds.stderrCh)

			return
		}

		ds.reader.Discard(stdWriterPrefixLen)

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

			n, err := io.ReadFull(ds.reader, buffer)
			if err != nil {
				logger.WithField("container", ds.respID).Errorf("pollout error: %v", err)

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
				logger.WithField("container", ds.respID).Errorf("got stdin output from exec connection")

				return
			case stdout:
				// Write on stdout.
				ds.stdoutCh <- reader
			case stderr:
				// Write on stderr.
				ds.stderrCh <- reader
			default:
				logger.WithField("container", ds.respID).Errorf("Unrecognized input header: %d", stream)

				return
			}
		}
	}
}

// cleanLegacyProcess clean the legacy processes before session disconnects.
func (ds *dockerSession) cleanLegacyProcess(isExec bool) error {
	if isExec {
		// Now clean legacy process only support sidecar scene.
		return nil
	}
	// Support the sidecar legacy process to kill.
	cid := ds.sidecarID

	cont, err := ds.client.ContainerInspect(context.Background(), cid)
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
