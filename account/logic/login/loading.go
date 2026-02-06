package login

import (
	"context"
	"errors"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/zap"
	"server/account/acc_db"
	"server/pkg/db"
	"server/pkg/model"
	"server/pkg/pb"
	"time"
)

type loader struct {
	loading chan *pb.S2SReqLogin
}

func newLoader() *loader {
	return &loader{
		loading: make(chan *pb.S2SReqLogin, 4096),
	}
}

func (l *loader) post(op *pb.S2SReqLogin) {
	l.loading <- op
}

func (l *loader) run(ctx context.Context) {
	const (
		batchSize     = 100
		flushInterval = 50 * time.Millisecond
	)

	batch := make([]*pb.S2SReqLogin, 0, batchSize)
	t := time.NewTicker(flushInterval)
	defer func() {
		t.Stop()
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

func (l *loader) loadBatch(batch []*pb.S2SReqLogin) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	pipe := db.Redis.Pipeline()
	for _, op := range batch {
		pipe.HGetAll(ctx, model.KeyAccount(op.Req.Account))
	}
	cmd, err := pipe.Exec(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		zap.L().Error("[login] redis load batch failed", zap.Error(err))
		return
	}

	batchFromDB := make([]*pb.S2SReqLogin, 0, len(cmd))
	for i, c := range cmd {
		data := c.(*redis.MapStringStringCmd)
		if /*c.Err() == nil*/ len(data.Val()) > 0 { // 加载成功
			acc := &Account{}
			err = data.Scan(acc)
			if err == nil {
				op := batch[i]
				afterCheck(acc, op)
			}
		} else /*if errors.Is(c.Err(), redis.Nil)*/ { // redis里没有
			batchFromDB = append(batchFromDB, batch[i])
		}
	}

	if len(batchFromDB) > 0 {
		l.loadFromDBBatch(ctx, batchFromDB)
	}
}

func (l *loader) loadFromDBBatch(ctx context.Context, batch []*pb.S2SReqLogin) {
	accs := make([]string, 0, len(batch))
	for _, op := range batch {
		accs = append(accs, op.Req.Account)
	}

	filter := bson.M{"account": bson.M{"$in": accs}}
	cursor, err := db.MongoDB.Collection(acc_db.AccountTable).Find(ctx, filter)
	if err != nil {
		zap.L().Error("[login] find role failed", zap.Error(err))
		return
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var accDatas []*Account
	if err = cursor.All(ctx, &accDatas); err != nil {
		zap.L().Error("[login] cursor all failed", zap.Error(err))
		return
	}
	result := make(map[string]*Account, len(accDatas))
	for _, acc := range accDatas {
		result[acc.Account] = acc
	}

	for _, op := range batch {
		if r, ok := result[op.Req.Account]; ok {
			r.Update(ctx, op)
			afterCheck(r, op)
		} else {
			acc, err := newAccount(ctx, op)
			if err != nil {
				// todo 发失败消息给前端？？
				return
			}
			afterCheck(acc, op)
		}
	}
}
