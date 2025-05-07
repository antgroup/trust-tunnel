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
	"net/http"
	"strconv"
	"time"

	"github.com/felixge/httpsnoop"
)

// WrapPrometheus wraps an HTTP handler to collect and record metrics related to the request handling.
// It takes an http.Handler as an argument and returns a new http.Handler that, when serving requests,
// records metrics data such as request duration, path, and method.
func WrapPrometheus(next http.Handler) http.Handler {
	// Returns an http.HandlerFunc that will be invoked to serve HTTP requests.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture the start time of the request.
		var (
			path   = r.URL.Path
			method = r.Method
			st     = time.Now()
		)

		// Increment the counter for current requests at the start of a request.
		MetricsHTTPCurrentRequests.WithLabelValues(path, method).Inc()

		// Captures metrics after the request is handled using the httpsnoop package.
		metrics := httpsnoop.CaptureMetrics(next, w, r)
		code := strconv.Itoa(metrics.Code)
		delta := time.Since(st).Milliseconds()

		// Decrement the counter for current requests once the request has been served.
		MetricsHTTPCurrentRequests.WithLabelValues(path, method).Dec()
		// Observe the request response time in milliseconds.
		MetricsHTTPRequestRt.WithLabelValues(path, method).Observe(float64(delta))
		// Increment the counter for requests based on path, method, and response code.
		MetricsHTTPRequests.WithLabelValues(path, method, code).Inc()
	})
}
