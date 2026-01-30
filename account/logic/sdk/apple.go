package sdk

import (
	"server/pkg/pb"
)

type Apple struct {
}

func (t *Apple) Login(req *pb.C2SLogin) error {
	return nil
}
