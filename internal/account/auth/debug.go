package auth

import (
	"context"
	"errors"
	"fmt"
	"server/api/pb"
	"server/internal/share/model"
	"server/pkg/db"
	"server/pkg/util"
	"sync"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
)

type debugCheck struct {
	AccID uint64
	Ok    bool
	SesID uint64
}

var debugAcc = make(map[string]*debugCheck)
var debugWait sync.WaitGroup

func debugAccKey(typ pb.SdkType, acc string) string {
	return fmt.Sprintf("%d:%s", typ, acc)
}

func DebugAddWait() {
	if util.Debug {
		debugWait.Add(1)
	}
}

func DebugAdd(req *pb.S2SReqLogin) {
	if util.Debug {
		debugAcc[debugAccKey(req.Req.SdkType, req.Req.Account)] = &debugCheck{
			SesID: req.SesID,
			Ok:    false,
		}
	}
}

func DebugCheck(req *pb.S2SReqLogin, success bool, acc *Account) {
	if !util.Debug {
		return
	}

	debugWait.Done()

	chk, ok := debugAcc[debugAccKey(req.Req.SdkType, req.Req.Account)]
	if !ok {
		zap.L().Fatal("not exist", zap.Any("req", req))
	}

	chk.Ok = success
	if success {
		accIDInDB := loadAccIDInDB(req.Req.Account, req.Req.SdkType)
		if accIDInDB != acc.AccID {
			zap.L().Fatal("db not match", zap.Any("req", req), zap.Any("acc", acc), zap.Uint64("real", accIDInDB))
		}

		accIDInCache, bindOk := loadAccInCache(req.Req.Account, req.Req.SdkType)
		if !bindOk {
			zap.L().Fatal("bind not exist", zap.Any("req", req), zap.Any("acc", acc), zap.Bool("bind", bindOk))
		}
		if accIDInCache != accIDInDB {
			zap.L().Fatal("cache not match", zap.Any("req", req), zap.Any("acc", acc), zap.Uint64("real", accIDInCache))
		}
	}
}

func loadAccInCache(account string, sdk pb.SdkType) (uint64, bool) {
	ctx := context.Background()
	accID, err := db.Redis.Get(ctx, model.KeyAccBind(FormatBindKey(sdk, account))).Uint64()
	if err != nil {
		zap.L().Fatal("loadAccInCache", zap.Error(err))
	}
	acc := Account{}
	ret := db.Redis.HMGet(ctx, model.KeyAccount(accID), AccFields()...)
	if ret.Err() != nil {
		zap.L().Fatal("loadAccInCache", zap.Error(err))
	}
	err = ret.Scan(&acc)
	if ret.Err() != nil {
		zap.L().Fatal("loadAccInCache", zap.Error(err))
	}

	if acc.AccID != accID {
		return accID, false
	}

	return accID, true
}

func loadAccIDInDB(account string, sdk pb.SdkType) uint64 {
	acc := Account{}

	filter := bson.M{acc.FieldBinds(): FormatBindKey(sdk, account)}
	err := db.MongoDB().Collection(acc.CollectionName()).FindOne(context.Background(), filter).Decode(&acc)
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		zap.L().Error("find account err", zap.Error(err))
		return 0
	}
	return acc.AccID
}
