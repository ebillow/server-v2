package sdk

import (
	"context"
	"server/pkg/pb"
)

type Apple struct {
}

func (t *Apple) Login(ctx context.Context, req *pb.C2SLogin) error {
	return nil
}
