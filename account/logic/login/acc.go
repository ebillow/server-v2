package login

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"
	"server/account/acc_db"
	"server/pkg/db"
	"server/pkg/model"
	"server/pkg/pb"
	"time"
)

type Account struct {
	AccID    uint64 `redis:"acc_id"`
	Freeze   bool   `redis:"freeze"`
	GameID   int32  `redis:"game_id"`
	Time     int64  `redis:"time"`
	Seq      uint32 `redis:"seq"`
	Passwd   uint64 `redis:"passwd"`
	Device   string
	AppleID  string
	GoogleID string
	FbID     string
}

type AccBind struct {
	Account string `redis:"account"`
	AccID   uint64 `redis:"acc_id"`
}

func RealAcc(typ pb.ESdkNumber, acc string) string {
	return fmt.Sprintf("%d@%s", typ, acc)
}

func AccFields() []string {
	return []string{"acc_id", "freeze", "game_id", "time", "seq", "passwd"}
}

func (acc *Account) Update(ctx context.Context, account string) {
	expiration := time.Hour * 24 * 7
	pipe := db.Redis.Pipeline()
	keyAcc := model.KeyAccount(acc.AccID)
	pipe.HSet(ctx, keyAcc, "acc_id", acc.AccID, "freeze", acc.Freeze)
	pipe.Expire(ctx, keyAcc, expiration)
	keyBind := model.KeyAccBind(account)
	pipe.Set(ctx, keyBind, acc.AccID, expiration)

	_, err := pipe.Exec(ctx)
	if err != nil {
		zap.L().Error("redis hset acc_id failed", zap.Error(err))
		return
	}
}

func (acc *Account) SaveLoginData(ctx context.Context) error {
	return db.Redis.HSet(ctx, model.KeyAccount(acc.AccID), "game_id", acc.GameID, "time", acc.Time, "seq", acc.Seq, "passwd", acc.Passwd).Err()
}

func (acc *Account) LoadSeq(ctx context.Context) uint32 {
	v, err := db.Redis.HGet(ctx, model.KeyAccount(acc.AccID), "seq").Int()
	if err != nil {
		return 0
	}
	return uint32(v)
}

func GetCurAccID(ctx context.Context) (uint64, error) {
	acc := &Account{}
	opts := options.FindOne().SetSort(bson.M{"accid": -1})
	err := db.MongoDB.Collection(acc_db.AccountTable).FindOne(ctx, bson.M{}, opts).Decode(acc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return 0, nil
		} else {
			return 0, err
		}
	}
	return acc.AccID, nil
}
