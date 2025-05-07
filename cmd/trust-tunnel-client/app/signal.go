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

//go:build linux || darwin

package app

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"golang.org/x/term"
	client "trust-tunnel/pkg/trust-tunnel-client"
)

const channelSize = 10

// setupSignal listens for window size change signals and adjusts the client session size accordingly.
func setupSignal(session client.Session) {
	sigCh := make(chan os.Signal, channelSize)
	signal.Notify(sigCh, syscall.SIGWINCH)

	go func() {
		for {
			sig := <-sigCh

			if sig == syscall.SIGWINCH {
				w, h, _ := term.GetSize(int(os.Stdin.Fd()))

				err := session.Resize(h, w)
				if err != nil {
					logrus.Errorf("failed to resize window: %v", err)
				}
			}
		}
	}()
}
