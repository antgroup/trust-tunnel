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
	"sync"
	"time"
)

// BlockingBuffer is a synchronized buffer that blocks Read operations until data is available.
type BlockingBuffer struct {
	readBuffer  *bytes.Buffer
	writeBuffer *bytes.Buffer
	// Signal channel for notifying data availability.
	signal chan struct{}
	lock   sync.RWMutex
}

// NewBlockingBuffer initializes a new instance of BlockingBuffer.
func NewBlockingBuffer() *BlockingBuffer {
	return &BlockingBuffer{
		readBuffer:  bytes.NewBuffer(nil),
		writeBuffer: bytes.NewBuffer(nil),
		signal:      make(chan struct{}, 1),
	}
}

// Read reads data from the buffer, blocking if the buffer is empty until data becomes available.
func (b *BlockingBuffer) Read(p []byte) (n int, err error) {
	for {
		b.lock.RLock()
		if b.readBuffer == nil {
			return 0, io.EOF
		}

		n, err = b.readBuffer.Read(p)
		b.lock.RUnlock()

		// Two conditions:
		// err is nil: read is finished successfully.
		// err is some real problem: return error.
		if err != io.EOF {
			return
		}

		// err is io.EOF indicates that buffer is drained, wait for next read signal.
		_, ok := <-b.signal
		if !ok {
			// Channel is closed.
			return 0, io.EOF
		}

		// Move writeBuffer to readBuffer.
		b.lock.Lock()
		// Closed.
		if b.readBuffer == nil {
			return 0, io.EOF
		}
		// Write buffer turns to be read buffer now and old read buffer will be cleaned by GC.
		b.readBuffer.Reset()
		b.readBuffer = b.writeBuffer
		// Make a new write buffer.
		b.writeBuffer = &bytes.Buffer{}
		b.lock.Unlock()
	}
}

// Write writes data into the buffer and notifies readers that data is available.
func (b *BlockingBuffer) Write(p []byte) (n int, err error) {
	b.lock.Lock()
	// Closed.
	if b.writeBuffer == nil {
		return 0, io.EOF
	}

	n, err = b.writeBuffer.Write(p)
	b.lock.Unlock()

	// Send signal that if there are contents to read.
	select {
	case b.signal <- struct{}{}:
	default:
	}

	return
}

// Close gracefully shuts down the buffer, ensuring all written data is read before closing.
func (b *BlockingBuffer) Close() error {
	for b.readBuffer.Len() != 0 || b.writeBuffer.Len() != 0 {
		// Wait for all data to be read before close the buffer.
		time.Sleep(time.Millisecond * 100)
	}
	close(b.signal)
	b.lock.Lock()

	b.readBuffer.Reset()
	b.writeBuffer.Reset()
	b.readBuffer = nil
	b.writeBuffer = nil
	b.lock.Unlock()

	return nil
}
