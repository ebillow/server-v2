package login_mgr

import (
	"context"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"
	"server/pkg/db"
	"server/pkg/model"
	"server/pkg/queue"
	"sync"
	"time"
)

type opSaveData struct {
	ID   uint64
	Data map[string]string
	Op   uint32
}

func (d *opSaveData) Values() []interface{} {
	ret := make([]interface{}, 0, len(d.Data))
	for k, v := range d.Data {
		ret = append(ret, k, v)
	}
	return ret
}

type saver struct {
	save *queue.SwapQueue[opSaveData]
	ctrl chan struct{}
}

func newSaver() *saver {
	return &saver{
		save: queue.NewSwapQueue[opSaveData](OpChanSize, OpChanSize*100),
		ctrl: make(chan struct{}),
	}
}

func (s *saver) close() {
	close(s.ctrl)
}

func (s *saver) post(op opSaveData) {
	if err := s.save.Push(op); err != nil {
		zap.L().Error("save chan full", zap.Uint64("id", op.ID))
	}
}

func (s *saver) run(wait *sync.WaitGroup) {
	const (
		batchSize     = 500
		flushInterval = time.Second
	)
	batch := make(map[uint64]opSaveData, batchSize)
	ticker := time.NewTicker(flushInterval)
	defer func() {
		wait.Done()
		ticker.Stop()
	}()

	flush := func() {
		if len(batch) > 0 {
			err := s.saveBatch(batch)
			if err == nil {
				batch = make(map[uint64]opSaveData, batchSize)
			}
		}
	}

	for {
		select {
		case <-s.save.Sig():
			s.save.Range(func(op opSaveData) bool {
				batch[op.ID] = op
				if len(batch) >= batchSize {
					flush()
					ticker.Reset(flushInterval)
				}
				return true
			})

		case <-ticker.C:
			flush()
		case <-s.ctrl:
			s.save.Range(func(op opSaveData) bool {
				batch[op.ID] = op
				if len(batch) >= batchSize {
					flush()
				}
				return true
			})
			flush()
			return
		}
	}
}

func (s *saver) saveBatch(batch map[uint64]opSaveData) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	pipe := db.Redis.Pipeline()
	toDB := make([]opSaveData, 0, len(batch))
	for _, v := range batch {
		pipe.HSet(ctx, model.KeyRole(v.ID), v.Values()...)
		pipe.Expire(ctx, model.KeyRole(v.ID), time.Hour*24*7)
		zap.L().Debug("[login] save to redis", zap.Uint64("id", v.ID), zap.Any("data", v))
		if v.Op == OpOffline {
			toDB = append(toDB, v)
		}
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		zap.S().Errorf("[login] real save role err:%v", err)
		return err
	}

	return s.saveToDB(ctx, toDB)
}

func (s *saver) saveToDB(ctx context.Context, toDB []opSaveData) error {
	models := make([]mongo.WriteModel, 0, len(toDB))
	for i := range toDB {
		doc := bson.D{}
		for k, v := range toDB[i].Data {
			doc = append(doc, bson.E{Key: "data." + k, Value: v})
		}
		if len(doc) == 0 {
			continue
		}
		update := bson.D{}
		update = append(update, bson.E{Key: "$set", Value: doc})

		mod := mongo.NewUpdateOneModel()
		mod.SetFilter(bson.M{"id": toDB[i].ID})
		mod.SetUpsert(true)
		mod.SetUpdate(update)

		models = append(models, mod)
		zap.S().Debugf("[login] bulk write save role %d to acc_db", toDB[i].ID)
	}

	if len(models) == 0 {
		return nil
	}

	opts := options.BulkWrite().SetOrdered(false) // 到这tolDB里没有重复的
	_, err := db.MongoDB().Collection("roles").BulkWrite(ctx, models, opts)
	if err != nil {
		zap.S().Errorf("[login] bulk write save role err:%v", err)
		return err
	}

	op := &Operator{Op: OpSaveSuccess}
	for i := range toDB {
		op.IDs = append(op.IDs, toDB[i].ID)
	}
	postOp(op)
	return nil
}
