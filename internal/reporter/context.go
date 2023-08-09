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

package reporter

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"github.com/solarwindscloud/solarwinds-apm-go/internal/log"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

type AuthStatus int

const (
	AuthOK = iota
	AuthBadTimestamp
	AuthNoSignatureKey
	AuthBadSignature
)

func (a AuthStatus) IsError() bool {
	return a != AuthOK
}

func (a AuthStatus) Msg() string {
	switch a {
	case AuthOK:
		return "ok"
	case AuthBadTimestamp:
		return "bad-timestamp"
	case AuthNoSignatureKey:
		return "no-signature-key"
	case AuthBadSignature:
		return "bad-signature"
	}
	log.Debugf("could not read msg for unknown AuthStatus: %s", a)
	return ""
}

// TODO: This could live in the `xtrace` package, except it requires
// TODO: the ability to extract the TT Token from oboe settings.
// TODO: Determine a clean/elegant way to clean this up.
func ValidateXTraceOptionsSignature(signature, ts, data string) AuthStatus {
	var err error
	_, err = tsInScope(ts)
	if err != nil {
		return AuthBadTimestamp
	}

	token, err := getTriggerTraceToken()
	if err != nil {
		return AuthNoSignatureKey
	}

	if HmacHash(token, []byte(data)) != signature {
		return AuthBadSignature
	}
	return AuthOK
}

func HmacHashTT(data []byte) (string, error) {
	token, err := getTriggerTraceToken()
	if err != nil {
		return "", err
	}
	return HmacHash(token, data), nil
}

func HmacHash(token, data []byte) string {
	h := hmac.New(sha1.New, token)
	h.Write(data)
	sha := hex.EncodeToString(h.Sum(nil))
	return sha
}

func getTriggerTraceToken() ([]byte, error) {
	setting, ok := getSetting()
	if !ok {
		return nil, errors.New("failed to get settings")
	}
	if len(setting.triggerToken) == 0 {
		return nil, errors.New("no valid signature key found")
	}
	return setting.triggerToken, nil
}

func tsInScope(tsStr string) (string, error) {
	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return "", errors.Wrap(err, "tsInScope")
	}

	t := time.Unix(ts, 0)
	if t.Before(time.Now().Add(time.Minute*-5)) ||
		t.After(time.Now().Add(time.Minute*5)) {
		return "", fmt.Errorf("timestamp out of scope: %s", tsStr)
	}
	return strconv.FormatInt(ts, 10), nil
}
