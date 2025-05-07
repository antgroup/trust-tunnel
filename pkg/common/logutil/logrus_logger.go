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
	"github.com/sirupsen/logrus"
)

// newLogrusLogger creates a new logrus logger with a daily roll writer output.
// The `moduleName` parameter is used as the log file name prefix.
// The `level` parameter sets the log level.
func newLogrusLogger(moduleName string) *logrus.Logger {
	l := logrus.New()

	l.Out = newDailyRollWriter(moduleName)
	l.Level = level

	return l
}

// setStaticFileForDailyRollWriter sets the `staticFile` property of the daily roll writer output
// of the given logger to the specified value.
func setStaticFileForDailyRollWriter(logger *logrus.Logger, static bool) {
	// Check if the logger's output is of type `dailyRollWriter`
	if drw, ok := logger.Out.(*dailyRollWriter); ok {
		drw.staticFile = static
	}
}
