//go:build !linux
// +build !linux

package metrics

import "github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/bson"

func appendUname(bbuf *bson.Buffer) {}

func addHostMetrics(bbuf *bson.Buffer, index *int) {}
