package sdk

import (
	"errors"
	"server/pkg/logger"
	"server/pkg/pb"
)

/*
写每个sdk登录须做2件事
1.CreateSdk函数中new一个对应的实例
2.实现对应实例的login函数，返回成功或失败
*/

var (
	errSDKCheckFailed = errors.New("SDK check return failed")
)

type ISdkLogin interface {
	Login(req *pb.C2SLogin) error
}

// 创建sdk的实例 根据sdk number
func CreateSdk(no pb.ESdkNumber) ISdkLogin {
	switch no {
	case pb.ESdkNumber_Guest:
		return &SdkLocal{}
	case pb.ESdkNumber_Google:
		return &Google{}
	case pb.ESdkNumber_Facebook:
		return &Facebook{}
	case pb.ESdkNumber_Apple:
		return &Apple{}

	default:
		logger.Warnf("recv not exist sdk no:%d", no)
		return nil
	}
}
