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
package solarwinds_apm

import (
	"context"
	"io"

	"github.com/pkg/errors"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/log"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/reporter"
)

var (
	errInvalidLogLevel = errors.New("invalid log level")
)

// WaitForReady checks if the agent is ready. It returns true is the agent is ready,
// or false if it is not.
//
// A call to this method will block until the agent is ready or the context is
// canceled, or the agent is already closed.
// The agent is considered ready if there is a valid default setting for sampling.
func WaitForReady(ctx context.Context) bool {
	if Closed() {
		return false
	}
	return reporter.WaitForReady(ctx)
}

func flushAgent() error {
	return reporter.Flush()
}

// Shutdown flush the metrics and stops the agent. The call will block until the agent
// flushes and is successfully shutdown or the context is canceled. It returns nil
// for successful shutdown and or error when the context is canceled or the agent
// has already been closed before.
//
// This function should be called only once.
func Shutdown(ctx context.Context) error {
	return reporter.Shutdown(ctx)
}

// Closed denotes if the agent is closed (by either calling Shutdown explicitly
// or being triggered from some internal error).
func Closed() bool {
	return reporter.Closed()
}

// SetLogLevel changes the logging level of the library
// Valid logging levels: DEBUG, INFO, WARN, ERROR
func SetLogLevel(level string) error {
	l, ok := log.ToLogLevel(level)
	if !ok {
		return errInvalidLogLevel
	}
	log.SetLevel(l)
	return nil
}

// GetLogLevel returns the current logging level of the library
func GetLogLevel() string {
	return log.LevelStr[log.Level()]
}

// SetLogOutput sets the output destination for the internal logger.
func SetLogOutput(w io.Writer) {
	log.SetOutput(w)
}

// SetServiceKey sets the service key of the agent
func SetServiceKey(key string) {
	reporter.SetServiceKey(key)
}