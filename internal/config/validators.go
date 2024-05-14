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
package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// InvalidEnv returns a string indicating invalid environment variables
func InvalidEnv(env string, val string) string {
	return fmt.Sprintf("invalid env, discarded - %s: \"%s\"", env, val)
}

// MissingEnv returns a string indicating missing environment variables
func MissingEnv(env string) string {
	return fmt.Sprintf("missing env - %s", env)
}

const (
	validServiceKeyPattern = `^([a-zA-Z0-9]{64}|[a-zA-Z0-9-_]{71}):.{1,255}$`

	serviceKeyPartsCnt  = 2
	serviceKeyDelimiter = ":"

	spacesPattern  = `\s`
	spacesReplacer = "-"

	invalidCharacters   = `[^a-z0-9.:_-]`
	invalidCharReplacer = ""
)

var (
	// IsValidServiceKey verifies if the service key is a valid one.
	// A valid service key is something like 'service_token:service_name'.
	// The service_token should be of 64 characters long and the size of
	// service_name is larger than 0 but up to 255 characters.
	IsValidServiceKey = regexp.MustCompile(validServiceKeyPattern).MatchString

	// ReplaceSpacesWith replaces all the spaces with valid characters (hyphen)
	ReplaceSpacesWith = regexp.MustCompile(spacesPattern).ReplaceAllString

	// RemoveInvalidChars remove invalid characters
	RemoveInvalidChars = regexp.MustCompile(invalidCharacters).ReplaceAllString
)

// ToServiceKey converts a string to a service key. The argument should be
// a valid service key string.
//
// It doesn't touch the service key but does the following to the original
// service name:
// - convert all characters to lowercase
// - convert spaces to hyphens
// - remove invalid characters ( [^a-z0-9.:_-])
func ToServiceKey(s string) string {
	parts := strings.SplitN(s, serviceKeyDelimiter, serviceKeyPartsCnt)
	if len(parts) != serviceKeyPartsCnt {
		// This should not happen as this method is called after service key
		// validation, which rejects a key without the delimiter. This check
		// is added here to avoid out-of-bound slice access later.
		return s
	}

	sToken, sName := parts[0], parts[1]

	sName = strings.ToLower(sName)
	sName = ReplaceSpacesWith(sName, spacesReplacer)
	sName = RemoveInvalidChars(sName, invalidCharReplacer)

	return strings.Join([]string{sToken, sName}, serviceKeyDelimiter)
}

// IsValidHost verifies if the host is in a valid format
func IsValidHost(host string) bool {
	// TODO
	return host != ""
}

// IsValidFile checks if the string represents a valid file.
func IsValidFile(file string) bool {
	// TODO
	return true
}

// IsValidEc2MetadataTimeout checks if the timeout is within the designated range
func IsValidEc2MetadataTimeout(t int) bool {
	return t >= 0 && t <= 3000
}

// IsValidTracingMode checks if the mode is valid
func IsValidTracingMode(m TracingMode) bool {
	return m == EnabledTracingMode || m == DisabledTracingMode
}

// IsValidSampleRate checks if the rate is valid
func IsValidSampleRate(rate int) bool {
	return rate >= MinSampleRate && rate <= MaxSampleRate
}

func IsValidTokenBucketRate(rate float64) bool {
	return rate >= 0 && rate <= maxTokenBucketRate
}

func IsValidTokenBucketCap(cap float64) bool {
	return cap >= 0 && cap <= maxTokenBucketCapacity
}

// NormalizeTracingMode converts an old-style tracing mode (always/never) to a
// new-style tracing mode (enabled/disabled).
func NormalizeTracingMode(m TracingMode) TracingMode {
	modeStr := strings.ToLower(strings.TrimSpace(string(m)))
	mode := m

	if modeStr == "always" {
		mode = EnabledTracingMode
	} else if modeStr == "never" {
		mode = DisabledTracingMode
	}

	return mode
}

// IsValidHostnameAlias checks if the alias is valid
func IsValidHostnameAlias(a string) bool {
	return true
}

// ToInteger converts a string to an integer
func ToInteger(i string) int {
	n, _ := strconv.Atoi(i)
	return n
}

// MaskServiceKey masks the middle part of the token and returns the
// masked service key. For example:
// key: "ae38315f6116585d64d82ec2455aa3ec61e02fee25d286f74ace9e4fea189217:go"
// masked:"ae38********************************************************9217:go"
func MaskServiceKey(validKey string) string {
	var sep = ":"
	var hLen, tLen = 4, 4
	var mask = "*"

	s := strings.Split(validKey, sep)
	tk := s[0]

	if len(tk) <= hLen+tLen {
		return validKey
	}

	tk = tk[0:4] + strings.Repeat(mask,
		utf8.RuneCountInString(tk)-hLen-tLen) + tk[len(tk)-4:]

	masked := tk + sep
	if len(s) >= 2 {
		masked += s[1]
	}
	return masked
}
