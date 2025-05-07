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

package app

import (
	"net"
	"net/http"
	"os"
	"trust-tunnel/pkg/trust-tunnel-agent/backend"
	"trust-tunnel/pkg/trust-tunnel-agent/monitor"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	tongsuogo "github.com/tongsuo-project/tongsuo-go-sdk"
	"github.com/tongsuo-project/tongsuo-go-sdk/crypto"
	agentSession "trust-tunnel/pkg/trust-tunnel-agent/session"
)

// NTLSServer represents a server structure that implements the server interface, specifically designed for the NTLS protocol.
type NTLSServer struct{}

// NewServer initializes and returns a new instance of NTLSServer.
func NewServer() Server {
	return &NTLSServer{}
}

func (s *NTLSServer) Start(opt *Option) error {
	addr := net.JoinHostPort(opt.Host, opt.Port)
	server := &http.Server{
		Addr: addr,
	}

	handler, err := backend.NewHandler(backend.Config{
		RootfsPrefix:     opt.RootfsPrefix,
		AuthConfig:       opt.AuthConfig,
		Endpoint:         opt.Endpoint,
		PhysTunnel:       opt.PhysTunnel,
		ContainerRuntime: agentSession.ContainerRuntime(opt.ContainerRuntime),
		SidecarImage:     opt.SidecarImage,
		ImageHubAuth:     opt.ImageHubAuth,
		SidecarLimit:     opt.SidecarLimit,
	})
	if err != nil {
		return err
	}

	r := mux.NewRouter()
	r.HandleFunc("/exec", func(w http.ResponseWriter, r *http.Request) {
		handler.Handle(w, r)
	})
	server.Handler = monitor.WrapPrometheus(r)

	// If NTLS verification is enabled, create a new NTLS listener and serve the HTTP server.
	if opt.NTLSConfig.NTLSVerify {
		lis, err := newNTLSListener(addr, opt.NTLSConfig, func(sslctx *tongsuogo.Ctx) error {
			return sslctx.SetCipherList(opt.NTLSConfig.Cipher)
		})
		if err != nil {
			return err
		}

		return server.Serve(*lis)
	}

	logrus.Info("start ntls server")

	return server.ListenAndServe()
}

// newNTLSListener creates a new NTLS listener with the specified address and configuration.
func newNTLSListener(addr string, ntlsConfig NTLSConfig, options ...func(sslctx *tongsuogo.Ctx) error) (*net.Listener, error) {
	ctx, err := tongsuogo.NewCtxWithVersion(tongsuogo.NTLS)
	if err != nil {
		return nil, err
	}

	for _, f := range options {
		if err := f(ctx); err != nil {
			return nil, err
		}
	}

	if err := ctx.LoadVerifyLocations(ntlsConfig.NTLSCaFile, ""); err != nil {
		return nil, err
	}

	if ntlsConfig.NTLSEncCertFile != "" {
		encCertPEM, err := os.ReadFile(ntlsConfig.NTLSEncCertFile)
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

	if ntlsConfig.NTLSEncKeyFile != "" {
		encKeyPEM, err := os.ReadFile(ntlsConfig.NTLSEncKeyFile)
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

	if ntlsConfig.NTLSSignCertFile != "" {
		signCertPEM, err := os.ReadFile(ntlsConfig.NTLSSignCertFile)
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

	if ntlsConfig.NTLSSignKeyFile != "" {
		signKeyPEM, err := os.ReadFile(ntlsConfig.NTLSSignKeyFile)
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

	lis, err := tongsuogo.Listen("tcp", addr, ctx)
	if err != nil {
		return nil, err
	}

	return &lis, nil
}
