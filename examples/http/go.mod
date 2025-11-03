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

module github.com/solarwinds/apm-go/examples/http

go 1.24.0

// TODO don't use the local repo
replace github.com/solarwinds/apm-go => ../..

require (
	github.com/XSAM/otelsql v0.26.0
	github.com/mattn/go-sqlite3 v1.14.18
	github.com/solarwinds/apm-go v1.2.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.63.0
	go.opentelemetry.io/otel v1.38.0
	go.opentelemetry.io/otel/trace v1.38.0
)

require (
	github.com/aws/aws-sdk-go v1.55.8 // indirect
	github.com/aws/aws-sdk-go-v2 v1.39.4 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.11 // indirect
	github.com/aws/smithy-go v1.23.1 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/coocood/freecache v1.2.4 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.3 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/solarwinds/apm-proto v1.0.8 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/detectors/aws/ec2 v1.38.0 // indirect
	go.opentelemetry.io/contrib/detectors/azure/azurevm v0.10.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.38.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.38.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.38.0 // indirect
	go.opentelemetry.io/otel/metric v1.38.0 // indirect
	go.opentelemetry.io/otel/sdk v1.38.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.38.0 // indirect
	go.opentelemetry.io/proto/otlp v1.8.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/net v0.46.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20251022142026-3a174f9686a8 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251022142026-3a174f9686a8 // indirect
	google.golang.org/grpc v1.76.0 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
