package utils

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRandom(t *testing.T) {
	m := make(map[string]bool)
	for i := 0; i < 10000; i++ {
		a := [16]byte{}
		Random(a[:])
		key := string(a[:])
		_, ok := m[key]
		require.False(t, ok)
		m[key] = true
	}
}
