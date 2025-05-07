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

package sessionutil

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	bufferSize                  = 4096
	expectedPasswdSegmentsCount = 7
)

type Process struct {
	PID          int
	PPID         int
	Name         string
	ChildProcess []*Process
}

// GetProcesses gets all the process in the system and return their process stats.
// Parse the /proc/$pid/stat file to get the process pid,ppid and name.
func GetProcesses() ([]*Process, error) {
	procDir := "/proc"

	// Read the /proc directory to obtain a list of entries.
	entries, err := os.ReadDir(procDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s directory: %v", procDir, err)
	}

	var processes []*Process

	// Iterate through all entries in the /proc directory.
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		// Construct the path to the process's stat file.
		statPath := filepath.Join(procDir, entry.Name(), "stat")

		statContent, err := os.ReadFile(statPath)
		if err != nil {
			continue
		}

		fields := strings.Fields(string(statContent))
		if len(fields) < 4 {
			continue
		}

		ppid, err := strconv.Atoi(fields[3])
		if err != nil {
			continue
		}

		// Extract the process name.
		name := strings.Trim(fields[1], "()")
		// Create a Process instance and add it to the slice.
		process := &Process{
			PID:  pid,
			PPID: ppid,
			Name: name,
		}
		processes = append(processes, process)
	}

	return processes, nil
}

// FindChildProcesses locates all child process IDs for a given parent process ID.
// It searches for all direct and indirect child processes of the specified parent.
func FindChildProcesses(targetPPID int, processes []*Process) []int {
	var pidList []int

	for _, process := range processes {
		if process.PPID == targetPPID {
			pidList = append(pidList, process.PID)
			pidList = append(pidList, FindChildProcesses(process.PID, processes)...)
		}
	}

	return pidList
}

// KillProcess sends SIGTERM to process.
func KillProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	err = process.Signal(syscall.SIGTERM)
	if err != nil {
		return err
	}

	go process.Wait()

	return nil
}

// GetProcessCmdLineByPID retrieves the command line arguments of a process given its PID.
func GetProcessCmdLineByPID(pid int) ([]string, error) {
	cmdlinePath := fmt.Sprintf("/proc/%d/cmdline", pid)

	data, err := os.ReadFile(cmdlinePath)
	if err != nil {
		return nil, err
	}

	cmdline := strings.Split(string(data), "\x00")

	return cmdline, nil
}

// KillProcessGroup terminates a process group identified by a parent process PID
// and a command line string. If the command line doesn't match or if 'inverted'
// is true, the killing order of child processes is reversed.
func KillProcessGroup(parentPID int, commandLine string, inverted bool) error {
	proc, err := os.FindProcess(parentPID)
	if err != nil {
		return err
	}

	// Attempt to send a signal 0 to the process to check if it's still running.
	// Signal 0 does not kill the process but can be used to check for its existence.
	err = proc.Signal(syscall.Signal(0))
	if err != nil {
		if strings.Contains(err.Error(), "process already finished") {
			return nil
		}

		return err
	}

	// Verify the command line of the process to ensure it matches the expected one.
	// This prevents killing a new process that reuses the PID of an old process.
	cmdLines, err := GetProcessCmdLineByPID(parentPID)
	if err != nil {
		return err
	}

	if commandLine != "" && !Contains(cmdLines, commandLine) {
		return nil
	}

	// Retrieve all child processes.
	allProcesses, err := GetProcesses()
	if err != nil {
		return err
	}

	childPIDs := FindChildProcesses(parentPID, allProcesses)

	if inverted {
		ReverseSlice(childPIDs)
	}

	// Terminate the child processes.
	for _, pid := range childPIDs {
		KillProcess(pid)
		// Throttle the killing to prevent overloading the system.
		time.Sleep(1 * time.Second)
	}

	return nil
}

// ReverseSlice reverses the order of integers in a slice.
func ReverseSlice(slice []int) {
	for i, j := 0, len(slice)-1; i < j; i, j = i+1, j-1 {
		slice[i], slice[j] = slice[j], slice[i]
	}
}

// Contains checks whether a string slice contains a specific string.
func Contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}

	return false
}

// findUser searches for user information in the specified password file by username.
// It returns the user's UID, GID, and login directory.
func findUser(username string, passwdPath string) (string, string, string, error) {
	file, err := os.Open(passwdPath)
	if err != nil {
		return "", "", "", fmt.Errorf("open host file (/etc/passwd) error: %w", err)
	}

	defer file.Close()

	buf := bufio.NewReader(file)

	// Read the file line by line until the end of the file.
	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return "", "", "", fmt.Errorf("read host file (/etc/passwd) error: %w", err)
			}
		}

		segs := strings.Split(line, ":")
		// Verify if this is a valid line of user information.
		if len(segs) != expectedPasswdSegmentsCount {
			continue
		}

		// Check if the current line corresponds to the specified username.
		// If so, extract the user ID, group ID, and login directory.
		if segs[0] != username {
			continue
		}

		// Check if the user is allowed to log in and has a valid shell.
		if !strings.Contains(line, "nologin") && strings.HasSuffix(segs[6], "sh\n") {
			uid, gid, loginDir := segs[2], segs[3], segs[5]

			return uid, gid, loginDir, nil
		}
	}

	return "", "", "", nil
}

// GetUserInfo retrieves the UID, GID, and login directory of a user given their username and the path to the password file.
func GetUserInfo(username string, passwdPath string) (string, string, string, error) {
	uid, gid, loginDir, err := findUser(username, passwdPath)
	if err != nil {
		return "", "", "", err
	}

	return uid, gid, loginDir, nil
}

// GetLoginDirAndIDs retrieves the UID, GID, and full path of the login directory for a user given their username.
func GetLoginDirAndIDs(username string, passwdPath string, rootfsPrefix string) (int, int, string, error) {
	uid, gid, loginDir, err := findUser(username, passwdPath)
	if err != nil {
		return 0, 0, "", fmt.Errorf("open host file (%s/etc/passwd) error: %w", rootfsPrefix, err)
	}

	if len(loginDir) == 0 {
		return 0, 0, "", fmt.Errorf("username %v is not permitted to login on host", username)
	}

	uidInt, _ := strconv.Atoi(uid)
	gidInt, _ := strconv.Atoi(gid)

	return uidInt, gidInt, rootfsPrefix + loginDir, nil
}

// OneRead reads data once from the provided Reader and returns a new Reader that can read the already read data.
// If there is no data to read or an error occurs, it returns an error.
func OneRead(origin io.Reader) (io.Reader, error) {
	var (
		buf    = make([]byte, bufferSize)
		reader io.Reader
	)

	n, err := origin.Read(buf)
	if n > 0 {
		reader = bytes.NewReader(buf[:n])

		return reader, nil
	}

	return nil, err
}
