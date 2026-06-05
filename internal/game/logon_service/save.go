package logon_service

import (
	"context"
	"errors"
	"server/internal/share/model"
	"server/pkg/db"
	"server/pkg/queue"
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
	q       *queue.SwapQueue[opSaveData]
	mgr     *LogonService
	closeCh chan struct{}
}

func newSaver(mgr *LogonService) *saver {
	return &saver{
		q:       queue.NewSwapQueue[opSaveData](4096, opChanSize),
		mgr:     mgr,
		closeCh: make(chan struct{}),
	}
}

func (s *saver) close() {
	s.q.Close()
	close(s.closeCh)
}

func (s *saver) post(op opSaveData) {
	err := s.q.Push(op)
	if err != nil {
		zap.L().Warn("logon_service.save", zap.Error(err))
	}
}

// ：关键数据（Logout）走阻塞路径保证不丢
func (s *saver) postMustSave(op opSaveData) {
	for {
		err := s.q.Push(op)
		if err == nil {
			return
		}
		if errors.Is(err, queue.ErrQueueClosed) {
			zap.L().Error("saver closed, data lost", zap.Uint64("id", op.ID))
			return
		}
		// 队列满，短暂等待后重试
		time.Sleep(10 * time.Millisecond)
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

	var backoff time.Duration
	var nextFlush time.Time

	flush := func() {
		if len(batch) > 0 {
			if time.Now().Before(nextFlush) {
				return
			}
			err := s.saveBatch(batch)
			if err == nil {
				batch = make(map[uint64]opSaveData, batchSize)
				backoff = 0
			} else {
				backoff = min(backoff*2+100*time.Millisecond, 5*time.Second)
				nextFlush = time.Now().Add(backoff)
				zap.S().Warnf("[login] save batch failed, retrying after %v, err: %v", backoff, err)
			}
		}
	}

	for {
		select {
		case <-s.q.Sig():
			s.q.Range(func(data opSaveData) {
				merge(batch, data)
			})
			if len(batch) >= batchSize {
				flush()
				ticker.Reset(flushInterval)
			}

		case <-ticker.C:
			flush()

		case <-s.closeCh:
			s.q.Range(func(data opSaveData) {
				merge(batch, data)
			})
			for len(batch) > 0 {
				flush()

				if len(batch) > 0 {
					time.Sleep(100 * time.Millisecond)
				}
			}
			return
		}
	}
}

func merge(batch map[uint64]opSaveData, data opSaveData) {
	if old, ok := batch[data.ID]; ok {
		if data.Both {
			old.Both = true
		}
		// 合并 Data map（新数据覆盖旧字段）
		for k, v := range data.Data {
			old.Data[k] = v
		}
		batch[data.ID] = old
	} else {
		batch[data.ID] = data
	}
}
func (s *saver) saveBatch(batch map[uint64]opSaveData) error {
	const chunkSize = 1000

	allData := make([]opSaveData, 0, len(batch))
	for _, v := range batch {
		allData = append(allData, v)
	}

	for i := 0; i < len(allData); i += chunkSize {
		end := i + chunkSize
		if end > len(allData) {
			end = len(allData)
		}
		chunk := allData[i:end]

		if err := s.saveChunk(chunk); err != nil {
			return err // 只要有一个块失败，就返回 error 触发整体重试
		}
	}
	return nil
}

func (s *saver) saveChunk(batch []opSaveData) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	pipe := db.Redis.Pipeline()
	toDB := make([]opSaveData, 0, len(batch))
	for _, v := range batch {
		pipe.HSet(ctx, model.KeyRole(v.ID), v.Values()...)
		pipe.Expire(ctx, model.KeyRole(v.ID), time.Hour*24*7)
		// zap.L().Debug("[login] save to redis", zap.Uint64("id", v.ID), zap.Any("data", v))
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

	opts := options.BulkWrite().SetOrdered(false)
	_, err := db.MongoDB().Collection("roles").BulkWrite(ctx, models, opts)
	if err != nil {
		zap.S().Errorf("[login] bulk write save role err:%v", err)
		return err
	}

	evt := &EvtSaveSuccess{
		IDs: make([]uint64, 0, len(toDB)),
	}
	for i := range toDB {
		evt.IDs = append(evt.IDs, toDB[i].ID)
	}
	s.mgr.postEvent(evt)

	return nil
}
