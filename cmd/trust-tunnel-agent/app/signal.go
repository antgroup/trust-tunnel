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

package app

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
)

const channelSize = 10

// setupSignal initializes a signal channel to listen for SIGINT and SIGTERM signals
// and handles these signals to ensure the program can exit gracefully or immediately as needed.
func setupSignal() {
	sigCh := make(chan os.Signal, channelSize)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for {
			sig := <-sigCh
			switch sig {
			case syscall.SIGINT:
				logrus.Infof("Got SIGINT, quit with grace")
				os.Exit(0)
			case syscall.SIGTERM:
				logrus.Infof("Got SIGTERM, quit immediately")
				os.Exit(0)
			}
		}
	}()
}
