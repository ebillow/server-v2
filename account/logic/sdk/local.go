package sdk

import (
	"server/pkg/pb"
)

// -----------------本地验证-------------------
type SdkLocal struct {
}

func (t *SdkLocal) Login(req *pb.C2SLogin) error {
	return nil
}
