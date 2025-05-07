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
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

// genTLSConfig generates a TLS configuration for the client.
func (c *Client) genTLSConfig() (*tls.Config, error) {
	pool := x509.NewCertPool()

	caCert, err := os.ReadFile(c.TLSCaCert)
	if err != nil {
		return nil, err
	}

	pool.AppendCertsFromPEM(caCert)

	cert, err := tls.LoadX509KeyPair(c.TLSCert, c.TLSKey)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		RootCAs:      pool,
		Certificates: []tls.Certificate{cert},
		ServerName:   "trust-tunnel-agent",
	}, nil
}

// start establishes a connection to the server and returns a session.
func (c *Client) start(networkConnection *net.Conn) (Session, error) {
	// Construct the server URL
	host := net.JoinHostPort(c.AgentAddr, strconv.Itoa(c.AgentPort))
	urlPath := url.URL{Host: host, Path: "/exec"}

	var tlsConfig *tls.Config

	var err error

	if c.TLSVerify {
		// Use secure websockets if TLS verify is enabled.
		urlPath.Scheme = "wss"

		tlsConfig, err = c.genTLSConfig()
		if err != nil {
			return nil, err
		}
	} else {
		// Use regular websockets if TLS verify is disabled.
		urlPath.Scheme = "ws"
	}

	// Get the base64 encoded command.
	var encodedCommand []string

	for _, comm := range c.Command {
		encodedData := base64.StdEncoding.EncodeToString([]byte(comm))
		encodedCommand = append(encodedCommand, encodedData)
	}

	// Construct the request headers.
	header := http.Header{
		"Session-Id":            []string{c.SessionID},
		"User-Name":             []string{c.UserName},
		"Login-Name":            []string{c.LoginName},
		"Login-Group":           []string{c.LoginGroup},
		"Ip-Address":            []string{c.IPAddress},
		"Interactive":           []string{strconv.FormatBool(c.Interactive)},
		"Tty":                   []string{strconv.FormatBool(c.Tty)},
		"Command":               c.Command,
		"Command-Base64-Encode": encodedCommand,
		"Cpus":                  []string{strconv.FormatFloat(c.Cpus, 'f', -1, 64)},
		"Memory":                []string{strconv.Itoa(c.MemoryMB)},
		"Agent-Addr":            []string{c.AgentAddr},
	}

	if c.DisableCleanMode {
		header["Disable-Clean-Mode"] = []string{"1"}
	}

	if c.Type == TargetPhys {
		header["Target-Type"] = []string{"physical"}
	} else {
		header["Target-Type"] = []string{"container"}
		header["Pod-Name"] = []string{c.PodName}

		if len(c.ContainerName) > 0 {
			header["Container-Name"] = []string{c.ContainerName}
		}

		if len(c.ContainerID) > 0 {
			header["Container-Id"] = []string{c.ContainerID}
		}
	}

	// Dial the agent and establish a websocket connection.
	conn, err := c.dialAgent(networkConnection, &urlPath, &header, tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("connecting to agent by websocket error: %v", err)
	}

	// Create and return a new agent session.
	agent := &agentConn{
		conn:         conn,
		interactive:  c.Interactive,
		tty:          c.Tty,
		stdoutBuffer: NewBlockingBuffer(),
		stderrBuffer: NewBlockingBuffer(),
	}
	go agent.ProcessMsg()

	return agent, nil
}

// Start the client and try to communicate with agent on conn.
// If conn is nil, a new connection will be established with given agent addr and port.
// If conn it not nil, it will be used for communication with agent. It's the caller's
// responsibility to guarantee the peer end of the connection could handle following
// communication messages.
func (c *Client) Start(conn *net.Conn) (Session, error) {
	return c.start(conn)
}
