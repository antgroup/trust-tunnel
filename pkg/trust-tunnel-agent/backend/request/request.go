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

package request

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	client "trust-tunnel/pkg/trust-tunnel-client"
)

type Info struct {
	SessionID        string            `json:"session_id"`
	AgentAddr        string            `json:"agent_addr"`
	UserName         string            `json:"user_name"`
	LoginName        string            `json:"login_name"`
	LoginGroup       string            `json:"login_group"`
	TargetType       client.TargetType `json:"target_type"`
	PodName          string            `json:"pod_name"`
	ContainerID      string            `json:"container_id"`
	ContainerName    string            `json:"container_name"`
	Interactive      bool              `json:"interactive"`
	Tty              bool              `json:"tty"`
	Cmd              []string          `json:"cmd"`
	UseBase64        bool              `json:"use_base64"`
	IPAddress        string            `json:"ip_address"`
	AppName          string            `json:"app_name"`
	Cpus             float64           `json:"cpus"`
	MemoryMB         int               `json:"memory_mb"`
	DisableCleanMode bool              `json:"disable_clean_mode"`
}

// String returns the JSON representation of the request information.
func (r *Info) String() string {
	b, _ := json.Marshal(*r)

	return string(b)
}

// GetRequestInfo extracts the request information from the HTTP request headers.
func GetRequestInfo(r *http.Request) (*Info, error) {
	var info Info

	var err error

	tmp := r.Header["Session-Id"]
	if len(tmp) > 0 {
		info.SessionID = tmp[0]
	}

	tmp = r.Header["Agent-Addr"]
	if len(tmp) > 0 {
		info.AgentAddr = tmp[0]
	}

	tmp = r.Header["User-Name"]
	if len(tmp) > 0 {
		info.UserName = tmp[0]
	}

	tmp = r.Header["App-Name"]
	if len(tmp) > 0 {
		info.AppName = tmp[0]
	}

	tmp = r.Header["Ip-Address"]
	if len(tmp) > 0 {
		info.IPAddress = tmp[0]
	}

	tmp = r.Header["Login-Name"]
	if len(tmp) > 0 {
		info.LoginName = tmp[0]
	}

	tmp = r.Header["Login-Group"]
	if len(tmp) > 0 {
		info.LoginGroup = tmp[0]
	}

	tmp = r.Header["Target-Type"]
	if len(tmp) > 0 {
		if tmp[0] == "physical" {
			info.TargetType = client.TargetPhys
		} else if tmp[0] == "container" {
			info.TargetType = client.TargetContainer
		} else {
			return nil, fmt.Errorf("request error: invalid target type")
		}
	}

	if info.TargetType == client.TargetContainer {
		tmp = r.Header["Pod-Name"]
		if len(tmp) == 0 {
			return nil, fmt.Errorf("request error: no pod name of container target")
		}

		info.PodName = tmp[0]

		tmp = r.Header["Container-Id"]
		if len(tmp) > 0 {
			info.ContainerID = tmp[0]
		}

		tmp = r.Header["Container-Name"]
		if len(tmp) > 0 {
			info.ContainerName = tmp[0]
		}
	}

	tmp = r.Header["Interactive"]
	if len(tmp) > 0 {
		info.Interactive, err = strconv.ParseBool(tmp[0])
		if err != nil {
			return nil, fmt.Errorf("request error: invalid interactive argument: %v", err)
		}
	}

	tmp = r.Header["Tty"]
	if len(tmp) > 0 {
		info.Tty, err = strconv.ParseBool(tmp[0])
		if err != nil {
			return nil, fmt.Errorf("request error: invalid tty argument: %v", err)
		}
	}

	tmp = r.Header["Command-Base64-Encode"]
	if len(tmp) == 0 {
		tmp = r.Header["Command"]
		if len(tmp) == 0 {
			return nil, fmt.Errorf("request error: no command")
		}

		info.Cmd = tmp
	} else {
		// base64 decode
		var decodedCommand []string

		for _, encodedString := range tmp {
			decodedData, err := base64.StdEncoding.DecodeString(encodedString)
			if err != nil {
				return nil, fmt.Errorf("decoding command error:%v", err)
			}

			decodedCommand = append(decodedCommand, string(decodedData))
		}

		info.UseBase64 = true

		info.Cmd = decodedCommand
	}

	tmp = r.Header["Cpus"]
	if len(tmp) > 0 {
		info.Cpus, err = strconv.ParseFloat(tmp[0], 64)
		if err != nil {
			return nil, fmt.Errorf("request error: invalid cpus argument: %v", err)
		}
	}

	tmp = r.Header["Memory"]
	if len(tmp) > 0 {
		info.MemoryMB, err = strconv.Atoi(tmp[0])
		if err != nil {
			return nil, fmt.Errorf("request error: invalid memoryMB argument: %v", err)
		}
	}

	tmp = r.Header["Disable-Clean-Mode"]
	if len(tmp) > 0 && tmp[0] == "1" {
		info.DisableCleanMode = true
	}

	return &info, nil
}
