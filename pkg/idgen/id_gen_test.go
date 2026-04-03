package idgen

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGen(t *testing.T) {
	err := Init(1)
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		id, err := Gen()
		require.NoError(t, err)
		t.Log(id)
		ser := MachineID(id)
		require.Equal(t, ser, int64(1))
	}
}
