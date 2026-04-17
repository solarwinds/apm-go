// © 2024 SolarWinds Worldwide, LLC. All rights reserved.
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

//go:build !windows

package oboe

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileBasedWatcher(t *testing.T) {
	o := NewOboe()
	w := NewFileBasedWatcher(o)
	assert.NotNil(t, w)
}

// ensureNoSettingsFile removes the settings file if it exists so each test
// starts from a clean state regardless of previous test failures.
func ensureNoSettingsFile(t *testing.T) {
	t.Helper()
	err := os.Remove(settingsFileName)
	if err == nil || os.IsNotExist(err) {
		return
	}
	t.Fatalf("unexpected error removing settings file: %s", err)
}

func writeSettingsFile(t *testing.T, content []byte) {
	t.Helper()
	require.NoError(t, os.WriteFile(settingsFileName, content, 0644))
	t.Cleanup(func() { _ = os.Remove(settingsFileName) })
}

var validSettingsContent = []byte(`[{"arguments":{"BucketCapacity":1,"BucketRate":1,"MetricsFlushInterval":1,"TriggerRelaxedBucketCapacity":1,"TriggerRelaxedBucketRate":1,"TriggerStrictBucketCapacity":1,"TriggerStrictBucketRate":1},"flags":"SAMPLE_START","layer":"","timestamp":1715900164,"ttl":120,"type":0,"value":1000000}]`)

func TestFileBasedWatcherReadSettingFromFile(t *testing.T) {
	t.Run("without file", func(t *testing.T) {
		ensureNoSettingsFile(t)
		o := NewOboe()
		w := NewFileBasedWatcher(o).(*fileBasedWatcher)
		w.readSettingFromFile()
		// File is absent: no settings should be loaded
		assert.False(t, o.HasDefaultSetting())
		assert.Nil(t, o.GetSetting())
	})

	t.Run("with file", func(t *testing.T) {
		ensureNoSettingsFile(t)
		writeSettingsFile(t, validSettingsContent)
		o := NewOboe()
		w := NewFileBasedWatcher(o).(*fileBasedWatcher)
		w.readSettingFromFile()
		// Settings should be loaded and the values must match the JSON content
		require.True(t, o.HasDefaultSetting())
		s := o.GetSetting()
		require.NotNil(t, s)
		assert.Equal(t, int64(1000000), int64(s.value))
		assert.Equal(t, 120*time.Second, s.ttl)
		assert.Equal(t, float64(1), s.bucket.capacity)
		assert.Equal(t, float64(1), s.triggerTraceRelaxedBucket.capacity)
		assert.Equal(t, float64(1), s.triggerTraceStrictBucket.capacity)
	})
}

// pollUntil retries cond every 5ms until it returns true or deadline elapses.
// Returns true if the condition was met, false if it timed out.
func pollUntil(deadline time.Duration, cond func() bool) bool {
	pollTicker := time.NewTicker(5 * time.Millisecond)
	defer pollTicker.Stop()
	timer := time.NewTimer(deadline)
	defer timer.Stop()
	for {
		if cond() {
			return true
		}
		select {
		case <-timer.C:
			return cond() // one final check after deadline
		case <-pollTicker.C:
		}
	}
}

func TestFileBasedWatcherStartStop(t *testing.T) {
	t.Run("without file", func(t *testing.T) {
		ensureNoSettingsFile(t)
		// Set timeout to 0 so waitForSettingsFile returns immediately
		t.Setenv(timeoutEnv, "0s")
		o := NewOboe()
		w := NewFileBasedWatcher(o)
		// Start launches the background goroutine and does an immediate read
		w.Start()
		// Stop sends a signal into the exit channel, causing the goroutine to return
		w.Stop()
		// No file present: goroutine started and stopped cleanly, settings remain unset.
		// readSettingFromFile is called synchronously inside Start, so this is already set.
		assert.False(t, o.HasDefaultSetting())
		// Wait until the goroutine has consumed the stop signal (channel becomes empty).
		assert.True(t, pollUntil(2*time.Second, func() bool { return len(exit) == 0 }),
			"goroutine did not consume stop signal within deadline")
	})

	t.Run("with file", func(t *testing.T) {
		ensureNoSettingsFile(t)
		writeSettingsFile(t, validSettingsContent)
		t.Setenv(timeoutEnv, "0s")
		o := NewOboe()
		w := NewFileBasedWatcher(o)
		// Start calls readSettingFromFile immediately (before the ticker fires)
		w.Start()
		w.Stop()
		// Settings must be applied by the immediate synchronous read inside Start
		require.True(t, o.HasDefaultSetting())
		assert.Equal(t, int64(1000000), int64(o.GetSetting().value))
		// Wait until the goroutine has consumed the stop signal (channel becomes empty).
		assert.True(t, pollUntil(2*time.Second, func() bool { return len(exit) == 0 }),
			"goroutine did not consume stop signal within deadline")
	})
}

func TestWaitForSettingsFile(t *testing.T) {
	t.Run("without file zero timeout", func(t *testing.T) {
		ensureNoSettingsFile(t)
		t.Setenv(timeoutEnv, "0s")
		// Zero timeout means skip the wait entirely; must return in well under 1s
		start := time.Now()
		waitForSettingsFile()
		assert.Less(t, time.Since(start), 500*time.Millisecond)
		// File must still not exist (function only waits, never creates)
		_, err := os.Stat(settingsFileName)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("with existing file", func(t *testing.T) {
		ensureNoSettingsFile(t)
		writeSettingsFile(t, []byte(`[{}]`))
		t.Setenv(timeoutEnv, "500ms")
		// File already exists: should return almost immediately, well before the 500ms timeout
		start := time.Now()
		waitForSettingsFile()
		assert.Less(t, time.Since(start), 400*time.Millisecond)
		// File must still exist
		_, err := os.Stat(settingsFileName)
		assert.NoError(t, err)
	})

	t.Run("without file times out", func(t *testing.T) {
		ensureNoSettingsFile(t)
		t.Setenv(timeoutEnv, "50ms")
		// No file present: must block for the full timeout duration then return
		start := time.Now()
		waitForSettingsFile()
		elapsed := time.Since(start)
		assert.GreaterOrEqual(t, elapsed, 50*time.Millisecond)
		// File must still not exist
		_, err := os.Stat(settingsFileName)
		assert.True(t, os.IsNotExist(err))
	})
}
