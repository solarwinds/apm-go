// © 2025 SolarWinds Worldwide, LLC. All rights reserved.
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

package swo

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/solarwinds/apm-go/internal/config"
	"github.com/solarwinds/apm-go/internal/entryspans"
	"github.com/solarwinds/apm-go/internal/log"
	"go.opentelemetry.io/otel/trace"
)

var (
	errInvalidLogLevel = errors.New("invalid log level")
)

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

// WaitForReady checks if the agent is ready. It returns true if the agent is ready,
// or false if it is not. Default timeout is 10 seconds, but can be overridden
// by providing a context with a deadline.
// WaitForReady is only meaningful when the agent was initialized via Start().
func WaitForReady(ctx context.Context) bool {
	if !config.GetEnabled() {
		return true
	}
	o := getGlobalOboe()
	if o == nil {
		return false
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
	}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		if o.HasDefaultSetting() {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
		}
	}
}

// SetTransactionName sets the transaction name of the current entry span. If set multiple times, the last is used.
// Returns nil on success; Error if the provided name is blank, or we are unable to set the transaction name.
func SetTransactionName(ctx context.Context, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("invalid transaction name")
	}
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return errors.New("could not obtain OpenTelemetry SpanContext from given context")
	}
	return entryspans.SetTransactionName(sc.TraceID(), name)
}
