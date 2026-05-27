package auth

import (
	"context"
	"hash/fnv"
	"math/rand"
	pb "server/api/pb"
	"server/internal/account/sdk"
	"server/internal/share/model"
	"server/pkg/db"
	"server/pkg/queue"
	"server/pkg/thread"
	"time"

	"go.uber.org/zap"
)

const (
	LoginCD    = 3
	LoadThread = 3
)

type Op int

const (
	OpLogin Op = iota
	OpAfterSDKCheck
	OpRoleClear
	OpLoginFail
)

type Event struct {
	Op    Op
	Login *pb.S2SReqLogin
	Acc   *Account
	Clear *pb.S2SRoleClear
	Code  pb.LoginCode
}

var (
	evt     = queue.NewSwapQueue[Event](9640, 96400)
	loading []*AccountLoader
)

func StartService(ctx context.Context) {
	if err := InitDistributedAccID(ctx); err != nil {
		zap.L().Fatal("init distributed acc id failed", zap.Error(err))
	}

	loading = make([]*AccountLoader, 0)
	for i := 0; i < LoadThread; i++ {
		l := newAccountLoader()
		loading = append(loading, l)
		thread.GoSafe(func() {
			l.run(ctx)
		})
	}
	thread.GoSafe(func() {
		for {
			select {
			case <-evt.Sig():
				evt.Range(func(event Event) bool {
					processEvent(event)
					return true
				})
			case <-ctx.Done():
				return
			}
		}
	})
}

func PushToLoader(data *pb.S2SReqLogin) {
	idx := hashAccount(data.Req.Account) % LoadThread
	loading[idx].loading <- data
	// zap.L().Debug("push to loader", zap.Any("req", data), zap.Uint32("idx", idx))
}

func dispatchEvent(e Event) {
	err := evt.PushAndWake(e)
	if err != nil {
		zap.L().Error("dispatch event failed", zap.Error(err))
		if e.Login != nil {
			sendLoginFailure(e.Login, pb.LoginCode_LCServerBusy)
		}
	}
}

func HandleLoginRequest(req *pb.S2SReqLogin) {
	DebugAddWait()

	dispatchEvent(Event{
		Op:    OpLogin,
		Login: req,
	})
}

func processEvent(e Event) {
	defer func() {
		if err := recover(); err != nil {
			thread.PrintStack("Login event:", err)
		}
	}()

	switch e.Op {
	case OpLogin:
		login(e.Login)
	case OpAfterSDKCheck:
		OnSDKAuthSuccess(e.Acc, e.Login)
	case OpLoginFail:
		sendLoginFailure(e.Login, e.Code)
	case OpRoleClear:
		HandleRoleLogout(model.GetAccID(e.Clear.RoleID), e.Clear.Seq)
	default:
	}
}

func login(req *pb.S2SReqLogin) {
	DebugAdd(req)

	if code := checkLoginRateLimit(req); code != pb.LoginCode_LCSuccess {
		dispatchEvent(Event{
			Op:    OpLoginFail,
			Code:  code,
			Login: req,
		})
		return
	}
	authenticateWithSDK(req)
}

func checkLoginRateLimit(req *pb.S2SReqLogin) pb.LoginCode {
	if req.Req.Account == "" {
		return pb.LoginCode_LCAccountEmpty
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	ok, err := db.Redis.SetNX(ctx, model.KeyAccLoginCD(req.Req.SdkType, req.Req.Account), 1, time.Duration(LoginCD)*time.Second).Result()
	if err != nil {
		zap.L().Error("redis setnx error", zap.Error(err))
		return pb.LoginCode_LCServerErr
	}
	if !ok { // 没拿到锁，说明在 CD 中
		return pb.LoginCode_LCCD
	}
	return pb.LoginCode_LCSuccess
}

func authenticateWithSDK(req *pb.S2SReqLogin) {
	var s = sdk.CreateSdk(req.Req.SdkType)
	if s == nil {
		zap.S().Errorf("can not create sdk:%d %s", req.Req.SdkType, req.Req.String())
		dispatchEvent(Event{
			Op:    OpLoginFail,
			Code:  pb.LoginCode_LCSDKErr,
			Login: req,
		})
	}

	go func() {
		defer func() {
			if err := recover(); err != nil {
				thread.PrintStack("Login check err:", err, req.Req.String())
				dispatchEvent(Event{
					Op:    OpLoginFail,
					Code:  pb.LoginCode_LCSdkCheckFaild,
					Login: req,
				})
			}
		}()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		err := s.Login(ctx, req.Req)
		if err != nil {
			dispatchEvent(Event{
				Op:    OpLoginFail,
				Code:  pb.LoginCode_LCSdkCheckFaild,
				Login: req,
			})
			return
		}

		PushToLoader(req)
	}()
}

func finalizeLoginSession(acc *Account, req *pb.S2SReqLogin) pb.LoginCode {
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
	if req.Req.Reconnect && acc.Passwd != 0 && req.ReConnToken != acc.Passwd {
		return pb.LoginCode_LCCanNotReConn
	}

	gameID, code := allocateGameServer(acc.GameID, 0)
	if code != pb.LoginCode_LCSuccess {
		return code
	}

	if acc.Passwd == 0 {
		acc.Passwd = rand.Uint64()
	}

	oldSeq := acc.Seq
	acc.Seq = oldSeq + 1
	acc.GameID = gameID

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	success, err := acc.SaveLoginData(ctx, oldSeq)
	if err != nil {
		zap.S().Warnf("save acc Login data err:%v", err)
		return pb.LoginCode_LCServerErr
	}
	if !success {
		zap.L().Warn("concurrent login conflict detected for accID", zap.Uint64("acc", acc.AccID), zap.Uint32("oldSeq", oldSeq))
		return pb.LoginCode_LCServerBusy // 返回繁忙，让客户端发起重试重新走流程
	}

	req.RoleID = model.GetRoleID(acc.AccID)
	req.ReConnToken = acc.Passwd
	req.Seq = acc.Seq

	return pb.LoginCode_LCSuccess
}

func OnSDKAuthSuccess(acc *Account, req *pb.S2SReqLogin) {
	// zap.L().Debug("loading finish", zap.Any("req", req), zap.Any("acc", acc))
	if acc == nil { // 加载失败
		sendLoginFailure(req, pb.LoginCode_LCServerErr)
		return
	}

	if code := finalizeLoginSession(acc, req); code != pb.LoginCode_LCSuccess {
		sendLoginFailure(req, code)
	} else {
		req.ConnectedAcc = acc.Binds
		DebugCheck(req, true, acc)
		// gnet.SendToGame(acc.GameID, req, 0, 0)
		zap.L().Info("acc login success", zap.Uint64("accID", acc.AccID), zap.Any("acc", acc))
	}
}

func hashAccount(acc string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(acc))
	return h.Sum32()
}

func sendLoginFailure(req *pb.S2SReqLogin, code pb.LoginCode) {
	DebugCheck(req, false, nil)
	zap.L().Warn("login fail", zap.Any("req", req), zap.Any("code", code))
	// gnet.SendToRole(&pb.S2CLogin{Code: code}, req.SesID, 0)
}
