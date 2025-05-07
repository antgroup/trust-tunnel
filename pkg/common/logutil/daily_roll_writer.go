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
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
	"time"
)

const (
	logFileDateLayout = "2006-01-02"
	defaultExpireDay  = 90
)

// Config represents the configuration for the dailyRollWriter.
type Config struct {
	Level      string `toml:"level"`
	ExpireDays int    `toml:"expire_days"`
}

var expireDay = defaultExpireDay

var (
	logDir = os.Getenv("DAILY_ROLL_LOGRUS_LOG_PATH")

	defaultLogPath = filepath.Join(os.Getenv("HOME"), "logs")
)

// init initializes the log directory by creating it if it doesn't exist.
func init() {
	if logDir == "" {
		logDir = defaultLogPath
	}

	if err := createLogDirIfNotExist(); err != nil {
		panic(err)
	}
}

// createLogDirIfNotExist creates the log directory if it doesn't exist.
func createLogDirIfNotExist() error {
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		return os.MkdirAll(logDir, os.ModePerm)
	}

	return nil
}

// newDailyRollWriter creates a new dailyRollWriter with the given prefix file name.
func newDailyRollWriter(prefixFileName string) *dailyRollWriter {
	ret := &dailyRollWriter{
		prefixFileName: prefixFileName,
		locker:         &sync.Mutex{},
	}

	runtime.SetFinalizer(ret, writerFinalizer)

	return ret
}

// dailyRollWriter represents a writer that rolls over to a new log file every day.
type dailyRollWriter struct {
	prefixFileName string
	current        string
	writer         *os.File
	locker         sync.Locker
	staticFile     bool
}

// initWriter initializes the writer by creating or opening the log file and setting it as the writer.
// If the log file does not exist, it creates a new one.
func (w *dailyRollWriter) initWriter() {
	w.locker.Lock()
	defer w.locker.Unlock()

	writerFinalizer(w)

	// Get the log file path based on whether it's a static file or a daily rolling file.
	logFile := w.getLogFilePath()

	// Open or create the log file and set it as the writer.
	log, err := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, os.FileMode(0o644))
	if err != nil {
		if os.IsNotExist(err) {
			log, err = os.Create(logFile)
		}

		if err != nil {
			panic(err)
		}
	}

	w.writer = log
}

// getLogFilePath returns the log file path based on whether it's a static file or a daily rolling file.
func (w *dailyRollWriter) getLogFilePath() string {
	if w.staticFile {
		return filepath.Join(logDir, fmt.Sprintf("%s.log", w.prefixFileName))
	}

	return filepath.Join(logDir, fmt.Sprintf("%s-%s.log", w.prefixFileName, w.current))
}

// Write writes the given byte slice to the log file. If the current date has changed since the last write,
// it initializes a new writer and starts a goroutine to clean up old log files.
func (w *dailyRollWriter) Write(p []byte) (int, error) {
	now := time.Now().Format(logFileDateLayout)

	if now != w.current {
		w.current = now

		w.initWriter()

		go cleanHistoryLogs()
	}

	if enableStdout {
		os.Stdout.Write(p)
	}

	return w.writer.Write(p)
}

// writerFinalizer closes the writer if it's not nil.
func writerFinalizer(w *dailyRollWriter) {
	if w.writer != nil {
		w.writer.Close()
	}
}

var logDateExp = regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)

// cleanHistoryLogs deletes log files that are older than the specified expiration date.
func cleanHistoryLogs() {
	logFiles, err := os.ReadDir(logDir)

	if nil != err {
		return
	}

	now := time.Now()
	expireDate := now.Add(-24 * time.Duration(expireDay) * time.Hour)

	for _, logFile := range logFiles {
		logDateStr := logDateExp.FindString(logFile.Name())

		if logDateStr == "" {
			continue
		}

		logDate, err := time.Parse(logFileDateLayout, logDateStr)

		if nil != err {
			continue
		}

		if expireDate.After(logDate) {
			fmt.Printf("Clean Log File %s\n", logFile.Name())
			os.Remove(path.Join(logDir, logFile.Name()))
		}
	}
}
