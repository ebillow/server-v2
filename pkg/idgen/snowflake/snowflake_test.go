package snowflake

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGen_unique(t *testing.T) {
	InitGenerator(1)

	var count = 100000
	retMap := make(map[int64]struct{}, count)

	for i := 0; i < count; i++ {
		id := Gen()
		retMap[id] = struct{}{}
	}

	require.Len(t, retMap, count) // 确保不会有重复的ID出现
}

func TestGen_cost(t *testing.T) {
	InitGenerator(1)

	var count = ((-1 ^ (-1 << seqBit)) + 1) * 10
	start := time.Now()
	for i := 0; i < count; i++ {
		Gen()
	}
	end := time.Now()
	fmt.Printf("generate %d ids cost: %s\n", count, end.Sub(start))

	for i := 0; i < 10; i++ {
		id := Gen()
		bin := fmt.Sprintf("%064b", id)
		var fmtBin string
		for j, c := range bin {
			char := byte(c)
			fmtBin += string(char)
			if j == timestampBit || j == timestampBit+machineBit {
				fmtBin += " "
			}
		}

		fmt.Printf("id(%02d): %d; bin: %s\n", i, id, fmtBin)
	}
}

func BenchmarkGen(b *testing.B) {
	InitGenerator(1)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Gen()
	}
}

func TestExtract(t *testing.T) {
	InitGenerator(1)

	var id int64
	for i := 0; i < 10; i++ {
		id = Gen()
	}

	genTime, machineId, seq := Extract(id)
	fmt.Printf("id: %d | genTime: %s; machineId: %d; seq: %d\n", id, genTime, machineId, seq)
}

func TestExtractMachineId(t *testing.T) {
	for c := int64(1); c < 10; c++ {
		InitGenerator(int(c))

		var id int64
		for i := 0; i < 10; i++ {
			id = Gen()
		}
		machineId := ExtractMachineId(id)
		fmt.Printf("id: %d | machineId: %d\n", id, machineId)
		require.Equal(t, machineId, c)
	}
}
