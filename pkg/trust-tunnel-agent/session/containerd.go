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
	gocontext "context"
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"strings"
	"syscall"
	"time"
	"trust-tunnel/pkg/common/sessionutil"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/cmd/ctr/commands"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	"golang.org/x/net/context"
)

const (
	randomSeed = 1048576
)

// containerdSession represents a session with a containerd process.
type containerdSession struct {
	process       containerd.Process
	stdin         *io.PipeWriter
	stdout        *io.PipeReader
	stderr        *io.PipeReader
	exitCh        <-chan containerd.ExitStatus
	inReaderPipe  *io.PipeReader
	outWriterPipe *io.PipeWriter
	errWriterPipe *io.PipeWriter
	detach        bool
	cancelFunc    gocontext.CancelFunc
	exitCode      uint32
	ctx           context.Context
	stdoutDone    chan struct{}
	stderrDone    chan struct{}
	execID        string
	task          containerd.Task
}

func (s *containerdSession) NextStdin() (io.WriteCloser, error) {
	return s.stdin, nil
}

func (s *containerdSession) NextStdout() (io.Reader, error) {
	reader, err := sessionutil.OneRead(s.stdout)
	// If the pipe is closed, return EOF.
	if err != nil && (strings.Contains(err.Error(), "closed pipe")) {
		return nil, io.EOF
	}

	return reader, err
}

func (s *containerdSession) NextStderr() (io.Reader, error) {
	reader, err := sessionutil.OneRead(s.stderr)
	// If the pipe is closed, return EOF.
	if err != nil && (strings.Contains(err.Error(), "closed pipe")) {
		return nil, io.EOF
	}

	return reader, err
}

func (s *containerdSession) StderrDone() error {
	s.stderrDone <- struct{}{}

	return nil
}

func (s *containerdSession) StdoutDone() error {
	s.stdoutDone <- struct{}{}

	return nil
}

func (s *containerdSession) Clean() error {
	select {
	case <-s.ctx.Done():
		// If the context is canceled, the process is already cleaned up, no action needed.
	default:
		// The task may not be killed, so kill it.
		if s.task != nil && s.execID != "" {
			err := s.task.Kill(s.ctx, syscall.SIGKILL, containerd.WithKillExecID(s.execID))
			if err != nil {
				logger.Errorf("kill task err:%v", err)
			}
		}
	}

	s.stdout.Close()
	s.stderr.Close()
	s.inReaderPipe.Close()

	return nil
}

func (s *containerdSession) Resize(h, w int) error {
	logger.Debugf("resize to %d*%d", h, w)

	if s.process == nil {
		return nil
	}

	// Resize the console.
	return s.process.Resize(gocontext.Background(), uint32(w), uint32(h))
}

func (s *containerdSession) ExitCode() int {
	return int(s.exitCode)
}

// wait implements waiting for the session to exit and cleans up the resources.
func (s *containerdSession) wait(exitCh <-chan containerd.ExitStatus) error {
	status := <-exitCh

	// Wait for 100 milliseconds before closing the standard input and output pipes.
	time.Sleep(100 * time.Millisecond)
	s.stdin.Close()
	s.outWriterPipe.Close()
	s.errWriterPipe.Close()

	// Wait for the stdout and stderr pipes to be closed.
	<-s.stdoutDone
	<-s.stderrDone
	logger.Infof("clean task process")

	if !s.detach && s.process != nil {
		s.process.Delete(s.ctx)
	}

	// Cancel the context.
	s.cancelFunc()

	var err error

	s.exitCode, _, err = status.Result()
	if err != nil {
		return err
	}

	return nil
}

// establishContainerdSession establishes a containerd session and returns the session and any errors.
func establishContainerdSession(c *Config, containerdClient *containerd.Client) (*containerdSession, error) {
	// Check if the containerd client is nil.
	if containerdClient == nil {
		return nil, fmt.Errorf("containerd Client is nil")
	}

	var session *containerdSession

	var loginDir string

	var err error

	// If the login name is provided in the config, get the user info.
	if c.LoginName != "" {
		// TODO:get gid from Config.LoginGroup
		_, _, loginDir, err = sessionutil.GetUserInfo(c.LoginName, c.RootfsPrefix+"/etc/passwd")
		if err != nil {
			return nil, err
		}
	}

	if len(c.Cmd) > 0 {
		c.Cmd[len(c.Cmd)-1] = "cd " + loginDir + ";" + c.Cmd[len(c.Cmd)-1]
	}

	logger.Infof("exec into container %s directly", c.ContainerID)

	// Now containerd runtime only support exec.
	session, err = execContainerd(c, containerdClient, c.ContainerNamespace)
	if err != nil {
		return nil, err
	}

	return session, nil
}

// execContainerd implements exec into a container with containerd runtime.
func execContainerd(c *Config, client *containerd.Client, namespace string) (*containerdSession, error) {
	// Get the container ID, command, TTY, login name and detach flag from the config.
	id := c.ContainerID
	args := c.Cmd
	tty := c.Tty
	user := c.LoginName

	// Check if the container ID is provided in the config.
	if id == "" {
		return nil, fmt.Errorf("container id must be provided")
	}

	// Create a namespace context and a cancel function.
	ctx := namespaces.WithNamespace(context.Background(), namespace)
	ctx, cancel := gocontext.WithCancel(ctx) //nolint:govet

	// Load the container using the containerd client.
	container, err := client.LoadContainer(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load container err:%v", err) //nolint:govet
	}

	// Get the container spec.
	spec, err := container.Spec(ctx)
	if err != nil {
		return nil, err
	}

	// Check if the login name is valid.
	if user != "" {
		c, err := container.Info(ctx)
		if err != nil {
			return nil, err
		}

		if err := oci.WithUser(user)(ctx, client, &c, spec); err != nil {
			return nil, err
		}
	}

	// Set the process task exec arguments.
	pSpec := spec.Process
	pSpec.Terminal = tty
	pSpec.Args = args
	pSpec.Env = []string{
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"TERM=xterm-256color",
	}

	// Create a task to execute commands in the container.
	task, err := container.Task(ctx, nil)
	if err != nil {
		return nil, err
	}

	var ioCreator cio.Creator
	// Create the input, output and error pipes.
	inReaderPipe, inWriterPipe := io.Pipe()
	outReaderPipe, outWriterPipe := io.Pipe()
	errReaderPipe, errWriterPipe := io.Pipe()

	// Set the cio options.
	cioOpts := []cio.Opt{cio.WithStreams(inReaderPipe, outWriterPipe, errWriterPipe)}

	if tty {
		cioOpts = append(cioOpts, cio.WithTerminal)
	}

	ioCreator = cio.NewCreator(cioOpts...)

	// Generate a random exec ID.
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	randomNumber := rng.Intn(randomSeed)
	execID := strconv.Itoa(randomNumber)
	logger.Infof("exec id is %s", execID)

	// Execute the process task using the cio creator.
	process, err := task.Exec(ctx, execID, pSpec, ioCreator)
	if err != nil {
		return nil, err
	}

	// Wait for the process to finish and get the status channel.
	statusC, err := process.Wait(ctx)
	if err != nil {
		return nil, err
	}

	if err := process.Start(ctx); err != nil {
		return nil, err
	}

	// Forward all signals to the process.
	sigs := commands.ForwardAllSignals(ctx, process)
	defer commands.StopCatch(sigs)

	s := &containerdSession{
		exitCh:        statusC,
		stdin:         inWriterPipe,
		stdout:        outReaderPipe,
		stderr:        errReaderPipe,
		outWriterPipe: outWriterPipe,
		errWriterPipe: errWriterPipe,
		inReaderPipe:  inReaderPipe,
		detach:        false,
		cancelFunc:    cancel,
		ctx:           ctx,
		stderrDone:    make(chan struct{}),
		stdoutDone:    make(chan struct{}),
		task:          task,
		execID:        execID,
	}
	go s.wait(statusC)

	return s, nil
}
