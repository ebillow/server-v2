package dh

import (
	"fmt"
	"testing"
)

func TestDH(t *testing.T) {
	X1, E1 := Exchange()
	X2, E2 := Exchange()

	fmt.Println("Secret 1:", X1, E1)
	fmt.Println("Secret 2:", X2, E2)

	KEY1 := GetKey(X1, E2)
	KEY2 := GetKey(X2, E1)

	fmt.Println("KEY1:", KEY1)
	fmt.Println("KEY2:", KEY2)

	if KEY1.Cmp(KEY2) != 0 {
		t.Error("Diffie-Hellman failed")
	}
}

func BenchmarkDH(b *testing.B) {
	for i := 0; i < b.N; i++ {
		X1, E1 := Exchange()
		X2, E2 := Exchange()

		GetKey(X1, E2)
		GetKey(X2, E1)
	}
}

func TestDH2(t *testing.T) {
	for i := 0; i != 3000; i++ {
		go Exchange()
	}
}
