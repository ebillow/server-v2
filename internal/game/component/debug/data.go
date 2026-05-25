package debug

import (
	"context"
	"errors"
	pb2 "server/api/pb"
	"server/internal/game/role"
	model2 "server/internal/share/model"
	"server/pkg/db"

	"github.com/bytedance/sonic"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Data 做单元测试用，后续删除
type Data struct {
	OnlineCnt  int32
	OfflineCnt int32
	dirty      bool
}

func New(r *role.Role) *Data {
	return &Data{}
}

func (d *Data) IsDirty() bool {
	return d.dirty
}

func (d *Data) ClearDirty() {
	d.dirty = false
}

func (d *Data) Online(r *role.Role) {
	d.OnlineCnt++
	// d.dirty = true

	// todo  test:检查,正式时删除
	ctx := context.Background()
	ret, err := db.Redis.HGet(ctx, model2.KeyRole(r.ID), model2.GetCompName(pb2.TypeComp_TCBase)).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		zap.L().Error("redis hget", zap.Error(err))
		return
	}
	if err == nil {
		data := pb2.RoleData{}
		err = sonic.UnmarshalString(ret, &data)
		if err != nil {
			zap.L().Error("json unmarshal", zap.Error(err))
			return
		}
		if r.ID != data.ID {
			zap.L().Error("role id err")
			return
		}
	}
}

func (d *Data) Offline(r *role.Role) {
	d.OfflineCnt++
	if d.OnlineCnt != d.OfflineCnt {
		panic("d.OnlineCnt != d.OfflineCnt")
	}
}
