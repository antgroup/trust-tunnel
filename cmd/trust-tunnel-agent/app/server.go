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
	"net"
	"net/http"
	"trust-tunnel/pkg/common/logutil"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// runServer configures and starts the trust-tunnel-agent server.
func runServer(opt *Option) error {
	// Setup logging.
	level, err := logrus.ParseLevel(opt.LogConfig.Level)
	if err != nil {
		return err
	}

	logutil.SetLevel(level)
	logutil.SetExpireDay(opt.LogConfig.ExpireDays)

	setupSignal()

	// Log global configuration.
	logGlobalConfig(opt)

	// Start monitoring server.
	go startMonitorServer()

	// Start serving requests.
	server := NewServer()

	return server.Start(opt)
}

// startMonitorServer starts the monitoring server.
func startMonitorServer() {
	addr := net.JoinHostPort("0.0.0.0", "19104")
	server := &http.Server{
		Addr: addr,
	}
	r := mux.NewRouter()
	r.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) { promhttp.Handler().ServeHTTP(w, r) })
	server.Handler = r
	server.ListenAndServe()
}
