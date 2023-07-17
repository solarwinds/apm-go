package reporter

import (
	"context"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/utils"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"
	"testing"
	"time"
)

func TestCreateInitMessage(t *testing.T) {
	tid := trace.TraceID{0x01, 0x02, 0x03, 0x04}
	r, err := resource.New(context.Background(), resource.WithAttributes(
		attribute.String("foo", "bar"),
		// service.name should be omitted
		attribute.String("service.name", "my cool service"),
	))
	require.NoError(t, err)
	a := time.Now()
	evt, err := createInitMessage(tid, r)
	b := time.Now()
	require.NoError(t, err)
	require.NotNil(t, evt)
	e, ok := evt.(*event)
	require.True(t, ok)
	require.Equal(t, tid, e.taskID)
	require.NotEqual(t, [8]byte{}, e.opID)
	require.True(t, e.t.After(a))
	require.True(t, e.t.Before(b))
	require.Equal(t, []attribute.KeyValue{
		attribute.String("foo", "bar"),
		attribute.Int("__Init", 1),
		attribute.String("APM.Version", utils.Version()),
	}, e.kvs)
	require.Equal(t, LabelUnset, e.label)
	require.Equal(t, "", e.layer)
	require.False(t, e.parent.IsValid())

}
