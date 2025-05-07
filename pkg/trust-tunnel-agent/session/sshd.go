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
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"trust-tunnel/pkg/common/sessionutil"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

const (
	privateKeyPath     = "/root/.ssh/id_rsa_trust_tunnel_agent"
	publicKeyPath      = "/root/.ssh/id_rsa_trust_tunnel_agent.pub"
	authorizedKeysPath = "/.ssh/authorized_keys"
	passwdPath         = "/etc/passwd"
	sshTimeout         = 5 * time.Second
)

type sshSession struct {
	client  *ssh.Client
	session *ssh.Session

	stdin  io.WriteCloser
	stdout io.Reader
	stderr io.Reader

	stdoutDone chan struct{}
	stderrDone chan struct{}

	exitCh   chan struct{}
	exitCode int
}

func (s *sshSession) NextStdin() (io.WriteCloser, error) {
	return s.stdin, nil
}

func (s *sshSession) NextStdout() (io.Reader, error) {
	return sessionutil.OneRead(s.stdout)
}

func (s *sshSession) NextStderr() (io.Reader, error) {
	return sessionutil.OneRead(s.stderr)
}

func (s *sshSession) StderrDone() error {
	s.stderrDone <- struct{}{}

	return nil
}

func (s *sshSession) StdoutDone() error {
	s.stdoutDone <- struct{}{}

	return nil
}

func (s *sshSession) Clean() error {
	s.session.Close()
	s.client.Close()

	return nil
}

func (s *sshSession) Resize(h, w int) error {
	logger.Debugf("resize to %d*%d", h, w)

	return s.session.WindowChange(h, w)
}

func (s *sshSession) ExitCode() int {
	select {
	case <-s.exitCh:
		return s.exitCode
	case <-time.After(2 * time.Second):
		return 0
	}
}

// establishSSHSession attempts to create an SSH session based on the provided configuration.
// It handles key management, session setup, and command execution.
func establishSSHSession(c *Config) (*sshSession, error) {
	logger.Infof("try to establish ssh session")

	// Insert the public key onto the host machine.
	err := insertPubKeyOnHost(c.LoginName, c.RootfsPrefix)
	if err != nil {
		return nil, fmt.Errorf("SSH public key insert error: %v", err)
	}

	// Read the private key file for SSH authentication.
	key, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("SSH private key read error: %v", err)
	}

	// Parse the private key into a format usable by SSH.
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("SSH private key parse error: %v", err)
	}

	config := &ssh.ClientConfig{
		User: c.LoginName,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         sshTimeout,
	}

	sshClient, err := ssh.Dial("tcp", "127.0.0.1:22", config)
	if err != nil {
		return nil, fmt.Errorf("SSH connect error: %v", err)
	}

	session, err := sshClient.NewSession()
	if err != nil {
		sshClient.Close()

		return nil, fmt.Errorf("SSH new session error: %v", err)
	}

	// If TTY mode enabled, set up a pseudo-terminal (PTY) for the session.
	if c.Tty {
		setupSessionTTY(session)
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdout, _ := session.StdoutPipe()

	stderr, _ := session.StderrPipe()

	cmd := ""
	if len(c.Cmd) > 0 {
		cmd = c.Cmd[len(c.Cmd)-1]
	}

	logger.Debugf("SSH exec commands: %s", cmd)

	err = session.Start(cmd)
	if err != nil {
		session.Close()
		sshClient.Close()

		return nil, fmt.Errorf("SSH session start error: %v", err)
	}

	s := getSSHSession(sshClient, session, stdin, stdout, stderr)
	go s.wait()

	return s, nil
}

// insertPubKeyOnHost inserts the public key into the specified user's authorized_keys file.
// It is used to automatically configure SSH login for users.
func insertPubKeyOnHost(username string, rootfsPrefix string) error {
	// Reads the content of the public key file.
	key, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return fmt.Errorf("read pub key error: %v", err)
	}

	keyStr := string(key)

	// Retrieves the user's login directory and UID, GID
	uid, gid, loginDir, err := sessionutil.GetLoginDirAndIDs(username, rootfsPrefix+passwdPath, rootfsPrefix)
	if err != nil {
		return err
	}

	// Creates the SSH directory and authorized_keys file.
	err = createSSHDirAndAuthorizedKeysFile(loginDir, uid, gid)
	if err != nil {
		return err
	}

	authKeysFile := loginDir + authorizedKeysPath

	// Attempts to add the public key to the authorized_keys file,
	// returning whether the key was found, new content, and any error.
	keyFound, newContent, err := addPublicKeyToAuthorizedKeys(key, keyStr, authKeysFile)
	if err != nil {
		return err
	}

	if !keyFound {
		err = os.WriteFile(authKeysFile, newContent, 0)
		if err != nil {
			return fmt.Errorf("write authorized_keys error: %v", err)
		}
	}

	return nil
}

// createSSHDirAndAuthorizedKeysFile creates the SSH directory and the authorized_keys file for a user, setting appropriate permissions and ownership.
// loginDir: The login directory of the user.
// uid: The user ID (UID).
// gid: The group ID (GID).
func createSSHDirAndAuthorizedKeysFile(loginDir string, uid int, gid int) error {
	// Construct the path for the SSH directory and the authorized_keys file.
	sshDir := loginDir + "/.ssh"
	authKeysFile := loginDir + authorizedKeysPath

	// Create the SSH directory with proper permissions.
	err := os.MkdirAll(sshDir, 0o700)
	if err != nil {
		return fmt.Errorf("create .ssh directory error: %v", err)
	}

	// Change the ownership of the SSH directory to the specified user and group.
	err = os.Chown(sshDir, uid, gid)
	if err != nil {
		return fmt.Errorf("change ownership of .ssh directory error: %v", err)
	}

	// Open or create the authorized_keys file with read/write permissions.
	file, err := os.OpenFile(authKeysFile, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("create authorized_keys file error: %v", err)
	}
	defer file.Close()

	// Change the ownership of the authorized_keys file to the specified user and group.
	err = file.Chown(uid, gid)
	if err != nil {
		return fmt.Errorf("change ownership of authorized_keys file error: %v", err)
	}

	return nil
}

// addPublicKeyToAuthorizedKeys adds a public key to the authorized_keys file if it does not already exist.
// key: The public key data as a byte slice.
// keyStr: The string representation of the key used for searching within the file.
// authKeysFile: The path to the authorized_keys file.
func addPublicKeyToAuthorizedKeys(key []byte, keyStr string, authKeysFile string) (bool, []byte, error) {
	// Open the authorized_keys file for reading.
	file, err := os.Open(authKeysFile)
	if err != nil {
		return false, nil, fmt.Errorf("open authorized_keys file error: %v", err)
	}
	defer file.Close()

	// Initialize variables to track whether the key was found and to store new content.
	var keyFound bool

	var newContent []byte

	// Read the file line by line.
	buf := bufio.NewReader(file)

	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return false, nil, fmt.Errorf("read authorized_keys file error: %v", err)
			}
		}

		// Check if the current line contains the key string.
		if strings.Contains(line, keyStr) {
			keyFound = true

			break
		}

		// Exclude lines that do not end with "trust-tunnel-agent" from the new content.
		if !strings.HasSuffix(line, "trust-tunnel-agent") {
			newContent = append(newContent, []byte(line)...)
		}
	}

	// If the key was not found, append it to the new content.
	if !keyFound {
		newContent = append(newContent, key...)
	}

	return keyFound, newContent, nil
}

func getSSHSession(client *ssh.Client, session *ssh.Session, stdin io.WriteCloser, stdout io.Reader, stderr io.Reader) *sshSession {
	s := &sshSession{
		client:     client,
		session:    session,
		stdin:      stdin,
		stdout:     stdout,
		stderr:     stderr,
		exitCh:     make(chan struct{}, 1),
		stdoutDone: make(chan struct{}, 1),
		stderrDone: make(chan struct{}, 1),
	}

	return s
}

// setupSessionTTY configures the TTY settings for the SSH session if TTY is enabled.
func setupSessionTTY(session *ssh.Session) {
	// Set up terminal modes and request a PTY
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.ECHOCTL:       0,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	width, height, err := term.GetSize(int(os.Stdin.Fd()))
	if err == nil {
		err = session.RequestPty("xterm-256color", height, width, modes)
		if err != nil {
			logger.Errorf("Error requesting PTY: %v", err)
		}
	} else {
		logger.Errorf("Failed to determine terminal size: %v", err)
	}
}

func (s *sshSession) wait() {
	<-s.stderrDone
	<-s.stdoutDone

	if err := s.session.Wait(); err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			s.exitCode = exitErr.ExitStatus()
		} else {
			logger.Warnf("ssh session exit error: %v", err)
		}
	}

	close(s.exitCh)
}
