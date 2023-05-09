#!/bin/bash

# © 2023 SolarWinds Worldwide, LLC. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

COVERPKG="github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/reporter,github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/log,github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/config,github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/host,github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm,github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/opentracing"
export SW_APM_DEBUG_LEVEL=1
go test -v -race -covermode=atomic -coverprofile=cov.out -coverpkg $COVERPKG

pushd internal/reporter/
go test -v -race -covermode=atomic -coverprofile=cov.out
popd

pushd internal/log/
go test -v -race -covermode=atomic -coverprofile=cov.out
popd

pushd internal/config/
go test -v -race -covermode=atomic -coverprofile=cov.out
popd

pushd internal/host/
go test -v -race -covermode=atomic -coverprofile=cov.out
popd

pushd opentracing
go test -v -race -covermode=atomic -coverprofile=cov.out
popd

gocovmerge cov.out internal/reporter/cov.out internal/log/cov.out internal/config/cov.out internal/host/cov.out opentracing/cov.out > covmerge.out

#go tool cover -html=covmerge.out
