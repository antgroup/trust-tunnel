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
	"fmt"
	"strings"
)

const (
	maxContainerIDLength = 6
)

// WrapContainerError wraps an error message with a container ID, providing a more descriptive error when applicable.
func WrapContainerError(errMsg string, containerID string) string {
	if len(containerID) > maxContainerIDLength {
		containerID = containerID[0:maxContainerIDLength]
	}

	switch {
	case strings.Contains(errMsg, "No such container") || strings.Contains(errMsg, "not found"):
		errMsg = fmt.Sprintf("can't find container:%s", containerID)

	case strings.Contains(errMsg, "is not running"):
		errMsg = fmt.Sprintf("container is not running:%s", containerID)

	case strings.Contains(errMsg, "no such file or directory") || strings.Contains(errMsg, "connection refused"):
		errMsg = "docker is unavailable"
	}

	return errMsg
}

// WrapErrorWithCode assigns an error code to an error message based on its content.
// errMsg: The original error message.
// Returns: An error message prefixed with an error code.
func WrapErrorWithCode(errMsg string) string {
	var code string

	switch {
	case strings.Contains(errMsg, "no space left on device"):
		code = "MA_513"
	case strings.Contains(errMsg, "visit authorization server failed"):
		code = "MA_518"
	case strings.Contains(errMsg, "verify client certificate error"):
		code = "MA_519"
	case strings.Contains(errMsg, "current sidecar num exceed the limit"):
		code = "MA_521"
	case strings.Contains(errMsg, "can't find container"):
		code = "MA_522"
	case strings.Contains(errMsg, "container is not running"):
		code = "MA_523"
	case strings.Contains(errMsg, "docker daemon is unavailable"):
		code = "MA_524"
	case strings.Contains(errMsg, "is not permitted to login on host"):
		code = "MA_525"
	case strings.Contains(errMsg, "user does not exist"):
		code = "MA_526"
	case strings.Contains(errMsg, "nsenter host namespace failed"):
		code = "MA_527"
	case strings.Contains(errMsg, "SSH public key insert error"):
		code = "MA_528"
	case strings.Contains(errMsg, "SSH private key read error"):
		code = "MA_529"
	case strings.Contains(errMsg, "SSH private key parse error"):
		code = "MA_530"
	case strings.Contains(errMsg, "SSH connect error"):
		code = "MA_531"
	default:
		code = "MA_-1"
	}

	return fmt.Sprintf("code=%s,msg=%s", code, errMsg)
}
