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

# build dot graph files for specified tests and open as PDFs
# runs go test $@ in current dir, e.g.:
# ./test_graphviz.sh -v
# ./test_graphviz.sh -v -run TestTraceHTTP
# ./test_graphviz.sh -v github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/reporter/
graphdir="${DOT_GRAPHDIR:=$(pwd)}"
DOT_GRAPHS=1 DOT_GRAPHDIR="$graphdir" go test "$@"
OPEN="echo"
if [ "$(uname)" == "Darwin" ] && [ -t 1 ]; then # open if interactive mac shell
    OPEN="sleep 2; open" # seems to avoid Preview.app permission error
fi
all=""
for i in $graphdir/*.dot; do
    outf="${i%.dot}.pdf"
    # draw graph for any new DOT files
    if [ ! -f "$outf" ]; then
        echo "GRAPHVIZ $outf"
        dot -Tpdf $i -o $outf 2>&1 >/dev/null | grep -v CGFontGetGlyph
        all+=" $outf"
    fi
done
eval $OPEN $all
