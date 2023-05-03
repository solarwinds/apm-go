package host

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsPhysicalInterface(t *testing.T) {
	assert.True(t, IsPhysicalInterface("i-am-not-a-network-interface"))
}
