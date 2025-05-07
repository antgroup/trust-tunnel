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
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/gorilla/websocket"
)

// agentConn represents a connection to an agent over a websocket.
type agentConn struct {
	conn        *websocket.Conn
	mu          sync.Mutex
	interactive bool
	tty         bool
	// Buffer to store standard output.
	stdoutBuffer *BlockingBuffer
	// Buffer to store standard error.
	stderrBuffer *BlockingBuffer
	err          error
	// Exit code returned on connection close.
	exitCode int
}

// closeHandler handles the event of the websocket closing.
func (ac *agentConn) closeHandler(code int, text string) error {
	if code == websocket.CloseNormalClosure {
		var closeMsg NormalCloseMessage

		err := json.Unmarshal([]byte(text), &closeMsg)
		if err != nil {
			if text != "" {
				// Old CloseNormalClosure message can't be unmarshaled, so we return nil
				// for keeping backward compatibility instead of an error.
				ac.err = fmt.Errorf("%s", text)
				ac.exitCode = -1
			}

			return nil
		}

		ac.exitCode = closeMsg.Code
		ac.err = closeMsg.Err
	} else {
		ac.exitCode = -1
		ac.err = fmt.Errorf("%s", text)
	}

	return nil
}

// ProcessMsg processes incoming websocket messages and writes
// them to the corresponding stdout or stderr buffers.
func (ac *agentConn) ProcessMsg() {
	ac.conn.SetCloseHandler(ac.closeHandler)

	for {
		messageType, message, err := ac.conn.ReadMessage()
		if err != nil {
			ac.err = err
			ac.stdoutBuffer.Close()
			ac.stderrBuffer.Close()

			return
		}

		switch messageType {
		case websocket.BinaryMessage:
			ac.stdoutBuffer.Write(message)
		case websocket.TextMessage:
			ac.stderrBuffer.Write(message)
		}
	}
}

// Read reads from the stdout buffer of the agent connection.
func (ac *agentConn) Read(p []byte) (int, error) {
	n, err := ac.stdoutBuffer.Read(p)
	if err != io.EOF {
		return n, err
	}

	return 0, ac.err
}

// ReadStderr reads from the stderr buffer of the agent connection.
func (ac *agentConn) ReadStderr(p []byte) (int, error) {
	n, err := ac.stderrBuffer.Read(p)
	if err != io.EOF {
		return n, err
	}

	return 0, ac.err
}

// Write sends the provided bytes as a websocket message.
func (ac *agentConn) Write(p []byte) (int, error) {
	if !ac.interactive {
		return len(p), nil
	}

	ac.mu.Lock()
	defer ac.mu.Unlock()

	if err := ac.conn.WriteMessage(websocket.BinaryMessage, p); err != nil {
		return 0, err
	}

	return len(p), nil
}

// Close closes the websocket connection.
func (ac *agentConn) Close() error {
	return ac.conn.Close()
}

// Resize sends a resize message over the websocket connection.
func (ac *agentConn) Resize(height int, width int) error {
	msg := fmt.Sprintf("resize: %d,%d", height, width)

	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.conn.WriteMessage(websocket.TextMessage, []byte(msg))

	return nil
}

// CloseSession sends a close session message over the websocket connection.
func (ac *agentConn) CloseSession() error {
	msg := "close session"

	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.conn.WriteMessage(websocket.TextMessage, []byte(msg))

	return nil
}

// ExitCode returns the exit code after the connection is closed.
func (ac *agentConn) ExitCode() int {
	return ac.exitCode
}
