package login_mgr

import (
	"context"
	"errors"
	jsoniter "github.com/json-iterator/go"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/zap"
	"server/game/role"
	"server/pkg/db"
	"server/pkg/logger"
	"server/pkg/model"
	"server/pkg/pb"
	"server/pkg/util"
	"sync"
	"time"
)

type loader struct {
	loading chan *Operator
}

func newLoader() *loader {
	return &loader{
		loading: make(chan *Operator, OpChanSize),
	}
}

func (l *loader) post(op *Operator) {
	l.loading <- op
}

func (l *loader) run(ctx context.Context, wait *sync.WaitGroup) {
	const (
		batchSize     = 100
		flushInterval = 200 * time.Millisecond
	)

	batch := make([]*Operator, 0, batchSize)
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

func (l *loader) loadBatch(batch []*Operator) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	pipe := db.Redis.Pipeline()
	for _, op := range batch {
		op.Data = &role.DataToSave{
			ID: op.Login.RoleID,
		}
		pipe.HGetAll(ctx, model.KeyRole(op.Data.ID))
	}
	cmd, err := pipe.Exec(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		zap.L().Error("[login] redis load batch failed", zap.Error(err))
		return
	}

	batchFromDB := make([]*Operator, 0, len(cmd))
	for i, c := range cmd {
		data := c.(*redis.MapStringStringCmd).Val()
		if /*c.Err() == nil*/ len(data) > 0 { // 加载成功
			op := batch[i]
			op.Op = OpUnmarshal
			op.Data.Data = data
			postOp(op)
		} else /*if errors.Is(c.Err(), redis.Nil)*/ { // redis里没有
			batchFromDB = append(batchFromDB, batch[i])
		}
	}

	if len(batchFromDB) > 0 {
		l.loadFromDBBatch(ctx, batchFromDB)
	}
}

func (l *loader) loadFromDBBatch(ctx context.Context, batch []*Operator) {
	ids := make([]uint64, 0, len(batch))
	for _, op := range batch {
		ids = append(ids, op.Data.ID)
	}

	filter := bson.M{"id": bson.M{"$in": ids}}
	cursor, err := db.MongoDB.Collection("roles").Find(ctx, filter)
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
		result[r.ID] = r
	}

	for _, op := range batch {
		op.Op = OpUnmarshal
		if r, ok := result[op.Login.RoleID]; ok {
			op.Data = r
		} else {
			rd, _ := newRoleInDB(op.Login.RoleID)
			op.Data.Data = rd.Data
		}
		postOp(op)
	}
}

func newRoleInDB(roleID uint64) (*role.DataToSave, error) {
	rData := pb.RoleData{
		ID:    roleID,
		Name:  util.ToString(roleID),
		Level: 1,
	}

	rd := &role.DataToSave{
		ID:   roleID,
		Data: make(map[string]string),
	}

	str, err := jsoniter.MarshalToString(&rData)
	if err != nil {
		logger.Errorf("[login] marshal role data err:%v", err)
		return nil, err
	}
	rd.Set(pb.TypeComp_TCBase, str)
	return rd, nil
}
