package util

import (
	"testing"
)

func TestNewUUID(t *testing.T) {
	u1 := NewUUID()
	u2 := NewUUID()

	if u1 == u2 {
		t.Fail()
	}
}

func TestNewUUIDRand(t *testing.T) {
	u1 := NewUUID()
	u2 := NewUUID()

	if u1 == u2 {
		t.Fail()
	}
}

func TestNewUUID2(t *testing.T) {
	list := make(map[string]bool)
	for i := 0; i != 1000000; i++ {
		u1 := NewUUID()
		if _, ok := list[u1]; ok {
			t.Fatalf("uuid repeated:%v %d", u1, i)
		} else {
			list[u1] = true
		}
	}
}
