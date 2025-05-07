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
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gorilla/websocket"
)

func TestDialAgentWithMockServer(t *testing.T) {
	// Set up mock server.
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers.
		if r.Header.Get("Session-Id") != "testsession" {
			t.Errorf("unexpected Session-Id header: got %s, want %s", r.Header.Get("Session-Id"), "testsession")
		}

		if r.Header.Get("User-Name") != "testuser" {
			t.Errorf("unexpected User-Name header: got %s, want %s", r.Header.Get("User-Name"), "testuser")
		}

		if r.Header.Get("Login-Name") != "testlogin" {
			t.Errorf("unexpected Login-Name header: got %s, want %s", r.Header.Get("Login-Name"), "testlogin")
		}

		if r.Header.Get("Login-Group") != "testgroup" {
			t.Errorf("unexpected Login-Group header: got %s, want %s", r.Header.Get("Login-Group"), "testgroup")
		}

		if r.Header.Get("App-Name") != "testapp" {
			t.Errorf("unexpected App-Name header: got %s, want %s", r.Header.Get("App-Name"), "testapp")
		}

		if r.Header.Get("Ip-Address") != "1.2.3.4" {
			t.Errorf("unexpected Ip-Address header: got %s, want %s", r.Header.Get("Ip-Address"), "1.2.3.4")
		}

		if r.Header.Get("Interactive") != "true" {
			t.Errorf("unexpected Interactive header: got %s, want %s", r.Header.Get("Interactive"), "true")
		}

		if r.Header.Get("Tty") != "true" {
			t.Errorf("unexpected Tty header: got %s, want %s", r.Header.Get("Tty"), "true")
		}

		if r.Header.Get("Command") != "ls -l" {
			t.Errorf("unexpected Command header: got %s, want %s", r.Header.Get("Command"), "ls -l")
		}

		if r.Header.Get("Command-Base64-Encode") != "bHMKLC1s" {
			t.Errorf("unexpected Command-Base64-Encode header: got %s, want %s", r.Header.Get("Command-Base64-Encode"), "bHMKLC1s")
		}

		if r.Header.Get("Cpus") != "1" {
			t.Errorf("unexpected Cpus header: got %s, want %s", r.Header.Get("Cpus"), "1")
		}

		if r.Header.Get("Memory") != "1024" {
			t.Errorf("unexpected Memory header: got %s, want %s", r.Header.Get("Memory"), "1024")
		}

		if r.Header.Get("Agent-Addr") != "example.com" {
			t.Errorf("unexpected Agent-Addr header: got %s, want %s", r.Header.Get("Agent-Addr"), "example.com")
		}

		if r.Header.Get("Target-Type") != "container" {
			t.Errorf("unexpected Target-Type header: got %s, want %s", r.Header.Get("Target-Type"), "container")
		}

		if r.Header.Get("Pod-Name") != "testpod" {
			t.Errorf("unexpected Pod-Name header: got %s, want %s", r.Header.Get("Pod-Name"), "testpod")
		}

		if r.Header.Get("Container-Name") != "testcontainer" {
			t.Errorf("unexpected Container-Name header: got %s, want %s", r.Header.Get("Container-Name"), "testcontainer")
		}

		if r.Header.Get("Container-Id") != "testcontainerid" {
			t.Errorf("unexpected Container-Id header: got %s, want %s", r.Header.Get("Container-Id"), "testcontainerid")
		}

		// Upgrade to websocket connection.
		upgrader := websocket.Upgrader{}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("failed to upgrade to websocket connection: %v", err)
		}

		defer conn.Close()
	}))

	// Set up test data.
	urlPath := &url.URL{Scheme: "wss", Host: server.Listener.Addr().String(), Path: "/exec"}
	header := &http.Header{
		"Session-Id":            []string{"testsession"},
		"User-Name":             []string{"testuser"},
		"Login-Name":            []string{"testlogin"},
		"Login-Group":           []string{"testgroup"},
		"App-Name":              []string{"testapp"},
		"Ip-Address":            []string{"1.2.3.4"},
		"Interactive":           []string{"true"},
		"Tty":                   []string{"true"},
		"Command":               []string{"ls -l"},
		"Command-Base64-Encode": []string{"bHMKLC1s"},
		"Cpus":                  []string{"1"},
		"Memory":                []string{"1024"},
		"Agent-Addr":            []string{"example.com"},
		"Target-Type":           []string{"container"},
		"Pod-Name":              []string{"testpod"},
		"Container-Name":        []string{"testcontainer"},
		"Container-Id":          []string{"testcontainerid"},
	}
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	// conn := &websocket.Conn{}

	// Call function being tested.
	wsConn, err := (&Client{}).dialAgent(nil, urlPath, header, tlsConfig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify results.
	if wsConn == nil {
		t.Errorf("unexpected nil websocket connection")
	}
}
