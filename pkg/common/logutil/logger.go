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
	"os"
	"sync"

	"github.com/sirupsen/logrus"
)

// Constants for environment variable keys.
const (
	EnvKeyEnableStdout = "DAILY_ROLL_LOGGERS_ENABLE_STDOUT"
	EnvKeyLogLevel     = "DAILY_ROLL_LOGGERS_LOG_LEVEL"
)

// Variables for storing loggers and settings.
var (
	logMap       = make(map[string]*logrus.Logger)
	locker       = &sync.Mutex{}
	enableStdout = true
	level        = logrus.DebugLevel
)

// init initializes the logger settings based on environment variables.
func init() {
	_enableStdout := os.Getenv(EnvKeyEnableStdout)

	if _enableStdout == "false" {
		enableStdout = false
	}

	_levelStr := os.Getenv(EnvKeyLogLevel)
	if _level, err := logrus.ParseLevel(_levelStr); nil == err {
		level = _level
	}
}

// SetEnableStdout sets whether to enable logging to stdout.
func SetEnableStdout(enable bool) {
	enableStdout = enable
}

// SetLevel sets the logging level for all loggers.
func SetLevel(l logrus.Level) {
	locker.Lock()
	defer locker.Unlock()

	for _, theLogger := range logMap {
		theLogger.Level = l
	}

	level = l
}

// SetStaticFile sets whether to use a static log file name for all loggers.
func SetStaticFile(static bool) {
	locker.Lock()
	defer locker.Unlock()

	for _, theLogger := range logMap {
		setStaticFileForDailyRollWriter(theLogger, static)
	}
}

// SetExpireDay sets the number of days after which log files should expire.
func SetExpireDay(days int) {
	if days <= 0 || days >= 365 {
		return
	}

	expireDay = days
}

// GetLogger returns the logger for the given module name, creating it if it doesn't exist.
func GetLogger(moduleName string) *logrus.Logger {
	locker.Lock()
	defer locker.Unlock()

	l, exist := logMap[moduleName]
	if exist {
		return l
	}

	logger := newLogrusLogger(moduleName)
	logMap[moduleName] = logger

	return logger
}
