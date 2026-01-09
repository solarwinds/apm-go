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
runtest:
	go test -race -timeout 3m -count=1 -short -covermode=atomic  ./... && echo "All tests passed."

runtestfast:
	go test -race -timeout 3m -short -covermode=atomic ./... && echo "All tests passed."

test: runtest
testfast: runtestfast

vet:
	@go vet -composites=false ./... && echo "Go vet analysis passed."

clean:
	@go clean -testcache

lint:
	golangci-lint run ./...

license-check:
	./license_check.sh

sure: clean test vet lint license-check

.PHONY: test vet clean lint license-check
