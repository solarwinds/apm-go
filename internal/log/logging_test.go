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
package log

import (
	"bytes"
	"errors"
	"github.com/solarwinds/apm-go/internal/utils"
	"io"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"sync"

	"github.com/stretchr/testify/assert"
)

func TestDebugLevel(t *testing.T) {
	tests := []struct {
		key      string
		val      string
		expected LogLevel
	}{
		{"SW_APM_DEBUG_LEVEL", "DEBUG", DEBUG},
		{"SW_APM_DEBUG_LEVEL", "Info", INFO},
		{"SW_APM_DEBUG_LEVEL", "warn", WARNING},
		{"SW_APM_DEBUG_LEVEL", "erroR", ERROR},
		{"SW_APM_DEBUG_LEVEL", "erroR  ", ERROR},
		{"SW_APM_DEBUG_LEVEL", "HelloWorld", DefaultLevel},
		{"SW_APM_DEBUG_LEVEL", "0", DEBUG},
		{"SW_APM_DEBUG_LEVEL", "1", INFO},
		{"SW_APM_DEBUG_LEVEL", "2", WARNING},
		{"SW_APM_DEBUG_LEVEL", "3", ERROR},
		{"SW_APM_DEBUG_LEVEL", "4", DefaultLevel},
		{"SW_APM_DEBUG_LEVEL", "1000", DefaultLevel},
	}

	for _, test := range tests {
		os.Setenv(test.key, test.val)
		SetLevelFromStr(os.Getenv(envSolarWindsAPMLogLevel))
		assert.EqualValues(t, test.expected, Level(), "Test-"+test.val)
	}

	os.Unsetenv("SW_APM_DEBUG_LEVEL")
	SetLevelFromStr(os.Getenv(envSolarWindsAPMLogLevel))
	assert.EqualValues(t, Level(), DefaultLevel)
}

func TestLog(t *testing.T) {
	var buffer bytes.Buffer
	SetOutput(&buffer)

	os.Setenv("SW_APM_DEBUG_LEVEL", "debug")
	SetLevelFromStr(os.Getenv(envSolarWindsAPMLogLevel))

	tests := map[string]string{
		"hello world": "hello world\n",
		"":            "\n",
		"hello %s":    "hello %!s(MISSING)\n",
	}

	for str, expected := range tests {
		buffer.Reset()
		Logf(INFO, str)
		assert.True(t, strings.HasSuffix(buffer.String(), expected))
	}

	buffer.Reset()
	Log(INFO, 1, 2, 3)
	assert.True(t, strings.HasSuffix(buffer.String(), "1 2 3\n"))

	buffer.Reset()
	Debug(1, "abc", 3)
	assert.True(t, strings.HasSuffix(buffer.String(), "1abc3\n"))

	buffer.Reset()
	Error(errors.New("hello"))
	assert.True(t, strings.HasSuffix(buffer.String(), "hello\n"))

	buffer.Reset()
	Warning("Áú")
	assert.True(t, strings.HasSuffix(buffer.String(), "Áú\n"))

	buffer.Reset()
	Info("hello")
	assert.True(t, strings.HasSuffix(buffer.String(), "\n"))

	buffer.Reset()
	Warningf("hello %s", "world")
	assert.True(t, strings.HasSuffix(buffer.String(), "hello world\n"))

	buffer.Reset()
	Infof("show me the %v", "code")
	assert.True(t, strings.HasSuffix(buffer.String(), "show me the code\n"))

	SetOutput(os.Stderr)
	os.Unsetenv("SW_APM_DEBUG_LEVEL")

}

func TestStrToLevel(t *testing.T) {
	tests := map[string]LogLevel{
		"DEBUG": DEBUG,
		"INFO":  INFO,
		"WARN":  WARNING,
		"ERROR": ERROR,
	}
	for str, lvl := range tests {
		l, _ := StrToLevel(str)
		assert.Equal(t, lvl, l)
	}
}

func TestVerifyLogLevel(t *testing.T) {
	tests := map[string]LogLevel{
		"DEBUG":   DEBUG,
		"Debug":   DEBUG,
		"debug":   DEBUG,
		" dEbUg ": DEBUG,
		"INFO":    INFO,
		"WARN":    WARNING,
		"ERROR":   ERROR,
		"ABC":     DefaultLevel,
	}
	for str, lvl := range tests {
		l, _ := ToLogLevel(str)
		assert.Equal(t, lvl, l)
	}
}

func TestSetLevel(t *testing.T) {
	var buf utils.SafeBuffer
	var writers []io.Writer

	writers = append(writers, &buf)
	writers = append(writers, os.Stderr)

	SetOutput(io.MultiWriter(writers...))

	defer func() {
		SetOutput(os.Stderr)
	}()

	SetLevel(INFO)
	var wg = &sync.WaitGroup{}
	wg.Add(100)
	for i := 0; i < 100; i++ {
		go func(wg *sync.WaitGroup) {
			time.Sleep(time.Millisecond * time.Duration(rand.Intn(5)))
			Debug("hello world")
			wg.Done()
		}(wg)
	}
	wg.Wait()
	assert.Equal(t, "", buf.String())

	buf.Reset()
	SetLevel(DEBUG)
	Debug("test")
	assert.True(t, strings.Contains(buf.String(), "test"))
	buf.Reset()
	Error("", "one", "two", "three")
	assert.Equal(t, DEBUG, Level())
	assert.True(t, strings.Contains(buf.String(), "onetwothree"))
}
