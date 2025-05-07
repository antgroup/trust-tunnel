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

package app

import (
	"fmt"
	"os"

	"github.com/gorilla/websocket"
	"golang.org/x/term"
	client "trust-tunnel/pkg/trust-tunnel-client"
)

const bufferSize = 1024

// createClient creates a client based on the given Option.
func createClient(opt *Option) (*client.Client, error) {
	targetType, err := getClientTargetType(opt.Type)
	if err != nil {
		return nil, err
	}

	cli := client.Client{
		SessionID:        opt.SessionID,
		AgentAddr:        opt.Host,
		AgentPort:        opt.Port,
		Type:             targetType,
		PodName:          opt.Pod,
		ContainerName:    opt.ContainerName,
		ContainerID:      opt.ContainerID,
		IPAddress:        opt.IP,
		Interactive:      opt.Interactive,
		Tty:              opt.Tty,
		Command:          opt.Cmd,
		LoginName:        opt.LoginName,
		LoginGroup:       opt.LoginGroup,
		UserName:         opt.UserName,
		TLSVerify:        opt.TLSVerify,
		TLSCaCert:        opt.TLSCa,
		TLSCert:          opt.TLSCert,
		TLSKey:           opt.TLSKey,
		NtlsVerify:       opt.NTLSVerify,
		NTLSCaFile:       opt.NTLSCa,
		NTLSSignCertFile: opt.NTLSSignCert,
		NTLSEncCertFile:  opt.NTLSEncCert,
		NTLSEncKeyFile:   opt.NTLSEncKey,
		NTLSSignKeyFile:  opt.NTLSSignKey,
		Cipher:           opt.Cipher,
		Cpus:             opt.Cpus,
		MemoryMB:         opt.MemoryMB,
		DisableCleanMode: opt.DisableCleanMode,
	}

	return &cli, nil
}

// getClientTargetType returns the client.TargetType based on the given targetType.
func getClientTargetType(targetType string) (client.TargetType, error) {
	switch targetType {
	case "phys":
		return client.TargetPhys, nil
	case "container":
		return client.TargetContainer, nil
	default:
		return 0, fmt.Errorf("wrong target type")
	}
}

// runClient creates a client and starts a session. It sets up signal handling and
// launches goroutines to handle local input and remote output and error streams.
func runClient(opt *Option) (int, error) {
	cli, err := createClient(opt)
	if err != nil {
		return -1, err
	}

	session, err := cli.Start(nil)
	if err != nil {
		return -1, err
	}

	w, h, _ := term.GetSize(int(os.Stdin.Fd()))

	err = session.Resize(h, w)
	if err != nil {
		return -1, err
	}

	setupSignal(session)

	if cli.Interactive && cli.Tty {
		fd := int(os.Stdin.Fd())

		oldState, err := term.MakeRaw(fd)
		if err != nil {
			return -1, err
		}
		defer term.Restore(fd, oldState)
	}

	errs := make(chan error, 1)

	go processLocalInput(errs, session)
	go processRemoteOutput(errs, session)
	go processRemoteErr(errs, session)

	err = <-errs

	return session.ExitCode(), err
}

// processLocalInput reads from os.Stdin and writes to a client.Session.
func processLocalInput(errs chan error, session client.Session) {
	buf := make([]byte, bufferSize)

	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			errs <- fmt.Errorf("read from stdin error: %v", err)

			return
		}

		written := 0
		for written < n {
			m, err := session.Write(buf[written:n])
			if err != nil {
				errs <- fmt.Errorf("write to remote error: %v", err)

				return
			}

			written += m
		}
	}
}

// processRemoteOutput reads from a client.Session and writes the output to os.Stdout.
func processRemoteOutput(errs chan error, session client.Session) {
	buf := make([]byte, 1024)

	for {
		n, err := session.Read(buf)
		if err != nil {
			if closeErr, ok := err.(*websocket.CloseError); ok && closeErr.Code == websocket.CloseNormalClosure {
				// If the error is a normal close error, ignore it and exit the loop.
				errs <- nil

				return
			}
			errs <- fmt.Errorf("read from remote error: %v", err)

			return
		}

		written := 0
		for written < n {
			m, err := os.Stdout.Write(buf[written:n])
			if err != nil {
				errs <- fmt.Errorf("write to Stdout error: %v", err)

				return
			}

			written += m
		}
	}
}

// processRemoteErr reads from a client.Session and writes the error output to os.Stderr.
func processRemoteErr(errs chan error, session client.Session) {
	buf := make([]byte, 1024)

	for {
		n, err := session.ReadStderr(buf)
		if err != nil {
			if closeErr, ok := err.(*websocket.CloseError); ok && closeErr.Code == websocket.CloseNormalClosure {
				// If the error is a normal close error, ignore it and exit the loop.
				errs <- nil

				return
			}
			errs <- fmt.Errorf("read from remote stderr error: %v", err)

			return
		}

		written := 0
		for written < n {
			m, err := os.Stderr.Write(buf[written:n])
			if err != nil {
				errs <- fmt.Errorf("write to Stderr error: %v", err)

				return
			}

			written += m
		}
	}
}
