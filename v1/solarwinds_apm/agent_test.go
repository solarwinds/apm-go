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
	"os"
	"strings"
	"testing"
	"time"

	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/log"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestSetGetLogLevel(t *testing.T) {
	oldLevel := GetLogLevel()

	err := SetLogLevel("INVALID")
	assert.Equal(t, err, errInvalidLogLevel)

	nl := "ERROR"
	err = SetLogLevel(nl)
	assert.Nil(t, err)

	newLevel := GetLogLevel()
	assert.Equal(t, newLevel, nl)

	SetLogLevel(oldLevel)
}

func TestShutdown(t *testing.T) {
	Shutdown(context.Background())
	assert.True(t, Closed())
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour*24)
	defer cancel()
	assert.False(t, WaitForReady(ctx))
}

func TestSetLogOutput(t *testing.T) {
	oldLevel := GetLogLevel()
	_ = SetLogLevel("DEBUG")
	defer SetLogLevel(oldLevel)

	var buf utils.SafeBuffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()

	log.Info("hello world")
	assert.True(t, strings.Contains(buf.String(), "hello world"))
}
