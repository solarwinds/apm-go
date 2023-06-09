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

package xtrace

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/log"
)

const (
	XTraceOptionsHeaderName    = "x-trace-options"
	XTraceOptionsSigHeaderName = "x-trace-options-signature"
)

var optRegex = regexp.MustCompile(";+")
var customKeyRegex = regexp.MustCompile(`^custom-[^\s]*$`)

func ParseXTraceOptions(opts string, sig string) Options {
	x := Options{
		opts:        opts,
		sig:         sig,
		swKeys:      "",
		customKVs:   make(map[string]string),
		timestamp:   0,
		ignoredKeys: make([]string, 0),
	}
	for _, opt := range optRegex.Split(opts, -1) {
		k, v, found := strings.Cut(opt, "=")
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if !found {
			// Only support trigger-trace without an equals sign
			if k == "trigger-trace" {
				x.tt = true
			} else {
				x.ignoredKeys = append(x.ignoredKeys, k)
			}
			continue
		}
		v = strings.TrimSpace(v)
		if k == "sw-keys" {
			x.swKeys = v
		} else if k == "ts" {
			ts, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				log.Debug("Invalid x-trace timestamp value", ts)
				x.ignoredKeys = append(x.ignoredKeys, k)
			} else {
				x.timestamp = ts
			}
		} else if k == "trigger-trace" {
			log.Debug("trigger-trace must be standalone flag, ignoring.")
		} else if customKeyRegex.MatchString(k) {
			x.customKVs[k] = v
		} else {
			x.ignoredKeys = append(x.ignoredKeys, k)
		}
	}
	if len(x.ignoredKeys) > 0 {
		log.Debugf("Some x-trace-options were ignored: %s", x.ignoredKeys)
	}
	return x
}

type Options struct {
	opts        string
	sig         string
	swKeys      string
	customKVs   map[string]string
	timestamp   int64
	tt          bool
	ignoredKeys []string
}

func (x Options) SwKeys() string {
	return x.swKeys
}

func (x Options) CustomKVs() map[string]string {
	return x.customKVs
}

func (x Options) Timestamp() int64 {
	return x.timestamp
}

func (x Options) TriggerTrace() bool {
	return x.tt
}

func (x Options) IgnoredKeys() []string {
	return x.ignoredKeys
}

func (x Options) Signature() string {
	return x.sig
}
