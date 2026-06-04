package logon_service

import (
	"context"
	"errors"
	"server/api/pb"
	"server/internal/game/role"
	"server/internal/share/model"
	"server/pkg/db"
	"server/pkg/util"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/zap"
)

type loader struct {
	loading chan *EvtLogin
	mgr     *LogonService
}

func newLoader(mgr *LogonService) *loader {
	return &loader{
		loading: make(chan *EvtLogin, opChanSize),
		mgr:     mgr,
	}
}

func (l *loader) post(op *EvtLogin) {
	l.loading <- op
}

func (l *loader) run(ctx context.Context, wait *sync.WaitGroup) {
	const (
		batchSize     = 100
		flushInterval = 50 * time.Millisecond
	)

	batch := make([]*EvtLogin, 0, batchSize)
	t := time.NewTicker(flushInterval)
	defer func() {
		t.Stop()
		wait.Done()
	}()

	flush := func() {
		if len(batch) > 0 {
			l.loadBatch(batch)
			batch = batch[:0]
		}
	}

	for {
		select {
		case p := <-l.loading:
			batch = append(batch, p)
			if len(batch) >= batchSize {
				flush()
				t.Reset(flushInterval)
			}
		case <-t.C:
			flush()
		case <-ctx.Done():
			return
		}
	}
}

func (l *loader) loadBatch(batch []*EvtLogin) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	pipe := db.Redis.Pipeline()
	for _, op := range batch {
		pipe.HGetAll(ctx, model.KeyRole(op.Login.RoleID))
	}
	cmd, err := pipe.Exec(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		zap.L().Error("[login] redis load batch failed", zap.Error(err))
		return
	}

	batchFromDB := make([]*EvtLogin, 0, len(cmd))
	for i, c := range cmd {
		data := c.(*redis.MapStringStringCmd).Val()
		if /*c.Err() == nil*/ len(data) > 0 { // 加载成功
			op := batch[i]
			l.mgr.postDBLoaded(&EvtDBLoaded{
				Login: op.Login,
				Data: &role.DataToSave{
					RoleID: op.Login.RoleID,
					Data:   data,
				},
			})
		} else /*if errors.Is(c.Err(), redis.Nil)*/ { // redis里没有
			batchFromDB = append(batchFromDB, batch[i])
		}
	}

	if len(batchFromDB) > 0 {
		l.loadFromDBBatch(ctx, batchFromDB)
	}
}

func (l *loader) loadFromDBBatch(ctx context.Context, batch []*EvtLogin) {
	ids := make([]uint64, 0, len(batch))
	for _, op := range batch {
		ids = append(ids, op.Login.RoleID)
	}

	filter := bson.M{"id": bson.M{"$in": ids}}
	cursor, err := db.MongoDB().Collection("roles").Find(ctx, filter)
	if err != nil {
		zap.L().Error("[login] find role failed", zap.Error(err))
		return
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var roles []*role.DataToSave
	if err = cursor.All(ctx, &roles); err != nil {
		zap.L().Error("[login] cursor all failed", zap.Error(err))
		return
	}
	result := make(map[uint64]*role.DataToSave, len(roles))
	for _, r := range roles {
		result[r.RoleID] = r
	}

	for _, op := range batch {
		var rData *role.DataToSave
		if r, ok := result[op.Login.RoleID]; ok {
			rData = r
		} else {
			rData, _ = newRoleDBData(op.Login.RoleID)
		}
		l.mgr.postDBLoaded(&EvtDBLoaded{
			Login: op.Login,
			Data:  rData,
		})
	}
}

func newRoleDBData(roleID uint64) (*role.DataToSave, error) {
	rData := pb.RoleData{
		ID:    roleID,
		Name:  util.IToString(roleID),
		Level: 1,
	}

	rd := &role.DataToSave{
		RoleID: roleID,
		Data:   make(map[string]string),
	}

	str, err := sonic.MarshalString(&rData)
	if err != nil {
		zap.L().Error("[login] marshal role data", zap.Error(err))
		return nil, err
	}
	rd.Set(pb.TypeComp_TCBase, str)
	return rd, nil
}
