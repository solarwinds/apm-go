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

check_file() {
    head -n 5 $1 | grep -q "Licensed under the Apache License"
    if [ $? -ne 0 ];
    then
        echo "No license header found: $1"
    fi
}

export -f check_file

TMPFILE="$(mktemp)"

find . -type f \
    -not -path '*/\.git*' \
    -not -path '*/go.sum' \
    -not -path '*/go.mod' \
    -not -path './LICENSE' \
    -not -path '*/README.md' \
    -not -path './\.editorconfig' \
    -not -path './\.codecov.yaml' \
    -not -path './\.idea/*' \
    -exec bash -c 'check_file "$0"' {} \; | tee $TMPFILE

EXITCODE=0
if [ -s $TMPFILE ]; then
  EXITCODE=127
fi

rm $TMPFILE
exit $EXITCODE