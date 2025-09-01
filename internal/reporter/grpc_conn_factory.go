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

package reporter

import (
	"os"

	"github.com/solarwinds/apm-go/internal/config"
	"github.com/solarwinds/apm-go/internal/log"
)

func CreateGrpcConnection() (*grpcConnection, error) {
	// collector address override
	addr := config.GetCollector()

	var opts []GrpcConnOpt
	// certificate override
	if certPath := config.GetTrustedPath(); certPath != "" {
		var err error
		cert, err := os.ReadFile(certPath)
		if err != nil {
			log.Errorf("Error reading cert file %s: %v", certPath, err)
			return nil, err
		}
		opts = append(opts, WithCert(string(cert)))
	}

	opts = append(opts, WithMaxReqBytes(config.ReporterOpts().GetMaxReqBytes()))

	if proxy := getProxy(); proxy != "" {
		opts = append(opts, WithProxy(proxy))
		opts = append(opts, WithProxyCertPath(getProxyCertPath()))
	}

	// create connection object for events client and metrics client
	grpcConn, err1 := newGrpcConnection("SolarWinds Observability gRPC channel", addr, opts...)
	if err1 != nil {
		log.Errorf("Failed to initialize gRPC reporter %v: %v", addr, err1)
		return nil, err1
	}
	return grpcConn, nil
}
