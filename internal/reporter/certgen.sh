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

CONFIG<<EOF
[req]
distinguished_name=req;
[san]
subjectAltName=DNS:localhost
EOF
openssl req -x509 -newkey rsa:4096 -sha256 -days 365 -nodes \
  -keyout for_test.key -out for_test.crt -extensions san -config "$CONFIG" \
  -subj "/CN=localhost"
