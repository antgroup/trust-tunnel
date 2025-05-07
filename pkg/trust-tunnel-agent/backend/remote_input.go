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

package backend

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
)

const (
	resizeHeader = "resize: "
	closeHeader  = "close session"
)

// processRemoteInput processes incoming messages from a remote connection.
// It continuously reads messages from the connection and dispatches them to appropriate handlers based on message type.
// This function runs until the connection is closed or an error occurs.
func (sessConn *Connection) processRemoteInput() {
	defer func() {
		// Do not clean the session, we might reuse it later.
		// s.Clean()
		close(sessConn.doneCh)
		close(sessConn.errCh)
	}()

	for {
		msgType, msgReader, err := sessConn.conn.NextReader()
		if err != nil {
			if closeErr, ok := err.(*websocket.CloseError); ok && closeErr.Code == websocket.CloseNormalClosure {
				// normal close, ignore error
				return
			}
			// Network connection closed indicates IO closing, so do "unexpected EOF"
			if strings.Contains(err.Error(), "use of closed network connection") ||
				strings.Contains(err.Error(), "unexpected EOF") {
				return
			}

			sessConn.errCh <- fmt.Errorf("read from remote error: %v", err)

			return
		}

		// Handling resize message.
		if msgType == websocket.TextMessage {
			msg := make([]byte, 128)

			n, err := msgReader.Read(msg)
			if err != nil {
				sessConn.errCh <- fmt.Errorf("read from remote error: %v", err)

				return
			}

			msg = msg[:n]

			if bytes.HasPrefix(msg, []byte(resizeHeader)) {
				msg = bytes.TrimPrefix(msg, []byte(resizeHeader))

				vals := bytes.Split(msg, []byte(","))
				if len(vals) == 2 {
					h, _ := strconv.Atoi(string(vals[0]))
					w, _ := strconv.Atoi(string(vals[1]))

					if h > 0 && w > 0 {
						sessConn.sess.Resize(h, w)
					}
				}
			} else if bytes.HasPrefix(msg, []byte(closeHeader)) {
				logger.Debug("received close message,return")

				return
			}

			continue
		}

		if msgType != websocket.BinaryMessage {
			continue
		}

		cmdStdin, err := sessConn.sess.NextStdin()
		if err != nil || cmdStdin == nil {
			sessConn.errCh <- fmt.Errorf("got cmd's stdin error: %v", err)

			return
		}

		// teeReader is used for logging cmd from user input.
		teeReader := io.TeeReader(msgReader, sessConn.cmdLogger)

		n, err := io.Copy(cmdStdin, teeReader)
		if err != nil {
			sessConn.errCh <- fmt.Errorf("copy data from websocket to cmd's stdin failed: %v", err)

			return
		}

		logger.Tracef("write to cmd's stdin %d bytes", n)
	}
}
