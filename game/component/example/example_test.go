package example

import (
	"crypto/md5"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/require"
	"testing"
)

// BenchmarkMd5-10    	 1484458	       790.5 ns/op
func BenchmarkMd5(b *testing.B) {
	d := Data{
		OnlineCnt:  11231231,
		OfflineCnt: 21231231,
		Today:      false,
		Name:       "adsfadsfa",
		Award:      map[int32]struct{}{1: {}, 2: {}, 3: {}},
		Info:       []int32{1, 2, 3},
		cur:        323123,
	}
	for i := 0; i < b.N; i++ {
		bts, err := jsoniter.Marshal(d)
		require.NoError(b, err)
		md5.Sum(bts)
	}
}
