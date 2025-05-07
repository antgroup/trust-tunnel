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
	"sync"
	"time"
	"trust-tunnel/pkg/common/logutil"
	"trust-tunnel/pkg/trust-tunnel-agent/session"

	"github.com/gorilla/websocket"
)

// SessionConfig is a structure for session configuration, used to store information related to session configurations.
type SessionConfig struct {
	// PhysTunnel specifies the way to establish the physical tunnel, which can be either "nsenter" or "sshd".
	PhysTunnel string `toml:"phys_tunnel"`

	// DelayReleaseSessionTimeout defines the timeout duration for delaying session release.
	DelayReleaseSessionTimeout time.Duration `toml:"delay_release_session_timeout"`
}

// StaleSession represents a stale session that needs to be released.
type StaleSession struct {
	userName string
	sess     session.Session
	// Death count down.
	deathClock       <-chan time.Time
	isSidecarSession bool
}

// Connection represents a client connection, encapsulating the management of session and websocket connections.
type Connection struct {
	// sess represents the client's session, used for maintaining session state.
	sess session.Session
	// conn represents the client's websocket connection, used for sending and receiving messages.
	conn *websocket.Conn
	// cmdLogger is used for logging command operations, providing detailed operation records.
	cmdLogger *logutil.CmdLogger
	errCh     chan error
	doneCh    chan struct{}
	lock      sync.Mutex
}

// delayReleaseSession periodically checks for stale sessions and releases them if they are outdated.
func (handler *Handler) delayReleaseSession() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		handler.lock.Lock()
		for id, staleSess := range handler.staleSessions {
			select {
			case <-staleSess.deathClock:
				logger.Debugf("session %s is outdated, let's release it", id)

				err := handler.releaseSession(id, staleSess.sess)
				if err == nil && staleSess.isSidecarSession {
					handler.currentSidecarNum--
				}
			default:
			}
		}
		handler.lock.Unlock()
	}
}

// releaseSession releases the given session and removes it from the stale sessions list.
func (handler *Handler) releaseSession(id string, sess session.Session) error {
	logger.Debugf("release session %s", id)

	// Clean up the session.
	err := sess.Clean()
	if err != nil {
		logger.Errorf("clean session err:%v", err)
	}

	// Remove the session from the stale sessions list.
	delete(handler.staleSessions, id)

	return err
}
