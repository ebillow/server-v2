package sdk

import (
	"server/pkg/pb"
)

// const (
//	FaceBookAppID       = "2942592536037408"
//	FaceBookAppSecret   = "8aad0a0155444d0257dbf9e5105e8c77"
//	FaceBookAccessToken = FaceBookAppID + "|" + FaceBookAppSecret
// )

type Facebook struct {
}

// https://graph.facebook.com/debug_token?access_token={App-token}&input_token={User-token}
// 以上的参数，User-token为用户的token，App-token为APP的token，值为 {Your AppId}%7C{Your AppSecret}。其中，%7C为urlencode的 | 符号
//
// type facebookRet struct {
//	AppId     string //"app_id": 000000000000000,
//	ExpiresAt string //"expires_at": 1352419328,
//	IsValid   bool   //"is_valid": true,
//	IssuedAt  string //"issued_at": 1347235328,
//	UserID    string //"user_id": 100207059
// }

func (t *Facebook) Login(req *pb.C2SLogin) error {
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
