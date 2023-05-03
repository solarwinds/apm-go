//go:build linux
// +build linux

// Copyright (C) 2023 SolarWinds Worldwide, LLC. All rights reserved.

package metrics

import (
	"strings"
	"syscall"
	"testing"

	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/bson"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestAppendUname(t *testing.T) {
	bbuf := bson.NewBuffer()
	appendUname(bbuf)
	bbuf.Finish()
	m := bsonToMap(bbuf)

	var sysname, release string

	var uname syscall.Utsname
	if err := syscall.Uname(&uname); err == nil {
		sysname = utils.Byte2String(uname.Sysname[:])
		release = utils.Byte2String(uname.Release[:])
		sysname = strings.TrimRight(sysname, "\x00")
		release = strings.TrimRight(release, "\x00")
	}

	assert.Equal(t, sysname, m["UnameSysName"])
	assert.Equal(t, release, m["UnameVersion"])
}
