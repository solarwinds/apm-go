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

package utils

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"math"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/solarwinds/apm-go/internal/constants"
	"gopkg.in/mgo.v2/bson"
)

// SPrintBson prints the BSON message. It's not concurrent-safe and is for testing only
func SPrintBson(message []byte) string {
	m := make(map[string]interface{})
	if err := bson.Unmarshal(message, m); err != nil {
		// Since this is only used in testing/debug, we'll just return the error message
		return err.Error()
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	return string(b)
}

// GetLineByKeyword reads a file, searches for the keyword and returns the matched line.
// It returns empty string "" if no match found or failed to open the path.
// Pass an empty string "" if you just need to get the first line.
func GetLineByKeyword(path string, keyword string) string {
	if path == "" {
		return ""
	}
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if line := scanner.Text(); strings.Contains(line, keyword) {
			return line
		}
	}
	// ignore any scanner.Err(), just return an empty string.
	return ""
}

// GetStrByKeyword read a file, searches for the keyword and returns the matched line
// with trailing line-feed character trimmed.
func GetStrByKeyword(path string, keyword string) string {
	return strings.Trim(GetLineByKeyword(path, keyword), "\n")
}

// GetStrByKeywordFiles does the same thing as GetStrByKeyword but searches for a list
// of files and returns the first matched files and line
func GetStrByKeywordFiles(pathes []string, keyword string) (path string, line string) {
	for _, path = range pathes {
		line = GetStrByKeyword(path, keyword)
		if line != "" {
			return path, line
		}
	}
	return "", ""
}

// Min returns the lower value
func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

// Max returns the greater value
func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

// Byte2String converts a byte array into a string
func Byte2String(bs []int8) string {
	b := make([]byte, len(bs))
	for i, v := range bs {
		b[i] = byte(v)
	}
	return string(b)
}

// CopyMap makes a copy of all elements of a map.
func CopyMap(from *map[string]string) map[string]string {
	to := make(map[string]string)
	for k, v := range *from {
		to[k] = v
	}

	return to
}

// IsHigherOrEqualGoVersion checks if go version is higher or equal to the given version
func IsHigherOrEqualGoVersion(version string) bool {
	goVersion := strings.Split(runtime.Version(), ".")
	compVersion := strings.Split(version, ".")
	for i := 0; i < len(goVersion) && i < len(compVersion); i++ {
		l := len(compVersion[i])
		if len(goVersion[i]) > l {
			l = len(goVersion[i])
		}
		compVersion[i] = strings.Repeat("0", l-len(compVersion[i])) + compVersion[i]
		goVersion[i] = strings.Repeat("0", l-len(goVersion[i])) + goVersion[i]
		if strings.Compare(goVersion[i], compVersion[i]) == 1 {
			return true
		} else if strings.Compare(goVersion[i], compVersion[i]) == -1 {
			return false
		}
	}
	return true
}

func ArgsToMap(capacity, ratePerSec, tRCap, tRRate, tSCap, tSRate float64,
	metricsFlushInterval, maxTransactions int, token []byte) map[string][]byte {
	args := make(map[string][]byte)

	if capacity > -1 {
		bits := math.Float64bits(capacity)
		bytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(bytes, bits)
		args[constants.KvBucketCapacity] = bytes
	}
	if ratePerSec > -1 {
		bits := math.Float64bits(ratePerSec)
		bytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(bytes, bits)
		args[constants.KvBucketRate] = bytes
	}
	if tRCap > -1 {
		bits := math.Float64bits(tRCap)
		bytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(bytes, bits)
		args[constants.KvTriggerTraceRelaxedBucketCapacity] = bytes
	}
	if tRRate > -1 {
		bits := math.Float64bits(tRRate)
		bytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(bytes, bits)
		args[constants.KvTriggerTraceRelaxedBucketRate] = bytes
	}
	if tSCap > -1 {
		bits := math.Float64bits(tSCap)
		bytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(bytes, bits)
		args[constants.KvTriggerTraceStrictBucketCapacity] = bytes
	}
	if tSRate > -1 {
		bits := math.Float64bits(tSRate)
		bytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(bytes, bits)
		args[constants.KvTriggerTraceStrictBucketRate] = bytes
	}
	if metricsFlushInterval > -1 {
		bytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(bytes, uint32(metricsFlushInterval))
		args[constants.KvMetricsFlushInterval] = bytes
	}
	if maxTransactions > -1 {
		bytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(bytes, uint32(maxTransactions))
		args[constants.KvMaxTransactions] = bytes
	}

	args[constants.KvSignatureKey] = token

	return args
}

// SafeBuffer is goroutine-safe buffer. It is for internal test use only.
type SafeBuffer struct {
	buf bytes.Buffer
	sync.Mutex
}

func (b *SafeBuffer) Read(p []byte) (int, error) {
	b.Lock()
	defer b.Unlock()
	return b.buf.Read(p)
}

func (b *SafeBuffer) Write(p []byte) (int, error) {
	b.Lock()
	defer b.Unlock()
	return b.buf.Write(p)
}

func (b *SafeBuffer) String() string {
	b.Lock()
	defer b.Unlock()
	return b.buf.String()
}

// Reset truncates the buffer
func (b *SafeBuffer) Reset() {
	b.Lock()
	defer b.Unlock()
	b.buf.Reset()
}

func (b *SafeBuffer) Len() int {
	b.Lock()
	defer b.Unlock()
	return b.buf.Len()
}
