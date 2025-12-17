// © 2025 SolarWinds Worldwide, LLC. All rights reserved.
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

package oboe

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/solarwinds/apm-go/internal/config"
	"github.com/solarwinds/apm-go/internal/log"
)

const (
	defaultTimeout = 10 * time.Second
)

type settingsService struct {
	baseURL     string
	serviceName string
	hostName    string
	bearerToken string
	client      *http.Client
}

func newSettingsService(baseURL, serviceName, hostName, bearerToken string) *settingsService {
	if hostName == "" {
		hostName = "unknown"
	}
	return &settingsService{
		baseURL:     baseURL,
		serviceName: serviceName,
		hostName:    hostName,
		bearerToken: bearerToken,
		client: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

func (s *settingsService) getSettings() (*httpSettings, error) {
	requestUrl := s.buildURL()
	log.Debugf("Fetching settings from URL: %s", requestUrl)
	req, err := http.NewRequest(http.MethodGet, requestUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	s.setAuthHeaders(req)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Warningf("failed to close response body querying settings: %v", closeErr)
		}
	}()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, config.ErrInvalidServiceKey
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var settings httpSettings
	if err := json.Unmarshal(body, &settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &settings, nil
}

func (s *settingsService) buildURL() string {
	return fmt.Sprintf("%s/v1/settings/%s/%s",
		strings.TrimSuffix(s.baseURL, "/"),
		url.PathEscape(s.serviceName),
		url.PathEscape(s.hostName))
}

func (s *settingsService) setAuthHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+s.bearerToken)
	req.Header.Set("Accept", "application/json")
}
