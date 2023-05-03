// © 2023 SolarWinds Worldwide, LLC. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package opentracing

import "github.com/opentracing/opentracing-go/ext"

// Map selected OpenTracing tag constants to SolarWinds Observability analogs
var otAPMMap = map[string]string{
	string(ext.Component): "OTComponent",

	string(ext.PeerService):  "RemoteController",
	string(ext.PeerAddress):  "RemoteURL",
	string(ext.PeerHostname): "RemoteHost",

	string(ext.HTTPUrl):        "URL",
	string(ext.HTTPMethod):     "Method",
	string(ext.HTTPStatusCode): "Status",

	string(ext.DBInstance):  "Database",
	string(ext.DBStatement): "Query",
	string(ext.DBType):      "Flavor",

	"resource.name": "TransactionName",
}

func translateTagName(key string) string {
	if k := otAPMMap[key]; k != "" {
		return k
	}
	return key
}
