package login

import (
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"server/account/logic/sdk"
	"server/pkg/gnet"
	"server/pkg/logger"
	"server/pkg/model"
	"server/pkg/pb"
	"server/pkg/thread"
	"sync/atomic"
	"time"
)

const (
	LoginCD = 3
)

type Op int

const (
	OpLogin Op = iota
	OpServerState
	OpRoleClear
)

type EvtParam struct {
	Op       Op
	Login    *pb.S2SReqLogin
	RoleInfo *pb.S2SRoleOffline
	SerState *pb.MsgServerClose
}

var (
	evt     = make(chan EvtParam, 4096)
	loading *loader
	// tokenBucket int32
	LastRunTime int64
)

func Start(ctx context.Context) {
	loading = newLoader()

	thread.GoSafe(func() {
		loading.run(ctx)
	})
	thread.GoSafe(func() {
		t := time.NewTicker(time.Second * 10)
		// tFillBucket := time.NewTicker(time.Millisecond * 100)
		defer func() {
			logger.Debug("stop Login mgr run")
			t.Stop()
		}()

		for {
			select {
			case e := <-evt:
				onEvent(e)
			case now := <-t.C:
				atomic.StoreInt64(&LastRunTime, now.Unix())
			// case <-tFillBucket.C:
			//	refillTokenBucket()
			case <-ctx.Done():
				return
			}
		}
	})
}

func PostLoading(data *pb.S2SReqLogin) {
	loading.loading <- data
}

func PostEvt(e EvtParam) {
	evt <- e
}

func Login(data *pb.S2SReqLogin) {
	PostEvt(EvtParam{
		Op:    OpLogin,
		Login: data,
	})
}

func onEvent(e EvtParam) {
	defer func() {
		if err := recover(); err != nil {
			thread.PrintStack("Login event:", err)
		}
	}()

	switch e.Op {
	case OpLogin:
		login(e.Login)
	case OpServerState:
		onServerState(e.SerState)
	case OpRoleClear:
		onRoleLogout(e.RoleInfo.Acc, e.RoleInfo.Seq)
	default:

	}
}

// func tryConsumeTokenBucket() bool {
//	if tokenBucket < 1 {
//		return false
//	} else {
//		tokenBucket--
//		return true
//	}
// }
//
// func refillTokenBucket() {
//	tokenBucket += int32(50)
// }

func login(req *pb.S2SReqLogin) {
	if err := canSdkCheck(req); err != nil {
		// todo 发消息
	} else {
		sdkCheck(req)
	}
}

func canSdkCheck(req *pb.S2SReqLogin) error {
	if req.Req.Account == "" {
		return errors.New("account is base")
	}

	// if !tryConsumeTokenBucket() {
	//	network.SendToGate(req.GtID, pb.MsgIDS2S_Acc2GtLoginAck, &pb.MsgLoginAck{Ret: pb.LoginCode_LCServerBusy, Data: req})
	//	return
	// }

	// rawAcc := share.GetRawAcc(req.SdkNo, req.Channel, req.Uid)
	//
	// if skipAllLoginChk || isInLocalChkList(rawAcc) {
	// 	checkAccState(req)
	// 	return
	// }
	//
	// isWhiteListAcc := isInWhiteList(rawAcc, req.Dev)
	// closeInfo, isClose := serverCloseInfo(req.Area)
	// if isClose && !isWhiteListAcc {
	// 	return
	// }
	return nil
}

func sdkCheck(req *pb.S2SReqLogin) {
	var s = sdk.CreateSdk(req.Req.SdkNo)
	if s == nil {
		zap.S().Errorf("can not create sdk:%d %s", req.Req.SdkNo, req.Req.String())
	}

	go func() {
		defer func() {
			if err := recover(); err != nil {
				thread.PrintStack("Login check err:", err, req.Req.String())
			}
		}()

		err := s.Login(req.Req)
		if err != nil {
			return
		}

		PostLoading(req)
	}()
}

func afterCheck(accData *Account, req *pb.S2SReqLogin) {
	// if data.Freeze { // 封号了
	// 	if data.FreezeEndTime == 0 || (data.FreezeEndTime > 0 && data.FreezeEndTime >= util.GetNowTimeS()) {
	// 		network.SendToGate(loginReq.GtID, &pb.S2SAcc2GtLogin{Code: pb.LoginCode_LCFreeze, Login: loginReq, RetDesc: util.ToString(data.FreezeEndTime)})
	// 		return
	// 	}
	// }
	// if data.BindDev != "" && (loginReq.CliInfo == nil || data.BindDev != loginReq.CliInfo.DevID) { // 绑定设备
	// 	network.SendToGate(loginReq.GtID, &pb.S2SAcc2GtLogin{Code: pb.LoginCode_LCBindDev, Login: loginReq})
	// 	return
	// }

	req.RoleID = model.GetRoleID(accData.AccID)

	ret, code := getGameCanEnter(accData, req)
	gameID := int32(0)
	if code != pb.LoginCode_LCSuccess {

	}
	req.ReConnToken = ret.Passwd
	req.Seq = ret.Seq
	gameID = ret.GameID
	gnet.SendToGame(gameID, req, 0, 0)
}

func realAcc(sdk pb.ESdkNumber, acc string) string {
	return fmt.Sprintf("%d:%s", sdk, acc)
}
