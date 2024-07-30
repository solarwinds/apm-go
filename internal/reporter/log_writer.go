// © 2023 SolarWinds Worldwide, LLC. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package reporter

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/solarwinds/apm-go/internal/log"
	"io"
	"os"
	"sync"
)

// FlushWriter offers an interface to write a byte slice to a specific destination.
// The caller needs to flush the buffer explicitly in async mode, while for sync
// mode, a flush is implicitly called after a write.
type FlushWriter interface {
	Write(t WriteType, bytes []byte) (int, error)
	Flush() error
}

func newLogWriter(syncWrite bool, dest io.Writer, maxChunkSize int) FlushWriter {
	return &logWriter{
		mu:           &sync.Mutex{},
		syncWrite:    syncWrite,
		dest:         dest,
		maxChunkSize: maxChunkSize,
	}
}

type LogData struct {
	Events  []string `json:"events,omitempty"`
	Metrics []string `json:"metrics,omitempty"`
}

// ServerlessMessage denotes the message to be written to AWS CloudWatch. The
// forwarder will decode the message and sent the messages to the SolarWinds Observability collector.
type ServerlessMessage struct {
	Data LogData `json:"apm-data"`
}

// logWriter writes the byte slices to a bytes buffer and flush the buffer when
// the trace ends. Note that it's for AWS Lambda only so there is no need to keep
// it concurrent-safe.
type logWriter struct {
	mu           *sync.Mutex
	syncWrite    bool
	dest         io.Writer
	maxChunkSize int
	chunkSize    int
	msg          ServerlessMessage
}

type WriteType int

const (
	EventWT = iota
	MetricWT
)

func (lr *logWriter) encode(bytes []byte) string {
	return base64.StdEncoding.EncodeToString(bytes)
}

func (lr *logWriter) Write(t WriteType, bytes []byte) (int, error) {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	encoded := lr.encode(bytes)
	if len(encoded) > lr.maxChunkSize {
		log.Infof("oversize message dropped: size=%v, limit=%v", len(encoded), lr.maxChunkSize)
		return 0, errors.New("message too big")
	}

	if !lr.syncWrite && lr.chunkSize+len(encoded) > lr.maxChunkSize {
		lr.flush()
	}

	if t == EventWT {
		lr.msg.Data.Events = append(lr.msg.Data.Events, encoded)
	} else if t == MetricWT {
		lr.msg.Data.Metrics = append(lr.msg.Data.Metrics, encoded)
	} else {
		return 0, fmt.Errorf("invalid write type: %v", t)
	}

	lr.chunkSize += len(encoded)
	if lr.syncWrite {
		lr.flush()
	}

	return len(bytes), nil
}

func (lr *logWriter) Flush() error {
	lr.mu.Lock()
	defer lr.mu.Unlock()
	return lr.flush()
}

func (lr *logWriter) flush() error {
	if lr.chunkSize == 0 {
		return errors.New("nothing to flush")
	}

	data, err := json.Marshal(lr.msg)
	if err != nil {
		return fmt.Errorf("error marshaling message: %w", err)
	}
	lr.msg.Data.Events = []string{}
	lr.msg.Data.Metrics = []string{}
	lr.chunkSize = 0

	data = append(data, "\n"...)

	if _, err := lr.dest.Write(data); err != nil {
		return fmt.Errorf("write to log reporter failed: %w", err)
	}

	if file, ok := lr.dest.(*os.File); ok {
		return file.Sync()
	}

	return nil
}
