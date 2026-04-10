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
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/mgo.v2/bson"
)

func TestSPrintBson(t *testing.T) {
	data, err := bson.Marshal(bson.M{"key": "value"})
	require.NoError(t, err)

	result := SPrintBson(data)
	assert.Contains(t, result, "key")
	assert.Contains(t, result, "value")
}

func TestSPrintBsonInvalidBytes(t *testing.T) {
	// Malformed BSON should return the error string rather than panic
	result := SPrintBson([]byte{0x00, 0x01})
	assert.NotEmpty(t, result)
}

func TestGetLineByKeyword(t *testing.T) {
	f, err := os.CreateTemp("", "utils_test_*")
	require.NoError(t, err)
	defer func() { _ = os.Remove(f.Name()) }()

	_, err = f.WriteString("line one\nfoo:bar\nline three\n")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	assert.Equal(t, "foo:bar", GetLineByKeyword(f.Name(), "foo"))
	assert.Equal(t, "line one", GetLineByKeyword(f.Name(), "line one"))
	assert.Equal(t, "", GetLineByKeyword(f.Name(), "nonexistent"))
	assert.Equal(t, "", GetLineByKeyword("", "foo"))
	assert.Equal(t, "", GetLineByKeyword("/nonexistent/path/file.txt", "foo"))
}

func TestGetStrByKeyword(t *testing.T) {
	f, err := os.CreateTemp("", "utils_test_*")
	require.NoError(t, err)
	defer func() { _ = os.Remove(f.Name()) }()

	_, err = f.WriteString("foo:bar\n")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	assert.Equal(t, "foo:bar", GetStrByKeyword(f.Name(), "foo"))
	assert.Equal(t, "", GetStrByKeyword(f.Name(), "nonexistent"))
}

func TestGetStrByKeywordFiles(t *testing.T) {
	f1, err := os.CreateTemp("", "utils_test_*")
	require.NoError(t, err)
	defer func() { _ = os.Remove(f1.Name()) }()

	f2, err := os.CreateTemp("", "utils_test_*")
	require.NoError(t, err)
	defer func() { _ = os.Remove(f2.Name()) }()

	_, err = f1.WriteString("match line\n")
	require.NoError(t, err)
	require.NoError(t, f1.Close())
	require.NoError(t, f2.Close())

	// f1 contains the keyword; f2 is empty
	path, line := GetStrByKeywordFiles([]string{f1.Name(), f2.Name()}, "match")
	assert.Equal(t, f1.Name(), path)
	assert.Equal(t, "match line", line)

	// f2 is checked second; keyword is not in either
	path2, line2 := GetStrByKeywordFiles([]string{f1.Name(), f2.Name()}, "nonexistent")
	assert.Equal(t, "", path2)
	assert.Equal(t, "", line2)

	// Empty file list
	path3, line3 := GetStrByKeywordFiles([]string{}, "match")
	assert.Equal(t, "", path3)
	assert.Equal(t, "", line3)
}

func TestMin(t *testing.T) {
	assert.Equal(t, 1, Min(1, 2))
	assert.Equal(t, 1, Min(2, 1))
	assert.Equal(t, 5, Min(5, 5))
}

func TestMax(t *testing.T) {
	assert.Equal(t, 2, Max(1, 2))
	assert.Equal(t, 2, Max(2, 1))
	assert.Equal(t, 5, Max(5, 5))
}

func TestByte2String(t *testing.T) {
	bs := []int8{'h', 'e', 'l', 'l', 'o'}
	assert.Equal(t, "hello", Byte2String(bs))
	assert.Equal(t, "", Byte2String([]int8{}))
}

func TestCopyMap(t *testing.T) {
	original := map[string]string{"a": "1", "b": "2"}
	copied := CopyMap(&original)
	assert.Equal(t, original, copied)

	// Modifying the copy must not affect the original
	copied["c"] = "3"
	_, ok := original["c"]
	assert.False(t, ok)
}

func TestIsHigherOrEqualGoVersion(t *testing.T) {
	current := runtime.Version() // e.g. "go1.24.2"

	// Current version is equal to itself
	assert.True(t, IsHigherOrEqualGoVersion(current))

	// Current version is higher than a very old version
	assert.True(t, IsHigherOrEqualGoVersion("go1.0"))

	// Current version is lower than a far-future version
	assert.False(t, IsHigherOrEqualGoVersion("go9.99"))
}
