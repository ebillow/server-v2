package sdk

import (
	"context"
	"server/pkg/pb"
)

type Google struct {
}

func (t *Google) Login(ctx context.Context, req *pb.C2SLogin) error {
	// accInfo, err := verifyIdToken(req.Token)
	// if err != nil {
	//	return err
	// }
	// if req.Uid != accInfo.UserId {
	//	return errors.New("user id is not same")
	// }

	return nil
}
