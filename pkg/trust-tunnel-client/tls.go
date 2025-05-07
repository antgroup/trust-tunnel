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

//go:build !ntls

package client

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
)

// dialAgent dials the agent and establishes a websocket connection.
func (c *Client) dialAgent(networkConnection *net.Conn, url *url.URL, header *http.Header, tlsConfig *tls.Config) (*websocket.Conn, error) {
	// Initialize a websocket dialer with the TLS configuration.
	dialer := websocket.Dialer{
		TLSClientConfig: tlsConfig,
	}

	// If a network connection is provided, use it for dialing.
	if networkConnection != nil {
		dialer.NetDial = func(_, address string) (net.Conn, error) {
			return *networkConnection, nil
		}
	}

	// Dial the agent and return the websocket connection.
	conn, _, err := dialer.Dial(url.String(), *header) //nolint:bodyclose

	return conn, err
}
