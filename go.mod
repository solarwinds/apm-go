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

module github.com/solarwinds/apm-go

go 1.21

require (
	github.com/aws/aws-sdk-go-v2 v1.26.1
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.1
	github.com/aws/smithy-go v1.20.2
	github.com/coocood/freecache v1.2.4
	github.com/google/uuid v1.6.0
	github.com/pkg/errors v0.9.1
	github.com/solarwinds/apm-proto v0.0.0-20231107001908-432e697887b6
	github.com/stretchr/testify v1.9.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.46.1
	go.uber.org/atomic v1.11.0
	google.golang.org/grpc v1.63.2
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/otel/metric v1.27.0 // indirect
	golang.org/x/net v0.24.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240415180920-8c6c420018be // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)

require (
	github.com/stretchr/objx v0.5.2 // indirect
	go.opentelemetry.io/otel v1.27.0
	go.opentelemetry.io/otel/sdk v1.25.0
	go.opentelemetry.io/otel/trace v1.27.0
	golang.org/x/sys v0.19.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
