package sdk

import (
	"context"
	"server/pkg/pb"
)

type Facebook struct {
}

func (t *Facebook) Login(ctx context.Context, req *pb.C2SLogin) error {
	// addr := "https://graph.facebook.com/debug_token"
	// param := make(url.Values)
	// param.Add("access_token", FaceBookAccessToken)
	// param.Add("input_token", req.Token)
	//
	// b, err := share.HttpGet(addr, []byte(param.Encode()), false)
	// if err != nil {
	//	logger.Warnf("facebook login check err:%v:%s", err, req.Uid)
	//	return err
	// }
	//
	// logger.Debugf("login sdk ret:%s", string(b))
	//
	// ret := &facebookRet{}
	// err = json.Unmarshal(b, ret)
	// if err != nil {
	//	return errors.New("facebook ret unmarshal err:" + err.Error())
	// }
	// if !ret.IsValid || ret.UserID != req.Uid {
	//	return errSDKCheckFailed
	// }
	return nil
}
