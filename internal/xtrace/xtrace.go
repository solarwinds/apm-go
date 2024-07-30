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
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"github.com/solarwinds/apm-go/internal/log"
	"github.com/solarwinds/apm-go/internal/oboe"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	OptionsHeaderName    = "x-trace-options"
	OptionsSigHeaderName = "x-trace-options-signature"
)

type CtxKey int

const (
	OptionsKey CtxKey = iota
	SignatureKey
)

type SignatureState int

const (
	NoSignature SignatureState = iota
	ValidSignature
	InvalidSignature
)

var optRegex = regexp.MustCompile(";+")
var customKeyRegex = regexp.MustCompile(`^custom-[^\s]*$`)

func GetXTraceOptions(ctx context.Context, o oboe.Oboe) Options {
	xtoStr, ok := ctx.Value(OptionsKey).(string)
	if !ok {
		xtoStr = ""
	}
	xtoSig, ok := ctx.Value(SignatureKey).(string)
	if !ok {
		xtoSig = ""
	}

	return parseXTraceOptions(o, xtoStr, xtoSig)
}

func parseXTraceOptions(o oboe.Oboe, opts string, sig string) Options {
	x := Options{
		opts:        opts,
		sig:         sig,
		customKVs:   make(map[string]string),
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
	if sig == "" {
		x.sigState = NoSignature
	} else {
		x.authStatus = validateXTraceOptionsSignature(o, sig, strconv.FormatInt(x.timestamp, 10), opts)
		if x.authStatus.IsError() {
			log.Warning("Invalid xtrace options signature", x.authStatus.Msg())
			x.sigState = InvalidSignature
		} else {
			x.sigState = ValidSignature
		}
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
	sigState    SignatureState
	authStatus  AuthStatus
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

func (x Options) SignatureState() SignatureState {
	return x.sigState
}

func (x Options) Opts() string {
	return x.opts
}

func (x Options) IncludeResponse() bool {
	return x.opts != ""
}

func (x Options) SigAuthMsg() string {
	return x.authStatus.Msg()
}

func validateXTraceOptionsSignature(o oboe.Oboe, signature, ts, data string) AuthStatus {
	var err error
	_, err = tsInScope(ts)
	if err != nil {
		return AuthBadTimestamp
	}

	token, err := o.GetTriggerTraceToken()
	if err != nil {
		return AuthNoSignatureKey
	}

	if hmacHash(token, []byte(data)) != signature {
		return AuthBadSignature
	}
	return AuthOK
}

func HmacHashTT(o oboe.Oboe, data []byte) (string, error) {
	token, err := o.GetTriggerTraceToken()
	if err != nil {
		return "", err
	}
	return hmacHash(token, data), nil
}

func hmacHash(token, data []byte) string {
	h := hmac.New(sha1.New, token)
	h.Write(data)
	sha := hex.EncodeToString(h.Sum(nil))
	return sha
}

func tsInScope(tsStr string) (string, error) {
	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return "", fmt.Errorf("tsInScope: %w", err)
	}

	t := time.Unix(ts, 0)
	if t.Before(time.Now().Add(time.Minute*-5)) ||
		t.After(time.Now().Add(time.Minute*5)) {
		return "", fmt.Errorf("timestamp out of scope: %s", tsStr)
	}
	return strconv.FormatInt(ts, 10), nil
}
