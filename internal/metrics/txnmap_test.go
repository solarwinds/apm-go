package metrics

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTxnMap(t *testing.T) {
	m := newTxnMap(3)
	assert.EqualValues(t, 3, m.cap())
	assert.True(t, m.isWithinLimit("t1"))
	assert.True(t, m.isWithinLimit("t2"))
	assert.True(t, m.isWithinLimit("t3"))
	assert.False(t, m.isWithinLimit("t4"))
	assert.True(t, m.isWithinLimit("t2"))
	assert.True(t, m.isOverflowed())

	m.SetCap(4)
	m.reset()
	assert.EqualValues(t, 4, m.cap())
	assert.False(t, m.isOverflowed())
}
