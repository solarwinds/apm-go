// Copyright (C) 2023 SolarWinds Worldwide, LLC. All rights reserved.

package solarwinds_apm_test

import (
	"os"
	"strings"
	"testing"

	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/config"
	g "github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/graphtest"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/reporter"
	"github.com/stretchr/testify/assert"
)

func TestCustomTransactionNameWithDomain(t *testing.T) {
	os.Setenv("SW_APM_PREPEND_DOMAIN", "true")
	config.Load()
	r := reporter.SetTestReporter() // set up test reporter

	// Test prepending the domain to transaction names.
	httpTestWithEndpoint(handler200CustomTxnName, "http://test.com/hello world/one/two/three?testq")
	r.Close(2)
	g.AssertGraph(t, r.EventBufs, 2, g.AssertNodeMap{
		// entry event should have no edges
		{"http.HandlerFunc", "entry"}: {Edges: g.Edges{}, Callback: func(n g.Node) {
			assert.Equal(t, "test.com", n.Map["HTTP-Host"])
		}},
		{"http.HandlerFunc", "exit"}: {Edges: g.Edges{{"http.HandlerFunc", "entry"}}, Callback: func(n g.Node) {
			// assert that response X-Trace header matches trace exit event
			assert.True(t, strings.HasPrefix(n.Map["TransactionName"].(string),
				"test.com/final-my-custom-transaction-name"),
				n.Map["TransactionName"].(string))
		}},
	})

	r = reporter.SetTestReporter() // set up test reporter

	// Test using X-Forwarded-Host if available.
	hd := map[string]string{
		"X-Forwarded-Host": "test2.com",
	}
	httpTestWithEndpointWithHeaders(handler200CustomTxnName, "http://test.com/hello world/one/two/three?testq", hd)
	r.Close(2)
	g.AssertGraph(t, r.EventBufs, 2, g.AssertNodeMap{
		// entry event should have no edges
		{"http.HandlerFunc", "entry"}: {Edges: g.Edges{}, Callback: func(n g.Node) {
			assert.Equal(t, "test.com", n.Map["HTTP-Host"])
		}},
		{"http.HandlerFunc", "exit"}: {Edges: g.Edges{{"http.HandlerFunc", "entry"}}, Callback: func(n g.Node) {
			// assert that response X-Trace header matches trace exit event
			assert.True(t, strings.HasPrefix(n.Map["TransactionName"].(string),
				"test2.com/final-my-custom-transaction-name"),
				n.Map["TransactionName"].(string))
		}},
	})
	os.Unsetenv("SW_APM_PREPEND_DOMAIN")
}
