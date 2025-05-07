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
	"os"
	"testing"
)

func TestGetUserInfo(t *testing.T) {
	// Set up test data.
	username := "testuser"

	passwdPath := "./passwd"

	passwdContent := "root:x:0:0:root:/root:/bin/bash\ntestuser:x:1000:1000:Test User:/home/testuser:/bin/bash\n"

	expectedUID := "1000"

	expectedGID := "1000"

	expectedLoginDir := "/home/testuser"

	// Write test data to temporary file.
	file, err := os.OpenFile(passwdPath, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	defer os.Remove(passwdPath)
	defer file.Close()
	file.WriteString(passwdContent)

	// Call function being tested.
	uid, gid, loginDir, err := GetUserInfo(username, passwdPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify results.
	if uid != expectedUID {
		t.Errorf("unexpected uid: %s", uid)
	}

	if gid != expectedGID {
		t.Errorf("unexpected gid: %s", gid)
	}

	if loginDir != expectedLoginDir {
		t.Errorf("unexpected loginDir: %s", loginDir)
	}
}
