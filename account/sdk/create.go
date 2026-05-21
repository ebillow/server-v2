package sdk

import (
	"context"
	"errors"
	"server/pkg/pb"

	"go.uber.org/zap"
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
	Login(ctx context.Context, req *pb.C2SLogin) error
}

// 创建sdk的实例 根据sdk number
func CreateSdk(no pb.SdkType) ISdkLogin {
	switch no {
	case pb.SdkType_Guest:
		return &SdkLocal{}
	case pb.SdkType_Google:
		return &Google{}
	case pb.SdkType_Facebook:
		return &Facebook{}
	case pb.SdkType_Apple:
		return &Apple{}

	default:
		zap.S().Warnf("recv not exist sdk no:%d", no)
		return nil
	}
}
