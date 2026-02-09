package login

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
	"math/rand"
	"server/account/acc_db"
	"server/account/logic/sdk"
	"server/pkg/db"
	"server/pkg/gnet"
	"server/pkg/logger"
	"server/pkg/model"
	"server/pkg/pb"
	"server/pkg/thread"
	"server/pkg/util"
	"sync/atomic"
	"time"
)

// todo 服务断线
// 重进

const (
	LoginCD = 3
)

type Op int

const (
	OpLogin Op = iota
	OpAfterSDKCheck
	OpRoleClear
)

type EvtParam struct {
	Op    Op
	Login *pb.S2SReqLogin
	Acc   *Account
	Clear *pb.S2SRoleClear
}

var (
	evt         = make(chan EvtParam, 4096)
	loading     *loader
	tokenBucket = TokenBucketMax
	LastRunTime int64
	loginTime   = make(map[string]int64)
	curAccID    atomic.Uint64
)

func Start(ctx context.Context) {
	accID, err := GetCurAccID(ctx)
	if err != nil {
		zap.L().Error("get max account id", zap.Error(err))
		return
	}
	curAccID.Store(accID)
	zap.L().Info("account id:", zap.Uint64("max account", accID))

	loading = newLoader()

	for i := 0; i < 3; i++ {
		thread.GoSafe(func() {
			loading.run(ctx)
		})
	}
	thread.GoSafe(func() {
		t := time.NewTicker(time.Minute)
		tFillBucket := time.NewTicker(time.Millisecond * 200)
		defer func() {
			logger.Debug("stop Login mgr run")
			t.Stop()
		}()

		for {
			select {
			case e := <-evt:
				onEvent(ctx, e)
			case now := <-t.C:
				atomic.StoreInt64(&LastRunTime, now.Unix())
				checkTimeout(now.Unix())
			case <-tFillBucket.C:
				refillTokenBucket()
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

func onEvent(ctx context.Context, e EvtParam) {
	defer func() {
		if err := recover(); err != nil {
			thread.PrintStack("Login event:", err)
		}
	}()

	switch e.Op {
	case OpLogin:
		login(ctx, e.Login)
	case OpAfterSDKCheck:
		AfterSDKCheck(e.Acc, e.Login)
	case OpRoleClear:
		onRoleLogout(model.GetAccID(e.Clear.RoleID), e.Clear.Seq)
	default:
	}
}

func tryConsumeTokenBucket() bool {
	if tokenBucket < 1 {
		return false
	} else {
		tokenBucket--
		return true
	}
}

const TokenBucketMax = int32(5000)

func refillTokenBucket() {
	tokenBucket += TokenBucketMax / 5
	if tokenBucket > TokenBucketMax {
		tokenBucket = TokenBucketMax
	}
}

func checkTimeout(now int64) {
	const MaxCount = 1000
	cnt := 0
	for k, v := range loginTime {
		cnt++
		if cnt >= MaxCount {
			return
		}
		if now-v > LoginCD*2 {
			delete(loginTime, k)
		}
	}
}

func login(ctx context.Context, req *pb.S2SReqLogin) {
	if code := canSdkCheck(req); code != pb.LoginCode_LCSuccess {
		gnet.SendToRole(&pb.S2CLogin{Code: code}, req.SesID, 0)
	} else {
		sdkCheck(ctx, req)
	}
}

func canSdkCheck(req *pb.S2SReqLogin) pb.LoginCode {
	if req.Req.Account == "" {
		return pb.LoginCode_LCAccountEmpty
	}

	if !tryConsumeTokenBucket() {
		return pb.LoginCode_LCServerBusy
	}

	req.Req.Account = RealAcc(req.Req.SdkNo, req.Req.Account)

	now := time.Now().Unix()
	if now-loginTime[req.Req.Account] < LoginCD {
		return pb.LoginCode_LCCD
	}

	// 白名单

	loginTime[req.Req.Account] = now
	return pb.LoginCode_LCSuccess
}

func sdkCheck(ctx context.Context, req *pb.S2SReqLogin) {
	var s = sdk.CreateSdk(req.Req.SdkNo)
	if s == nil {
		zap.S().Errorf("can not create sdk:%d %s", req.Req.SdkNo, req.Req.String())
	}

	if util.Debug {
		debugAcc[req.Req.Account] = &debugCheck{
			AccID: debugGetAccID(req.Req.Account, req.Req.SdkNo),
			Ok:    false,
		}
	}

	go func() {
		defer func() {
			if err := recover(); err != nil {
				thread.PrintStack("Login check err:", err, req.Req.String())
			}
		}()

		err := s.Login(ctx, req.Req)
		if err != nil {
			return
		}

		PostLoading(req)
	}()
}

func afterSDKCheck(acc *Account, req *pb.S2SReqLogin) pb.LoginCode {
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
	now := time.Now().Unix()
	gameID, code := choseGame(acc.GameID, 0)
	if code != pb.LoginCode_LCSuccess {
		return code
	}

	acc.Time = now
	if acc.Passwd == 0 {
		acc.Passwd = rand.Uint64()
	}

	acc.Seq++
	acc.GameID = gameID

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	err := acc.SaveLoginData(ctx)
	if err != nil {
		logger.Warnf("save acc Login data err:%v", err)
		return pb.LoginCode_LCServerErr
	}

	req.RoleID = model.GetRoleID(acc.AccID)
	req.ReConnToken = acc.Passwd
	req.Seq = acc.Seq

	return pb.LoginCode_LCSuccess
}

func AfterSDKCheck(acc *Account, req *pb.S2SReqLogin) {
	if acc == nil { // 加载失败
		gnet.SendToRole(&pb.S2CLogin{Code: pb.LoginCode_LCServerErr}, req.SesID, 0)
		return
	}

	if util.Debug {
		DebugCheck(acc, req)
	}

	if code := afterSDKCheck(acc, req); code != pb.LoginCode_LCSuccess {
		gnet.SendToRole(&pb.S2CLogin{Code: code}, req.SesID, 0)
	} else {
		gnet.SendToGame(acc.GameID, req, 0, 0)
		zap.L().Info("acc login success", zap.Uint64("accID", acc.AccID), zap.Any("acc", acc))
	}
}

type debugCheck struct {
	AccID uint64
	Ok    bool
}

var debugAcc = make(map[string]*debugCheck)

func DebugCheck(acc *Account, req *pb.S2SReqLogin) {
	chk, ok := debugAcc[req.Req.Account]
	if !ok {
		zap.L().Error("not exist", zap.Any("req", req))
	}
	if chk.AccID == 0 {
		chk.AccID = debugGetAccID(req.Req.Account, req.Req.SdkNo)
	}
	if chk.AccID != acc.AccID {
		zap.L().Error("not match", zap.Any("req", req), zap.Any("acc", acc), zap.Any("real", chk))
	}
	chk.Ok = true
}

func debugGetAccID(account string, sdk pb.ESdkNumber) uint64 {
	filter := bson.M{"device": account}
	switch sdk {
	case pb.ESdkNumber_Google:
		filter = bson.M{"googleid": account}
	case pb.ESdkNumber_Apple:
		filter = bson.M{"appleid": account}
	case pb.ESdkNumber_Facebook:
		filter = bson.M{"fbid": account}
	default:

	}
	acc := Account{}
	err := db.MongoDB.Collection(acc_db.AccountTable).FindOne(context.Background(), filter).Decode(&acc)
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		zap.L().Error("find account err", zap.Error(err))
		return 0
	}
	return acc.AccID
}
