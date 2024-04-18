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

go 1.21

// TODO don't use the local repo
replace github.com/solarwinds/apm-go => ../..

require (
	github.com/XSAM/otelsql v0.26.0
	github.com/mattn/go-sqlite3 v1.14.18
	github.com/solarwinds/apm-go v0.1.1
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.46.0
	go.opentelemetry.io/otel v1.20.0
	go.opentelemetry.io/otel/trace v1.20.0
)

require (
	github.com/aws/aws-sdk-go-v2 v1.22.2 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.14.3 // indirect
	github.com/aws/smithy-go v1.16.0 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/coocood/freecache v1.2.4 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.3.0 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/uuid v1.4.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/solarwinds/apm-proto v0.0.0-20231107001908-432e697887b6 // indirect
	go.opentelemetry.io/otel/metric v1.20.0 // indirect
	go.opentelemetry.io/otel/sdk v1.20.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/net v0.18.0 // indirect
	golang.org/x/sys v0.14.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231106174013-bbf56f31fb17 // indirect
	google.golang.org/grpc v1.59.0 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
