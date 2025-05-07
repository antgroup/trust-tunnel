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

package client

import (
	"bytes"
	"io"
	"testing"
)

func TestBlockingBuffer_Read(t *testing.T) {
	bb := NewBlockingBuffer()
	expected := []byte("testdata")
	bb.Write(expected)

	// Read the data from the buffer
	readData := make([]byte, len(expected))

	n, err := bb.Read(readData)
	if err != nil {
		t.Errorf("Unexpected error during read: %v", err)
	}

	if n != len(expected) {
		t.Errorf("Expected read length %d, got %d", len(expected), n)
	}

	if !bytes.Equal(readData[:n], expected) {
		t.Errorf("Expected read data %v, got %v", expected, readData[:n])
	}
}

func TestBlockingBuffer_ReadWriteClose(t *testing.T) {
	bb := NewBlockingBuffer()

	// Write data to the buffer
	expected := []byte("testdata")

	n, err := bb.Write(expected)
	if err != nil {
		t.Errorf("Unexpected error during write: %v", err)
	}

	if n != len(expected) {
		t.Errorf("Expected write length %d, got %d", len(expected), n)
	}

	// Read the data from the buffer
	readData := make([]byte, len(expected))

	n, err = bb.Read(readData)
	if err != nil {
		t.Errorf("Unexpected error during read: %v", err)
	}

	if n != len(expected) {
		t.Errorf("Expected read length %d, got %d", len(expected), n)
	}

	if !bytes.Equal(readData[:n], expected) {
		t.Errorf("Expected read data %v, got %v", expected, readData[:n])
	}

	// Close the buffer and ensure that Read returns EOF
	bb.Close()

	readData = make([]byte, len(expected))

	n, err = bb.Read(readData)
	if n != 0 || err != io.EOF {
		t.Errorf("Expected EOF after close, got n=%d, err=%v", n, err)
	}
}
