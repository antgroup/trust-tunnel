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
	"net"
	"os"
	"strings"
)

// GetIPAddrs gets the ip addresses of the host.
func GetIPAddrs() ([]string, error) {
	ret := make([]string, 0)

	adds, err := net.InterfaceAddrs()
	if err != nil {
		return ret, err
	}

	for _, a := range adds {
		if aspnet, ok := a.(*net.IPNet); ok && !aspnet.IP.IsLoopback() {
			if aspnet.IP.To4() != nil {
				ret = append(ret, aspnet.IP.String())
			}
		}
	}

	return ret, nil
}

// FindNonPrivateIP retrieves an IP address that is not in the 192.168 subnet from a given list of IP addresses.
func FindNonPrivateIP(ipAdds []string) string {
	if len(ipAdds) == 0 {
		return ""
	}

	for _, ip := range ipAdds {
		if !strings.Contains(ip, "192.168") {
			return ip
		}
	}

	return ipAdds[0]
}

func GetMainIP() string {
	var primaryIP string

	ipAdds, err := GetIPAddrs()
	if err != nil {
		return ""
	}

	if len(ipAdds) > 0 {
		primaryIP = FindNonPrivateIP(ipAdds)
	}

	return primaryIP
}

// GetHostName gets the hostname of the host.
func GetHostName() (string, error) {
	name, err := os.Hostname()
	if err != nil {
		return "", err
	}

	return name, nil
}
