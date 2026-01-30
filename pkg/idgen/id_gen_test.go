package idgen

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGenUpperHex(t *testing.T) {
	Init(1)
	hexId := ConvertToUpperHex(GenUint64())
	println("Generated Upper Hex ID:", hexId)

	hexMap := make(map[string]int32)
	for i := 0; i < 1000; i++ {
		hexStr := ConvertToUpperHex(GenUint64())
		fmt.Printf("%d: %s %d\n", i, hexStr, len(hexStr))
		hexMap[hexStr]++
	}
	for _, c := range hexMap {
		require.EqualValues(t, c, 1)
	}
}

// BenchmarkGenUpperHex-8   	 4923463	       244.1 ns/op	      48 B/op	       3 allocs/op
func BenchmarkGenUpperHex(b *testing.B) {
	Init(1)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ConvertToUpperHex(GenUint64())
	}
}
