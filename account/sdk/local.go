package sdk

import (
	"context"
	"server/pkg/pb"
)

// -----------------本地验证-------------------
type SdkLocal struct {
}

func (t *SdkLocal) Login(ctx context.Context, req *pb.C2SLogin) error {
	return nil
}
