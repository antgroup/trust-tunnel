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

package logutil

import (
	"bytes"

	"github.com/sirupsen/logrus"
)

const (
	maxLength = 512
)

// CmdLogger represents a logger for command output.
type CmdLogger struct {
	buf    []byte
	cmdCh  chan []byte
	doneCh chan struct{}
	l      *logrus.Entry
}

// NewCmdLogger creates a new CmdLogger instance.
func NewCmdLogger(l *logrus.Entry) *CmdLogger {
	cmdL := &CmdLogger{
		buf:    make([]byte, 0, maxLength),
		cmdCh:  make(chan []byte, 50),
		doneCh: make(chan struct{}),
		l:      l,
	}
	go cmdL.log()

	return cmdL
}

// Write writes the command output to the logger.
func (cmdLogger *CmdLogger) Write(p []byte) (int, error) {
	cmdLogger.cmdCh <- p

	return len(p), nil
}

// Destroy closes the logger.
func (cmdLogger *CmdLogger) Destroy() {
	close(cmdLogger.doneCh)
}

// log processes the command output and logs it.
func (cmdLogger *CmdLogger) log() {
	for {
		var p []byte
		select {
		case <-cmdLogger.doneCh:
			// Goroutine quit.
			return
		case p = <-cmdLogger.cmdCh:
			if p == nil {
				// Error case.
				cmdLogger.l.Errorf("BUG: unexpected closure of cmd log channel")

				return
			}
		}

		for {
			if len(p) == 0 {
				break
			}
			// Append p to the buffer.
			leftSpace := maxLength - len(cmdLogger.buf)
			if leftSpace >= len(p) {
				cmdLogger.buf = append(cmdLogger.buf, p...)
				p = []byte{}
			} else {
				cmdLogger.buf = append(cmdLogger.buf, p[:leftSpace]...)
				p = p[leftSpace:]
			}
			// If '\r\n' is found, flush the buffer.
			newline := bytes.IndexAny(cmdLogger.buf, "\r\n")
			if newline != -1 {
				// Flush contents until the \r\n.
				cmdLogger.l.Infof("Cmd: %s", string(cmdLogger.buf[:newline]))
				// Keep the remaining contents in cmdLogger.buf.
				if newline+1 < len(cmdLogger.buf) {
					// Keep the remaining bytes.
					cmdLogger.buf = cmdLogger.buf[newline+1:]
				} else {
					// Empty the buffer.
					cmdLogger.buf = cmdLogger.buf[:0]
				}
			} else if len(cmdLogger.buf) == maxLength {
				// Flush the full log buffer.
				cmdLogger.l.Infof("Cmd: %s", string(cmdLogger.buf))
				// Empty the buffer.
				cmdLogger.buf = cmdLogger.buf[:0]
			}
		}
	}
}
