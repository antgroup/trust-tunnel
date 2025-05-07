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

package app

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"os"
	"trust-tunnel/pkg/trust-tunnel-agent/backend"
	"trust-tunnel/pkg/trust-tunnel-agent/monitor"

	"github.com/gorilla/mux"
)

type TLSServer struct{}

func NewServer() Server {
	return &TLSServer{}
}

func (s *TLSServer) Start(opt *Option) error {
	addr := net.JoinHostPort(opt.Host, opt.Port)
	server := &http.Server{
		Addr: addr,
	}

	// If TLS verification is enabled, configure the TLS settings for the server.
	if opt.TLSConfig.TLSVerify {
		tlsConfig, err := ConfigTLS(&TLSConfig{
			TLSCA:   opt.TLSConfig.TLSCA,
			TLSCert: opt.TLSConfig.TLSCert,
			TLSKey:  opt.TLSConfig.TLSKey,
		})
		if err != nil {
			return err
		}

		server.TLSConfig = tlsConfig
	}

	handler, err := backend.NewHandler(&backend.Config{
		ContainerConfig: opt.ContainerConfig,
		AuthConfig:      opt.AuthConfig,
		SessionConfig:   opt.SessionConfig,
		SidecarConfig:   opt.SidecarConfig,
	})
	if err != nil {
		return err
	}

	r := mux.NewRouter()
	r.HandleFunc("/exec", func(w http.ResponseWriter, r *http.Request) {
		handler.Handle(w, r)
	})

	// Wrap the router with Prometheus monitoring middleware.
	server.Handler = monitor.WrapPrometheus(r)

	// If TLS is enabled, start the server in TLS mode.
	if opt.TLSConfig.TLSVerify {
		return server.ListenAndServeTLS("", "")
	}

	// Start the HTTP server without TLS.
	return server.ListenAndServe()
}

// ConfigTLS creates a TLS configuration from command line options.
func ConfigTLS(config *TLSConfig) (*tls.Config, error) {
	pool := x509.NewCertPool()

	caCert, err := os.ReadFile(config.TLSCA)
	if err != nil {
		return nil, err
	}

	pool.AppendCertsFromPEM(caCert)

	cert, err := tls.LoadX509KeyPair(config.TLSCert, config.TLSKey)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		RootCAs:      pool,
		ClientCAs:    pool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{cert},
	}

	return tlsConfig, nil
}
