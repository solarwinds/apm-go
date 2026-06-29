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
	"testing"
	"time"

	"github.com/solarwinds/apm-go/internal/config"
	"github.com/solarwinds/apm-go/internal/oboe"
	"github.com/solarwinds/apm-go/internal/oboetestutils"
	"github.com/stretchr/testify/assert"
)

// withGetEnabled temporarily overrides config.GetEnabled for the duration of
// the test and restores it via t.Cleanup.
func withGetEnabled(t *testing.T, val bool) {
	t.Helper()
	orig := config.GetEnabled
	config.GetEnabled = func() bool { return val }
	t.Cleanup(func() { config.GetEnabled = orig })
}

// withGlobalOboe sets the global oboe instance for the duration of the test
// and clears it via t.Cleanup.
func withGlobalOboe(t *testing.T, o oboe.Oboe) {
	t.Helper()
	setGlobalOboe(o)
	t.Cleanup(func() { setGlobalOboe(nil) })
}

// TestWaitForReady_agentDisabled verifies that WaitForReady returns true
// immediately when the agent is disabled, without consulting the oboe state.
func TestWaitForReady_agentDisabled(t *testing.T) {
	withGetEnabled(t, false)
	// global oboe is nil — would return false if the disabled check were absent
	assert.True(t, WaitForReady(context.Background()))
}

// TestWaitForReady_nilOboe verifies that WaitForReady returns false when the
// agent is enabled but no oboe instance has been registered (e.g. Start() was
// never called).
func TestWaitForReady_nilOboe(t *testing.T) {
	withGetEnabled(t, true)
	withGlobalOboe(t, nil)
	assert.False(t, WaitForReady(context.Background()))
}

// TestWaitForReady_settingsAlreadyReady verifies that WaitForReady returns true
// immediately when the oboe instance already has default settings loaded.
func TestWaitForReady_settingsAlreadyReady(t *testing.T) {
	withGetEnabled(t, true)
	o := oboe.NewOboe()
	o.UpdateSetting(oboetestutils.GetDefaultSettingForTest())
	withGlobalOboe(t, o)

	assert.True(t, WaitForReady(context.Background()))
}

// TestWaitForReady_contextTimeout verifies that WaitForReady returns false when
// the context deadline is exceeded before settings become available.
func TestWaitForReady_contextTimeout(t *testing.T) {
	withGetEnabled(t, true)
	// Oboe with no settings — HasDefaultSetting() will return false.
	withGlobalOboe(t, oboe.NewOboe())

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	assert.False(t, WaitForReady(ctx))
}
