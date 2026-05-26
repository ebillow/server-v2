package auth

import (
	"context"
	"errors"
	"server/api/pb"
	"server/internal/share/model"
	"server/pkg/db"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"
)

const AccountCollection = "accounts"

const saveLoginLua = `
local key = KEYS[1]
local expected_seq = ARGV[1]
local current_seq = redis.call('HGET', key, 'seq')
if not current_seq then current_seq = '0' end

if current_seq == expected_seq then
	redis.call('HSET', key, 'game_id', ARGV[2], 'seq', ARGV[3], 'passwd', ARGV[4])
	return 1
end
return 0
`

// CAS 清除 GameID 的 Lua 脚本（用于登出）
const clearGameLua = `
local key = KEYS[1]
local expected_seq = ARGV[1]
local current_seq = redis.call('HGET', key, 'seq')
if not current_seq then current_seq = '0' end

if current_seq == expected_seq then
	redis.call('HSET', key, 'game_id', 0)
	return 1
end
return 0
`

type Account struct {
	AccID    uint64 `redis:"acc_id" bson:"acc_id"`
	Freeze   bool   `redis:"freeze" bson:"freeze"`
	GameID   uint8  `redis:"game_id" bson:"-"`
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

func AccFields() []string {
	return []string{"acc_id", "device", "apple_id", "google_id", "fb_id", "freeze", "game_id", "seq", "passwd"}
}

func (acc *Account) SaveLoginData(ctx context.Context, oldSeq uint32) (bool, error) {
	key := model.KeyAccount(acc.AccID)
	res, err := db.Redis.Eval(ctx, saveLoginLua, []string{key},
		oldSeq, acc.GameID, acc.Seq, acc.Passwd).Result()

	if err != nil && !errors.Is(err, redis.Nil) {
		return false, err
	}
	// 返回 1 表示更新成功，0 表示 seq 不匹配被别人改了
	return res.(int64) == 1, nil
}
func (acc *Account) ClearGameID(ctx context.Context, expectedSeq uint32) (bool, error) {
	key := model.KeyAccount(acc.AccID)
	res, err := db.Redis.Eval(ctx, clearGameLua, []string{key}, expectedSeq).Result()

	if err != nil && !errors.Is(err, redis.Nil) {
		return false, err
	}
	return res.(int64) == 1, nil
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

// InitDistributedAccID 在节点启动时同步一次最大ID到Redis
func InitDistributedAccID(ctx context.Context) error {
	// 先检查 Redis 是否已经有值（其他节点可能已经初始化过了）
	exists, err := db.Redis.Exists(ctx, model.RedisKeyIDs).Result()
	if err != nil {
		return err
	}
	if exists > 0 {
		return nil
	}

	// 如果Redis没有，从Mongo查最大的
	acc := &Account{}
	opts := options.FindOne().SetSort(bson.M{acc.FieldAccID(): -1})
	err = db.MongoDB().Collection(acc.CollectionName()).FindOne(ctx, bson.M{}, opts).Decode(acc)

	var maxID = uint64(pb.ActorID_IDAccBegin)
	if err == nil {
		maxID = acc.AccID
	} else if !errors.Is(err, mongo.ErrNoDocuments) {
		return err
	}

	// 使用 SetNX 防止多节点同时启动时的覆盖竞争
	db.Redis.SetNX(ctx, model.RedisKeyIDs, maxID, 0)
	zap.L().Info("init distributed acc id", zap.Uint64("max_id", maxID))
	return nil
}

// GenerateNextAccID 使用 Redis INCR 生成分布式唯一 ID
func GenerateNextAccID(ctx context.Context) (uint64, error) {
	id, err := db.Redis.Incr(ctx, model.RedisKeyIDs).Result()
	if err != nil {
		return 0, err
	}
	return uint64(id), nil
}
