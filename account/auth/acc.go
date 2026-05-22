package auth

import (
	"context"
	"errors"
	"fmt"
	"server/pkg/db"
	"server/pkg/model"
	"server/pkg/pb"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const AccountCollection = "accounts"

type Account struct {
	AccID    uint64 `redis:"acc_id" bson:"acc_id"`
	Freeze   bool   `redis:"freeze" bson:"freeze"`
	GameID   uint8  `redis:"game_id" bson:"-"`
	Time     int64  `redis:"time" bson:"-"`
	Seq      uint32 `redis:"seq" bson:"-"`
	Passwd   uint64 `redis:"passwd" bson:"-"`
	Device   string `redis:"device" bson:"device,omitempty"`
	AppleID  string `redis:"apple_id" bson:"apple_id,omitempty"`
	GoogleID string `redis:"google_id" bson:"google_id,omitempty"`
	FbID     string `redis:"fb_id" bson:"fb_id,omitempty"`
}

type AccBind struct {
	Account string `redis:"account"`
	AccID   uint64 `redis:"acc_id"`
}

func FormatAccKey(typ pb.SdkType, acc string) string {
	return fmt.Sprintf("%d@%s", typ, acc)
}

func AccFields() []string {
	return []string{"acc_id", "device", "apple_id", "google_id", "fb_id", "freeze", "game_id", "time", "seq", "passwd"}
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

func (acc *Account) FieldAccID() string {
	return "acc_id"
}
func (acc *Account) FieldGoogleID() string {
	return "google_id"
}
func (acc *Account) FieldAppleID() string {
	return "apple_id"
}
func (acc *Account) FieldFBID() string {
	return "fb_id"
}
func (acc *Account) FieldDevice() string {
	return "device"
}
func (acc *Account) CollectionName() string {
	return AccountCollection
}

func GetCurAccID(ctx context.Context) (uint64, error) {
	acc := &Account{}
	opts := options.FindOne().SetSort(bson.M{acc.FieldAccID(): -1})
	err := db.MongoDB().Collection(acc.CollectionName()).FindOne(ctx, bson.M{}, opts).Decode(acc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return uint64(pb.ActorID_IDAccBegin), nil
		}
		return 0, err
	}
	return acc.AccID, nil
}
