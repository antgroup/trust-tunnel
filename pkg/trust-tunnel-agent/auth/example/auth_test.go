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

package example

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"trust-tunnel/pkg/trust-tunnel-agent/auth"
	"trust-tunnel/pkg/trust-tunnel-agent/backend/request"
	client "trust-tunnel/pkg/trust-tunnel-client"
)

func TestVerifyAccessPermission(t *testing.T) {
	type TestCase struct {
		Name         string
		Request      *request.Info
		ExpectedCode auth.Code
	}

	tests := []TestCase{
		{
			Name: "Alice should have access",
			Request: &request.Info{
				UserName:   "alice",
				LoginName:  "root",
				TargetType: client.TargetPhys,
				IPAddress:  "ip1",
			},
			ExpectedCode: auth.Success,
		},
		{
			Name: "Bob should not have access",
			Request: &request.Info{
				UserName:   "bob",
				LoginName:  "root",
				TargetType: client.TargetPhys,
				IPAddress:  "ip2",
			},
			ExpectedCode: auth.Forbidden,
		},
	}
	// Create a mock HTTP server to simulate the authorization server.
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read the request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}

		// Unmarshal the request body into a Request struct.
		var req request.Info

		err = json.Unmarshal(body, &req)
		if err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}

		var resp auth.Response
		// Check if the user is permitted to access the target.
		if req.UserName == "alice" && req.IPAddress == "ip1" {
			// Return a successful response.
			resp = auth.Response{
				Code:   auth.Success,
				ErrMsg: "",
			}
			respBytes, _ := json.Marshal(resp)
			w.Write(respBytes)
		} else {
			// Return a forbidden response.
			resp = auth.Response{
				Code:   auth.Forbidden,
				ErrMsg: "",
			}
			respBytes, _ := json.Marshal(resp)
			w.Write(respBytes)
		}
	}))
	defer mockServer.Close()

	// Create a new AuthHandler with the mock server URL.
	authHandler := &AuthHandler{
		AuthURL: mockServer.URL,
		Client:  mockServer.Client(),
	}

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			// Perform the test.
			resp := authHandler.VerifyAccessPermission(tc.Request)
			if resp.Code != tc.ExpectedCode {
				t.Errorf("Test '%s' failed: expected error %v, but got: %v", tc.Name, tc.ExpectedCode, resp.Code)
			}
		})
	}
}
