// Copyright (C) 2023 SolarWinds Worldwide, LLC. All rights reserved.

package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrappers(t *testing.T) {
	os.Unsetenv(envSolarWindsAPMCollector)
	os.Unsetenv(envSolarWindsAPMHistogramPrecision)
	Load()

	assert.NotEqual(t, nil, conf)
	assert.Equal(t, getFieldDefaultValue(&Config{}, "Collector"), GetCollector())
	assert.Equal(t, ToInteger(getFieldDefaultValue(&Config{}, "Precision")), GetPrecision())

	assert.NotEqual(t, nil, ReporterOpts())
}
