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

import "github.com/solarwinds/apm-go/internal/log"

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
