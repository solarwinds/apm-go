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
SHELL=bash

prev_branch :=$(shell git rev-parse --abbrev-ref HEAD)

coverpkg="github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/reporter,github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/log,github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/bson,github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/metrics,github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm,github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/config,github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/host,github.com/solarwindscloud/solarwinds-apm-go/v1/contrib/multitracer"
cov_args=-covermode=atomic -coverpkg=$(coverpkg)
cov_out=-coverprofile=cov.out
cov_files=cov.out internal/reporter/cov.out
cov_merge=covmerge.out
test_args=-race

test: export SW_APM_DEBUG_LEVEL=1
#test: export SW_APM_DEBUG_LEVEL=0
test:
	go test $(test_args) $(cov_args) -v $(cov_out)
	cd internal/reporter && go test $(test_args) $(cov_args) $(cov_out) -v

$(cov_merge): test $(cov_files)
	gocovmerge $(cov_files) > $(cov_merge)

coverhtml: $(cov_merge)
	go tool cover -html=$(cov_merge)
