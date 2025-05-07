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
	"encoding/json"
	"io"
	"strings"

	"github.com/gorilla/websocket"
	client "trust-tunnel/pkg/trust-tunnel-client"
)

// processLocalOutput handles local output by preparing and sending a normal session closure message.
func (sessConn *Connection) processLocalOutput() {
	err := sessConn.processOutOrErr(false)
	// Close the connection in output processing.
	msg := client.NormalCloseMessage{
		Code: sessConn.sess.ExitCode(),
	}

	if err != nil {
		if !strings.Contains(err.Error(), "close sent") {
			// normal closed
			msg.Err = err
		}
	}

	data, _ := json.Marshal(msg)

	sessConn.lock.Lock()
	defer sessConn.lock.Unlock()
	sessConn.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, truncWebsocketErrMsg(string(data))))
}

func (sessConn *Connection) processLocalError() {
	sessConn.processOutOrErr(true)
}

// processOutOrErr handles the session's output or errors.
// processErr indicates whether it's processing stderr or stdout.
func (sessConn *Connection) processOutOrErr(processErr bool) error {
	defer func() {
		if processErr {
			sessConn.sess.StderrDone()
		} else {
			sessConn.sess.StdoutDone()
		}
	}()

	for {
		select {
		case <-sessConn.doneCh:
			return nil
		default:
		}

		var (
			// Read from cmd in container or host.
			cmdReader io.Reader

			err error
		)

		if processErr {
			cmdReader, err = sessConn.sess.NextStderr()
		} else {
			cmdReader, err = sessConn.sess.NextStdout()
		}

		if err != nil {
			if err == io.EOF {
				// Connection closed
				return nil
			}

			logger.Errorf("get output reader from cmd failed with error %v , reader = %v,isErr = %v", err, cmdReader, processErr)

			return err
		}

		if err = sessConn.write(cmdReader, processErr); err != nil {
			return err
		}
	}
}

// write is used to send data to the websocket connection.
// reader: the data source to be sent.
// isErr: indicates whether the data being sent is an error message.
func (sessConn *Connection) write(reader io.Reader, isErr bool) error {
	// If the reader is nil, there's no data to send, so return nil directly.
	if reader == nil {
		return nil
	}
	// Writer for websocket client.
	var (
		msgWriter io.WriteCloser
		err       error
	)

	sessConn.lock.Lock()
	defer sessConn.lock.Unlock()

	if isErr {
		msgWriter, err = sessConn.conn.NextWriter(websocket.TextMessage)
	} else {
		msgWriter, err = sessConn.conn.NextWriter(websocket.BinaryMessage)
	}

	// Ensure the message writer is closed to avoid resource leaks.
	defer func() {
		if msgWriter != nil {
			msgWriter.Close()
		}
	}()

	if err != nil {
		logger.Errorf("get websocket writer failed: %v,isErr %v", err, isErr)

		return err
	}

	// Copy data from reader to msgWriter. If reader is not nil, because the check is done above.
	var n int64

	if reader != nil {
		n, err = io.Copy(msgWriter, reader)
		if err != nil {
			logger.Errorf("copy message to websocket failed: %v", err)

			return err
		}
	}

	logger.Tracef("write output back to websocket %d bytes", n)

	return nil
}
