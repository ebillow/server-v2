package logon_service

import (
	"context"
	"server/internal/share/model"
	"server/pkg/db"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"
)

type opSaveData struct {
	ID   uint64
	Data map[string]string
	Both bool
}

func (d *opSaveData) Values() []interface{} {
	ret := make([]interface{}, 0, len(d.Data))
	for k, v := range d.Data {
		ret = append(ret, k, v)
	}
	return ret
}

type saver struct {
	save chan opSaveData
}

func newSaver() *saver {
	return &saver{
		save: make(chan opSaveData, OpChanSize),
	}
}

func (s *saver) close() {
	close(s.save)
}

func (s *saver) post(op opSaveData) { // 这里不能丢数据,写不进去就背压
	s.save <- op
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

func (s *saver) saveBatch(batch map[uint64]opSaveData) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	pipe := db.Redis.Pipeline()
	toDB := make([]opSaveData, 0, len(batch))
	for _, v := range batch {
		pipe.HSet(ctx, model.KeyRole(v.ID), v.Values()...)
		pipe.Expire(ctx, model.KeyRole(v.ID), time.Hour*24*7)
		zap.L().Debug("[login] save to redis", zap.Uint64("id", v.ID), zap.Any("data", v))
		if v.Both {
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
