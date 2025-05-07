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

package monitor

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	MetricsHTTPRequestRt = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_rt_us",
		Help:    "The time of each http request",
		Buckets: []float64{1000, 2000, 3000, 5000, 8000},
	}, []string{"path", "method"})

	MetricsHTTPRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "The count of http request on ip address and status code",
	}, []string{"path", "method", "code"})

	MetricsHTTPCurrentRequests = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "http_current_requests_total",
		Help: "The count of current http request on ip address and status code",
	}, []string{"path", "method"})

	MetricsVerifyClientCertError = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "verify_client_cert_error",
		Help: "The count of verify client cert error",
	}, []string{})

	MetricsEstablishSessionError = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "establish_session_error",
		Help: "The count of establish session error",
	}, []string{})

	MetricsEstablishSessionSuccess = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "establish_session_success",
		Help: "The count of establish session success",
	}, []string{})

	MetricsKillLegacyProcessCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "kill_residual_process_count",
		Help: "The count of legacy process to kill",
	}, []string{"mode"})

	MetricsLegacySidecarCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "legacy_sidecar_count",
		Help: "The count of legacy sidecar container",
	})
)

func init() {
	prometheus.MustRegister(
		MetricsHTTPRequestRt,
		MetricsHTTPRequests,
		MetricsHTTPCurrentRequests,
		MetricsVerifyClientCertError,
		MetricsEstablishSessionError,
		MetricsEstablishSessionSuccess,
		MetricsKillLegacyProcessCount,
		MetricsLegacySidecarCount,
	)
}
