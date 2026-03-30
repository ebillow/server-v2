package util

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRandNotRepeated(t *testing.T) {
	r := NewRandUnique(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	ret := make(map[int]struct{})
	for i := 0; i != 20; i++ {
		v, err := r.Get()
		if i < 10 {
			require.NoError(t, err)
		} else {
			require.EqualError(t, err, "RandUnique empty")
		}
		if err == nil {
			if _, ok := ret[v]; ok {
				t.Fatalf("RandNotRepeated duplicated")
			}
			ret[v] = struct{}{}
		}
	}
}

func TestRandByWeight(t *testing.T) {
	r := NewRandByWeight[int]()
	cnt := 10
	for i := 1; i <= cnt; i++ {
		r.Add(1000, i)
	}
	ret := make(map[int]int)
	randCnt := 1000000
	for i := 0; i != randCnt; i++ {
		v, err := r.Get()
		require.NoError(t, err)
		ret[v]++
	}

	for k, v := range ret {
		t.Logf("%d, %d rate %f", k, v, float64(v)/float64(randCnt))
	}
}

func TestRandRange(t *testing.T) {
	v := RandRange(0, 1)
	require.Equal(t, 0, v)

	v = RandRange(0, 0)
	require.Equal(t, 0, v)

	for i := 0; i < 10000; i++ {
		v = RandRange(0, 10)
		require.True(t, v >= 0)
		require.True(t, v < 10)
	}

	for i := 0; i < 10000; i++ {
		v = RandRangeIntCloseInterval(0, 10)
		require.True(t, v >= 0)
		require.True(t, v <= 10)
	}
}
