package codec

import (
	"server/pkg/pb"
	"testing"
)

func TestEncode(t *testing.T) {
	for i := 0; i < 2; i++ {
		out, bp, err := Encode(&pb.NatsMsg{
			MsgID:   1,
			Data:    nil,
			SerID:   2,
			SerType: 0,
			RoleID:  0,
			SesID:   0,
			Forward: false,
		})
		if err != nil {
			FreeBuffer(bp)
			t.Errorf("encode err: %v", err)
		}
		t.Logf("%p %p", out, bp)
		FreeBuffer(bp)
	}
}
