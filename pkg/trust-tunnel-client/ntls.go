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

//go:build ntls
// +build ntls

package client

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"os"

	"github.com/gorilla/websocket"
	tongsuogo "github.com/tongsuo-project/tongsuo-go-sdk"
	"github.com/tongsuo-project/tongsuo-go-sdk/crypto"
)

func (c *Client) dialAgent(nc *net.Conn, url *url.URL, header *http.Header, tlsConfig *tls.Config) (*websocket.Conn, error) {
	d := websocket.Dialer{}
	if nc != nil {
		d.NetDial = func(net, addr string) (net.Conn, error) {
			return *nc, nil
		}
	} else {
		d.NetDial = func(net, addr string) (net.Conn, error) {
			return c.DialSessionUsingNtls(addr)
		}
	}

	conn, _, err := d.Dial(url.String(), *header)
	return conn, err
}

// DialSessionUsingNTLS establishes a connection to the server using the NTLS protocol.
func (c *Client) DialSessionUsingNTLS(url string) (net.Conn, error) {
	ctx, err := tongsuogo.NewCtxWithVersion(tongsuogo.NTLS)
	if err != nil {
		return nil, err
	}

	if err := ctx.SetCipherList(c.Cipher); err != nil {
		return nil, err
	}

	if c.NTLSSignCertFile != "" {
		signCertPEM, err := os.ReadFile(c.NTLSSignCertFile)
		if err != nil {
			return nil, err
		}
		signCert, err := crypto.LoadCertificateFromPEM(signCertPEM)
		if err != nil {
			return nil, err
		}

		if err := ctx.UseSignCertificate(signCert); err != nil {
			return nil, err
		}
	}

	if c.NTLSSignKeyFile != "" {
		signKeyPEM, err := os.ReadFile(c.NTLSSignKeyFile)
		if err != nil {
			return nil, err
		}
		signKey, err := crypto.LoadPrivateKeyFromPEM(signKeyPEM)
		if err != nil {
			return nil, err
		}

		if err := ctx.UseSignPrivateKey(signKey); err != nil {
			return nil, err
		}
	}

	if c.NTLSEncCertFile != "" {
		encCertPEM, err := os.ReadFile(c.NTLSEncCertFile)
		if err != nil {
			return nil, err
		}
		encCert, err := crypto.LoadCertificateFromPEM(encCertPEM)
		if err != nil {
			return nil, err
		}

		if err := ctx.UseEncryptCertificate(encCert); err != nil {
			return nil, err
		}
	}

	if c.NTLSEncKeyFile != "" {
		encKeyPEM, err := os.ReadFile(c.NTLSEncKeyFile)
		if err != nil {
			return nil, err
		}

		encKey, err := crypto.LoadPrivateKeyFromPEM(encKeyPEM)
		if err != nil {
			return nil, err
		}

		if err := ctx.UseEncryptPrivateKey(encKey); err != nil {
			return nil, err
		}
	}

	if c.NTLSCaFile != "" {
		if err := ctx.LoadVerifyLocations(c.NTLSCaFile, ""); err != nil {
			return nil, err
		}
	}

	// Establish a TCP connection using the NTLS context and skip host verification (not recommended).
	conn, err := tongsuogo.Dial("tcp", url, ctx, tongsuogo.InsecureSkipHostVerification)

	return conn, err
}
