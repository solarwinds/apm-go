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

package uams

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestReadFromFileNoExists(t *testing.T) {
	testfile := "/tmp/foobarbaz"
	require.NoFileExists(t, testfile)

	uid, err := ReadFromFile(testfile)
	require.Equal(t, uuid.Nil, uid)
	require.Error(t, err)
	require.Equal(t, "could not stat uams client file: stat /tmp/foobarbaz: no such file or directory", err.Error())
}

func TestReadFromFileDirectory(t *testing.T) {
	testfile := "mytemporarydirectory" // A directory!
	require.NoDirExists(t, testfile)
	require.NoError(t, os.Mkdir(testfile, os.ModeDir))
	defer func() {
		if err := os.Remove(testfile); err != nil {
			fmt.Println("Failed to remove test directory", err)
		}
	}()
	require.DirExists(t, testfile)

	uid, err := ReadFromFile(testfile)
	require.Equal(t, uuid.Nil, uid)
	require.Error(t, err)
	require.Equal(t, "could not open path (mytemporarydirectory); Expected a file, got a directory instead", err.Error())
}

func TestReadFromFileEmptyFile(t *testing.T) {
	testfile := "uams_test_file_empty"
	require.NoFileExists(t, testfile)
	require.NoError(t, os.WriteFile(testfile, []byte{}, 0600))
	defer func() {
		if err := os.Remove(testfile); err != nil {
			fmt.Println("Failed to remove test file", err)
		}
	}()

	uid, err := ReadFromFile(testfile)
	require.Equal(t, uuid.Nil, uid)
	require.Error(t, err)
	require.Equal(t, "uams client file (uams_test_file_empty) did not contain parseable UUID: invalid UUID length: 0", err.Error())
}

func TestReadFromFileInvalidLength(t *testing.T) {
	testfile := "uams_test_file_invalid_format"
	require.NoFileExists(t, testfile)
	require.NoError(t, os.WriteFile(testfile, []byte("Now is the winter of our discontent\nMade glorious summer by this sun of York"), 0600))
	defer func() {
		if err := os.Remove(testfile); err != nil {
			fmt.Println("Failed to remove test file", err)
		}
	}()

	uid, err := ReadFromFile(testfile)
	require.Equal(t, uuid.Nil, uid)
	require.Error(t, err)
	require.Equal(t, "uams client file (uams_test_file_invalid_format) did not contain parseable UUID: invalid UUID length: 76", err.Error())
}

func TestReadFromFileInvalidFormat(t *testing.T) {
	testfile := "uams_test_file_invalid_format"
	require.NoFileExists(t, testfile)
	// This string is the same length as a hex UUID, 36 characters
	require.NoError(t, os.WriteFile(testfile, []byte("Now is the winter of our discontent\n"), 0600))
	defer func() {
		if err := os.Remove(testfile); err != nil {
			fmt.Println("Failed to remove test file", err)
		}
	}()

	uid, err := ReadFromFile(testfile)
	require.Equal(t, uuid.Nil, uid)
	require.Error(t, err)
	require.Equal(t, "uams client file (uams_test_file_invalid_format) did not contain parseable UUID: invalid UUID format", err.Error())
}

func TestReadFromFileValidFormat(t *testing.T) {
	testfile := "uams_test_file_valid_format"
	expected, err := uuid.NewRandom()
	require.NoError(t, err)
	require.NoFileExists(t, testfile)
	require.NoError(t, os.WriteFile(testfile, []byte(expected.String()), 0600))
	defer func() {
		if err := os.Remove(testfile); err != nil {
			fmt.Println("Failed to remove test file", err)
		}
	}()

	uid, err := ReadFromFile(testfile)
	require.Equal(t, expected, uid)
	require.NoError(t, err)
}
