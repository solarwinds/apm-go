package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRateCounts(t *testing.T) {
	rc := &RateCounts{}

	rc.RequestedInc()
	assert.EqualValues(t, 1, rc.Requested())

	rc.SampledInc()
	assert.EqualValues(t, 1, rc.Sampled())

	rc.LimitedInc()
	assert.EqualValues(t, 1, rc.Limited())

	rc.TracedInc()
	assert.EqualValues(t, 1, rc.Traced())

	rc.ThroughInc()
	assert.EqualValues(t, 1, rc.Through())

	original := *rc
	cp := rc.FlushRateCounts()

	assert.Equal(t, original, *cp)
	assert.Equal(t, &RateCounts{}, rc)
}
