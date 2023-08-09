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
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/solarwindscloud/solarwinds-apm-go/internal/log"
	"io"
	"net/http"
)

const clientUrl = "http://127.0.0.1:2113/info/uamsclient"

func ReadFromHttp(url string) (uuid.UUID, error) {
	resp, err := http.Get(url)
	if err != nil {
		return uuid.Nil, err
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Warning("uamsclient: failed to close body", err)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return uuid.Nil, fmt.Errorf("uamsclient: expected 200 status code, got %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return uuid.Nil, err
	}
	result := make(map[string]interface{})
	if err = json.Unmarshal(body, &result); err != nil {
		return uuid.Nil, err
	}
	u, ok := result["uamsclient_id"]
	if !ok {
		return uuid.Nil, errors.New("uamsclient_id not found")
	}
	uidStr, ok := u.(string)
	if !ok {
		return uuid.Nil, fmt.Errorf("expected string, got %T instead", u)
	}
	if uid, err := uuid.Parse(uidStr); err != nil {
		return uuid.Nil, err
	} else {
		return uid, err
	}
}
