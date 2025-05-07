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
	"encoding/json"
	"time"
	"trust-tunnel/pkg/common/logutil"
	"trust-tunnel/pkg/common/sessionutil"
	"trust-tunnel/pkg/trust-tunnel-agent/backend/request"
)

var auditLogger = logutil.GetLogger("trust-tunnel-audit")

// LogInfo records the login and operation information of a user.
type LogInfo struct {
	// Cmd represents the command executed to the target.
	Cmd string `json:"cmd"`

	// GmtCreate represents the creation time of the log record, using the GMT format.
	GmtCreate string `json:"gmt_create"`

	// LoginTime represents the user's login time.
	LoginTime string `json:"login_time"`

	// LoginIP represents the IP address of the target.
	LoginIP string `json:"login_ip"`

	// UserName represents the login name to the target.
	UserName string `json:"username"`

	// HostName represents the hostname of the target.
	HostName string `json:"hostname"`

	// SessionID represents the session identifier for the session.
	SessionID string `json:"session_id"`

	// SrcIP represents the source IP address of the session request.
	SrcIP string `json:"src_ip"`

	// SrcPort represents the source port of the session request.
	SrcPort int `json:"src_port"`
}

// constructAuditInfo generates the audit log of the specified struct.
func constructAuditInfo(req *request.Info) {
	agentAddr := sessionutil.GetMainIP()
	logInfo := LogInfo{
		SessionID: req.SessionID,
		UserName:  req.LoginName,
	}

	if req.TargetType == 0 {
		logInfo.LoginIP = agentAddr
	} else {
		logInfo.LoginIP = req.IPAddress
	}

	logInfo.HostName, _ = sessionutil.GetHostName()

	var command string

	for _, v := range req.Cmd {
		command = command + v + " "
	}

	logInfo.Cmd = command
	timeNow := time.Now().Format("2006.01.02 15:04:05")
	logInfo.LoginTime = timeNow
	logInfo.GmtCreate = timeNow
	printLog(logInfo)
}

// printLog prints the log in the format of json string.
func printLog(info LogInfo) {
	b, err := json.Marshal(info)
	if err != nil {
		return
	}

	s := string(b)
	auditLogger.Info(s)
}
