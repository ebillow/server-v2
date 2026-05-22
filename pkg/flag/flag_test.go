package flag

import (
	"fmt"
	"server/api/pb"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSrvName(t *testing.T) {
	fmt.Println(pb.Server_Account.String())
}

// BenchmarkSrvName-10    	141012943	         8.378 ns/op
func BenchmarkSrvName(b *testing.B) {
	name := ""
	for i := 0; i < b.N; i++ {
		name = pb.Server_Account.String()
	}
	require.Equal(b, name, pb.Server_Account.String())
}

// BenchmarkSrvName2-10    	733219022	         1.576 ns/op
func BenchmarkSrvName2(b *testing.B) {
	name := ""
	for i := 0; i < b.N; i++ {
		name = pb.Server_name[int32(pb.Server_Account)]
	}
	require.Equal(b, name, pb.Server_Account.String())
}
