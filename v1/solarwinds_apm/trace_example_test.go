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
package solarwinds_apm_test

import (
	"context"

	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm"
)

func ExampleNewTrace() {
	f0 := func(ctx context.Context) { // example span
		l, _ := solarwinds_apm.BeginSpan(ctx, "myDB",
			"Query", "SELECT * FROM tbl1",
			"RemoteHost", "db1.com")
		// ... run a query ...
		l.End()
	}

	// create a new trace, and a context to carry it around
	ctx := solarwinds_apm.NewContext(context.Background(), solarwinds_apm.NewTrace("myExample"))
	// do some work
	f0(ctx)
	// end the trace
	solarwinds_apm.EndTrace(ctx)
}

func ExampleBeginSpan() {
	// create trace and bind to context, reporting first event
	ctx := solarwinds_apm.NewContext(context.Background(), solarwinds_apm.NewTrace("baseSpan"))
	// ... do something ...

	// instrument a DB query
	l, _ := solarwinds_apm.BeginSpan(ctx, "DBx", "Query", "SELECT * FROM tbl")
	// .. execute query ..
	l.End()

	// end trace
	solarwinds_apm.EndTrace(ctx)
}
