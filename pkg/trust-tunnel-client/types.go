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

package client

import (
	"io"
)

// TargetType represents the type of target host to log in,
// either physical machine or container.
type TargetType byte

const (
	TargetPhys TargetType = iota
	TargetContainer
)

// NormalCloseMessage represents a message for a normal close with a code and error.
type NormalCloseMessage struct {
	Code int
	Err  error
}

// Client represents the configuration and data for a client connecting to a server.
type Client struct {
	// Session ID.
	SessionID string

	// IP address of agent.
	AgentAddr string

	// Port of agent.
	AgentPort int

	// Type of target host to log in (physical machine or container).
	Type TargetType

	// UserName specifies the username for the user's identity.
	UserName string

	// LoginName specifies the login name for the target to connect.
	LoginName string

	// LoginGroup specifies the login group for the target to connect.
	LoginGroup string

	// Ip Address to be used for auth.
	IPAddress string

	// Name of pod to log in, ignored if type is TargetPhys.
	PodName string

	// Name of container to execute command (e.g., main), ignored if type is TargetPhys.
	ContainerName string

	// ID of container to execute command, ignored if type is TargetPhys.
	ContainerID string

	// Enable tls verification if set to true.
	TLSVerify bool

	// Path of CA certificate file of TLS.
	TLSCaCert string

	// Path of certificate file of TLS.
	TLSCert string

	// Path of key file of TLS.
	TLSKey string

	// Enable ntls verification if set to true.
	NtlsVerify bool

	// Path of sign cert file of NTLS.
	NTLSSignCertFile string

	// Path of sign key file of NTLS.
	NTLSSignKeyFile string

	// Path of enc cert file of NTLS.
	NTLSEncCertFile string

	// Path of enc key file of NTLS.
	NTLSEncKeyFile string

	// Path of CA certificate file of NTLS.
	NTLSCaFile string

	// Cipher of NTLS.
	Cipher string

	// Redirect STDIN of target host.
	Interactive bool

	// Allocate a tty device.
	Tty bool

	// Commands to be executed on target.
	Command []string

	// CPU resource for limiting the commands, e.g. 0.5, 2.0.
	Cpus float64

	// Memory resource in MB for limiting the commands, e.g. 500, 2048.
	MemoryMB int

	// DisableCleanMode is set to false as default.
	// Disable clean mode means remote cmd will be executed via "docker exec" for container,
	// and "ssh" for physical host.
	DisableCleanMode bool
}

// Session represents a bidirectional RPC session for interacting with the target host.
type Session interface {
	io.ReadWriteCloser

	// ReadStderr reads error output from the remote command.
	ReadStderr(p []byte) (n int, err error)

	// Resize adjusts the size of the remote terminal.
	Resize(height int, width int) error

	// CloseSession closes the current session.
	CloseSession() error

	// ExitCode returns the exit code of the remote command.
	ExitCode() int
}
