package login_mgr

import (
	"context"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
	"server/pkg/db"
	"server/pkg/model"
	"sync"
	"time"
)

type opSaveData struct {
	ID   uint64
	Data map[string]string
	Op   uint32
}

func (d *opSaveData) Values() []string {
	ret := make([]string, 0, len(d.Data))
	for k, v := range d.Data {
		ret = append(ret, k, v)
	}
	return ret
}

type saver struct {
	save chan *opSaveData
	once sync.Once
}

func newSaver() *saver {
	return &saver{
		save: make(chan *opSaveData, OpChanSize),
	}
}

func (s *saver) post(op *opSaveData) {
	s.save <- op // 反压
	// select {
	// case s.save <- op:
	// default:
	// 	zap.L().Error("save chan full", zap.Uint64("id", op.ID))
	// }
}

func (s *saver) close() {
	s.once.Do(func() {
		close(s.save)
	})
}

func (s *saver) run(wait *sync.WaitGroup) {
	const (
		batchSize     = 500
		flushInterval = time.Second
	)
	batch := make(map[uint64]*opSaveData, batchSize)
	ticker := time.NewTicker(flushInterval)
	defer func() {
		wait.Done()
		ticker.Stop()
	}()

	flush := func() {
		if len(batch) > 0 {
			s.saveBatch(batch)
			batch = make(map[uint64]*opSaveData, batchSize)
		}
	}

	for {
		select {
		case op, ok := <-s.save:
			if !ok {
				flush()
				return
			}
			batch[op.ID] = op
			if len(batch) >= batchSize {
				flush()
				ticker.Reset(flushInterval)
			}
		case <-ticker.C:
			flush()
		}
	}
}

func (s *saver) saveBatch(batch map[uint64]*opSaveData) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	pipe := db.Redis.Pipeline()
	toDB := make([]*opSaveData, 0, len(batch))
	for _, v := range batch {
		pipe.HSet(ctx, model.KeyRole(v.ID), v.Values())
		pipe.Expire(ctx, model.KeyRole(v.ID), time.Hour*24*7)
		zap.L().Debug("[login] save to redis", zap.Uint64("id", v.ID))
		if v.Op == OpOffline {
			toDB = append(toDB, v)
		}
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		zap.S().Errorf("[login] real save role err:%v", err)
		return
	}

	s.saveToDB(ctx, toDB)
}

func (s *saver) saveToDB(ctx context.Context, toDB []*opSaveData) {
	models := make([]mongo.WriteModel, 0, len(toDB))
	for i := range toDB {
		mod := mongo.NewUpdateOneModel()
		mod.SetFilter(bson.M{"id": toDB[i].ID})
		mod.SetUpsert(true)
		mod.SetUpdate(bson.D{{"$set", bson.D{
			{"data", toDB[i].Data},
		}}})
		models = append(models, mod)
		zap.S().Debugf("[login] bulk write save role %d to acc_db", toDB[i].ID)
	}

	_, err := db.MongoDB.Collection("roles").BulkWrite(ctx, models)
	if err != nil {
		zap.S().Errorf("[login] bulk write save role err:%v", err)
		return
	}

	op := &Operator{Op: OpSaveSuccess}
	for i := range toDB {
		op.IDs = append(op.IDs, toDB[i].ID)
	}
	postOp(op)
}
