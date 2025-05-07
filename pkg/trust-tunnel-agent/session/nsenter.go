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
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"trust-tunnel/pkg/common/sessionutil"

	"github.com/creack/pty"
)

// nsenterSession represents a session structure for using nsenter to enter the host's namespace.
type nsenterSession struct {
	// cmd represents the command to be executed.
	cmd *exec.Cmd
	// exitCode stores the exit code of the command executed in the session.
	exitCode int
	// exitCh is used to signal the exit of the session.
	exitCh chan struct{}
	// tty indicates whether a pseudo-TTY is allocated for the session.
	tty bool

	// stdout, stderr, and stdin respectively represent the standard output, standard error, and standard input.
	stdout io.ReadCloser
	stderr io.ReadCloser
	stdin  io.WriteCloser

	// pid stores the process ID of the command executed in the session.
	pid int

	// stdoutDone and stderrDone are used to signal the completion of reading standard output and standard error.
	stdoutDone chan struct{}
	stderrDone chan struct{}

	// ptyChan is used to receive signals related to the pseudo-TTY.
	ptyChan chan os.Signal

	// master and slave respectively represent the master and slave ends of the pseudo-TTY.
	master, slave *os.File
}

func (s *nsenterSession) NextStdin() (io.WriteCloser, error) {
	return s.stdin, nil
}

func (s *nsenterSession) NextStdout() (io.Reader, error) {
	reader, err := sessionutil.OneRead(s.stdout)
	if err != nil && s.tty && (strings.Contains(err.Error(), "file already closed") ||
		strings.Contains(err.Error(), "input/output error")) {
		return nil, io.EOF
	}

	return reader, err
}

func (s *nsenterSession) NextStderr() (io.Reader, error) {
	reader, err := sessionutil.OneRead(s.stderr)
	if err != nil && s.tty && (strings.Contains(err.Error(), "file already closed") ||
		strings.Contains(err.Error(), "input/output error")) {
		return nil, io.EOF
	}

	return reader, err
}

func (s *nsenterSession) StderrDone() error {
	s.stderrDone <- struct{}{}

	return nil
}

func (s *nsenterSession) StdoutDone() error {
	s.stdoutDone <- struct{}{}

	return nil
}

func (s *nsenterSession) Clean() error {
	logger.Infof("clean process %d when session ends", s.pid)
	err := sessionutil.KillProcessGroup(s.pid, "nsenter", false)

	return err
}

func (s *nsenterSession) Resize(height, weight int) error {
	logger.Debugf("resize to %d*%d", height, weight)

	if s.master != nil {
		return pty.Setsize(s.master, &pty.Winsize{
			Rows: uint16(height),
			Cols: uint16(weight),
		})
	}

	return nil
}

func (s *nsenterSession) ExitCode() int {
	select {
	case <-s.exitCh:
		return s.exitCode
	case <-time.After(2 * time.Second):
		// Wait "wait()" func for returning.
		return 0
	}
}

func (s *nsenterSession) Exited() bool {
	select {
	case <-s.exitCh:
		return true
	default:
	}

	return false
}

// establishNsenterSession creates an nsenterSession by entering the host namespace based on provided configuration.
// It sets up either a console or raw I/O depending on the Tty flag in the configuration.
func establishNsenterSession(config *Config) (*nsenterSession, error) {
	logger.Infof("try to establish nsenter session")

	var (
		uid, gid string
		loginDir string
		err      error
	)

	if config.LoginName != "" {
		uid, gid, loginDir, err = sessionutil.GetUserInfo(config.LoginName, config.RootfsPrefix+"/etc/passwd")
		if err != nil {
			return nil, err
		}

		if uid == "" {
			return nil, fmt.Errorf("user does not exist:%s", config.LoginName)
		}
	}

	// Initialize the nsenter command arguments.
	// The arguments include the target PID, namespace types, and the command to be executed.
	args := []string{"-t", "1", "-m", "-u", "-i", "-n", "-p"}
	if uid != "" {
		args = append(args, "-S", uid, "-G", gid, "--wd="+config.RootfsPrefix+loginDir)
	}

	args = append(args, config.Cmd...)

	cmd := exec.Command("nsenter", args...)
	cmd.Env = []string{
		"PWD=" + loginDir,
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"TERM=xterm-256color",
	}

	session := &nsenterSession{
		cmd:        cmd,
		tty:        config.Tty,
		exitCh:     make(chan struct{}),
		stderrDone: make(chan struct{}),
		stdoutDone: make(chan struct{}),
		ptyChan:    make(chan os.Signal, 1),
	}

	// Set up either a console or raw I/O based on Tty flag.
	if config.Tty {
		if err = session.setupConsole(cmd); err != nil {
			return nil, fmt.Errorf("setup console failed: %v", err)
		}
	} else {
		if err = session.setupRawIO(cmd); err != nil {
			return nil, fmt.Errorf("setup raw IO failed: %v", err)
		}
	}

	if err = cmd.Start(); err != nil {
		return nil, fmt.Errorf("nsenter host namespace failed: %v", err)
	}

	// Record the PID of the started process.
	session.pid = cmd.Process.Pid

	go session.wait()

	return session, nil
}

// setupRawIO configures the raw I/O for the command execution.
// It sets up pipes for standard input, output, and error streams of the command.
// This allows the session to directly interact with the command's I/O.
func (s *nsenterSession) setupRawIO(cmd *exec.Cmd) error {
	var err error

	s.stdout, err = cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get command stdout pipe: %v", err)
	}

	s.stderr, err = cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get command stderr pipe: %v", err)
	}

	s.stdin, err = cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get command stdin pipe: %v", err)
	}

	return nil
}

// setupConsole configures a pseudo-TTY for the command execution.
// This is used to simulate a console environment for the command,
// allowing it to interact with the user directly.
func (s *nsenterSession) setupConsole(cmd *exec.Cmd) error {
	// Start the command with a pty.
	master, slave, err := pty.Open()
	if err != nil {
		return err
	}

	signal.Notify(s.ptyChan, syscall.SIGCHLD)

	cmd.Stdin, cmd.Stdout, cmd.Stderr = slave, slave, slave

	// Add the slave end of the pseudo-TTY to the command's extra files.
	// This is necessary to ensure the slave end is closed properly after the command finishes.
	cmd.ExtraFiles = append(cmd.ExtraFiles, slave)

	// Configure the command to run in a new session and set the controlling terminal.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid:  true,
		Setctty: true,
		Ctty:    0,
	}

	// Update the session's I/O to use the master end of the pseudo-TTY.
	s.stdin, s.stdout, s.stderr = master, master, master

	s.master = master
	s.slave = slave

	return nil
}

// wait will wait for the command to finish and sets the exit code.
func (s *nsenterSession) wait() {
	// If the session is running in TTY mode, wait for the pty to be closed.
	if s.tty {
		<-s.ptyChan
		signal.Reset(syscall.SIGCHLD)
		close(s.ptyChan)

		if s.master != nil {
			s.master.Close()
			s.slave.Close()
		}
	}

	<-s.stdoutDone

	<-s.stderrDone

	// Get the exit code of the command.
	s.exitCode = getExitCode(s.cmd)

	close(s.exitCh)
}

// getExitCode waits for the command to finish and returns the exit code.
func getExitCode(cmd *exec.Cmd) int {
	err := cmd.Wait()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ProcessState != nil {
				return exitErr.ExitCode()
			}
		} else {
			logger.Warnf("failed to wait command: %v", err)
		}
	}

	return 0
}
