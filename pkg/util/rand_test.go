package util

import "testing"

func TestRandNotRepeated(t *testing.T) {
	r := NewRandNotRepeated(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	for i := 0; i != 20; i++ {
		t.Log(r.Rand())
	}
}

func TestRand(t *testing.T) {
	r := NewRander()
	for i := 1; i != 10; i++ {
		r.Add(1000, i)
	}
	r.Add(20, 0)
	ret := make([]int, 10)
	randCnt := 1000000
	for i := 0; i != randCnt; i++ {
		ret[r.Get().(int)]++
	}

	for i := 0; i != 10; i++ {
		t.Logf("%d, %d rate %f", i, ret[i], float64(ret[i])/float64(randCnt))
	}
}

func TestRandRangeIntCloseInterval(t *testing.T) {
	min := 5
	max := 10
	for i := 0; i < 10000; i++ {
		v := RandRangeIntCloseInterval(min, max)
		if v < min || v > max {
			t.Failed()
		}
	}
}
