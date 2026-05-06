package util

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestItoString(t *testing.T) {
	i := 12312312312312313
	str := ItoString(i)
	require.Equal(t, "12312312312312313", str)

	i = 2
	str = ItoString(i)
	require.Equal(t, "2", str)

	i2 := uint64(math.MaxUint64)
	str = ItoString(i2)
	require.Equal(t, "18446744073709551615", str)

	ii := -1231231231233
	str = ItoString(ii)
	require.Equal(t, "-1231231231233", str)
}

func TestFtoString(t *testing.T) {
	f := 123123123123.12312
	str := FtoString(f)
	require.Equal(t, "123123123123.12312", str)

	f = 1.12312231231
	str = FtoString(f)
	require.Equal(t, "1.12312231231", str)

	f = -1.12312231231
	str = FtoString(f)
	require.Equal(t, "-1.12312231231", str)
}
