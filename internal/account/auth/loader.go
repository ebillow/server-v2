package auth

import (
	"context"
	"errors"
	pb "server/api/pb"
	"server/internal/share/model"
	"server/pkg/db"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/zap"
)

type AccountLoader struct {
	loading chan *pb.S2SReqLogin
}

func newLoader() *AccountLoader {
	return &AccountLoader{
		loading: make(chan *pb.S2SReqLogin, 4096),
	}
}

func (l *AccountLoader) push(op *pb.S2SReqLogin) {
	zap.L().Debug("start load", zap.Any("op", op))
	l.loading <- op
}

func (l *AccountLoader) run(ctx context.Context) {
	const (
		batchSize     = 500
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

func (l *AccountLoader) loadBatch(batch []*pb.S2SReqLogin) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	// ctx := context.Background()

	pipeBind := db.Redis.Pipeline()
	for _, op := range batch {
		pipeBind.Get(ctx, model.KeyAccBind(op.Req.Account))
	}
	cmdBind, err := pipeBind.Exec(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		zap.L().Error("[login] redis load batch failed", zap.Error(err))
		return
	}

	pipeAcc := db.Redis.Pipeline()
	batchFromDB := make([]*pb.S2SReqLogin, 0, len(cmdBind))
	cmdAcc := make(map[int]*redis.SliceCmd)
	for i, c := range cmdBind {
		if c.Err() == nil {
			accID, err := c.(*redis.StringCmd).Uint64()
			if err == nil {
				cmdAcc[i] = pipeAcc.HMGet(ctx, model.KeyAccount(accID), AccFields()...)
			} else {
				batchFromDB = append(batchFromDB, batch[i])
			}
		} else {
			batchFromDB = append(batchFromDB, batch[i])
		}
	}

	if len(cmdAcc) > 0 {
		_, err = pipeAcc.Exec(ctx)
		if err != nil && !errors.Is(err, redis.Nil) {
			zap.L().Error("[login] redis load batch pipeAcc failed", zap.Error(err))
			return
		}
		for i, c := range cmdAcc {
			if c.Err() == nil {
				acc := &Account{}
				err = c.Scan(acc)
				if err == nil && acc.AccID > 0 {
					PostEvt(Event{
						Op:    OpAfterSDKCheck,
						Login: batch[i],
						Acc:   acc,
					})
				} else {
					batchFromDB = append(batchFromDB, batch[i])
				}
			} else {
				batchFromDB = append(batchFromDB, batch[i])
			}
		}
	}

	if len(batchFromDB) > 0 {
		l.loadFromDBBatch(ctx, batchFromDB)
	}
}

func (l *AccountLoader) loadFromDBBatch(ctx context.Context, all []*pb.S2SReqLogin) {
	type Tmp struct {
		accs  []string
		batch []*pb.S2SReqLogin
	}
	batch := make(map[pb.SdkType]*Tmp)
	for _, op := range all {
		one, ok := batch[op.Req.SdkType]
		if !ok {
			one = &Tmp{}
			batch[op.Req.SdkType] = one
		}
		one.accs = append(one.accs, op.Req.Account)
		one.batch = append(one.batch, op)
	}
	acc := &Account{}
	for k, bt := range batch {
		filter := bson.M{acc.FieldDevice(): bson.M{"$in": bt.accs}}
		switch k {
		case pb.SdkType_Apple:
			filter = bson.M{acc.FieldAppleID(): bson.M{"$in": bt.accs}}
		case pb.SdkType_Google:
			filter = bson.M{acc.FieldAppleID(): bson.M{"$in": bt.accs}}
		case pb.SdkType_Facebook:
			filter = bson.M{acc.FieldFBID(): bson.M{"$in": bt.accs}}
		default:
		}
		l.loadOneKindAccFromDB(ctx, filter, bt.batch, k)
	}
}

func (l *AccountLoader) loadOneKindAccFromDB(ctx context.Context, filter bson.M, batch []*pb.S2SReqLogin, typ pb.SdkType) {
	cursor, err := db.MongoDB().Collection(AccountCollection).Find(ctx, filter)
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
		switch typ {
		case pb.SdkType_Apple:
			result[acc.AppleID] = acc
		case pb.SdkType_Google:
			result[acc.GoogleID] = acc
		case pb.SdkType_Facebook:
			result[acc.FbID] = acc
		default:
			result[acc.Device] = acc
		}
	}

	newAccBatch := make([]*pb.S2SReqLogin, 0, len(batch))
	updateAccBatch := make([]accWrap, 0, len(batch))
	for _, op := range batch {
		if r, ok := result[op.Req.Account]; ok {
			PostEvt(Event{
				Op:    OpAfterSDKCheck,
				Login: op,
				Acc:   r,
			})
			updateAccBatch = append(updateAccBatch, accWrap{Acc: r, Account: op.Req.Account}) // 延后了点，需要mgr保证不重进
		} else {
			newAccBatch = append(newAccBatch, op)
		}
	}

	if len(newAccBatch) > 0 {
		l.newAccountBatch(ctx, newAccBatch)
	}
	if len(updateAccBatch) > 0 {
		l.updateBatch(ctx, updateAccBatch)
	}
}

type accWrap struct {
	Account string
	Acc     *Account
}

func (l *AccountLoader) updateBatch(ctx context.Context, batch []accWrap) {
	const expiration = time.Hour * 24 * 7
	pipe := db.Redis.Pipeline()
	for _, b := range batch {
		keyAcc := model.KeyAccount(b.Acc.AccID)
		pipe.HSet(ctx, keyAcc, b.Acc.FieldAccID(), b.Acc.AccID,
			"freeze", b.Acc.Freeze,
			b.Acc.FieldDevice(), b.Acc.Device,
			b.Acc.FieldAppleID(), b.Acc.AppleID,
			b.Acc.FieldGoogleID(), b.Acc.GoogleID,
			b.Acc.FieldFBID(), b.Acc.FbID)
		pipe.Expire(ctx, keyAcc, expiration)
		keyBind := model.KeyAccBind(b.Account)
		pipe.Set(ctx, keyBind, b.Acc.AccID, expiration)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		zap.L().Error("redis hset acc_id failed", zap.Error(err))
		return
	}
}

func (l *AccountLoader) newAccountBatch(ctx context.Context, batch []*pb.S2SReqLogin) {
	accBat := make([]*Account, 0, len(batch))
	pipe := db.Redis.Pipeline()
	const expiration = time.Hour * 24 * 7

	for _, req := range batch {
		id := curAccID.Add(1)

		acc := &Account{
			AccID:  id,
			Device: req.Req.Dev,
		}

		keyAcc := model.KeyAccount(acc.AccID)

		switch req.Req.SdkType {
		case pb.SdkType_Apple:
			acc.AppleID = req.Req.Account
			pipe.HSet(ctx, keyAcc, acc.FieldAccID(), acc.AccID, acc.FieldAppleID(), acc.AppleID)
		case pb.SdkType_Google:
			acc.GoogleID = req.Req.Account
			pipe.HSet(ctx, keyAcc, acc.FieldAccID(), acc.AccID, acc.FieldGoogleID(), acc.GoogleID)
		case pb.SdkType_Facebook:
			acc.FbID = req.Req.Account
			pipe.HSet(ctx, keyAcc, acc.FieldAccID(), acc.AccID, acc.FieldFBID(), acc.FbID)
		default:
			acc.Device = req.Req.Account
			pipe.HSet(ctx, keyAcc, acc.FieldAccID(), acc.AccID, acc.FieldDevice(), acc.Device)
		}
		accBat = append(accBat, acc)

		pipe.Expire(ctx, keyAcc, expiration)
		keyBind := model.KeyAccBind(req.Req.Account)
		pipe.Set(ctx, keyBind, acc.AccID, expiration)
	}

	_, err := db.MongoDB().Collection(AccountCollection).InsertMany(ctx, accBat)
	if err != nil {
		zap.L().Error("[login] insert account failed", zap.Error(err))
		return
	}

	// 写redis
	_, err = pipe.Exec(ctx)
	if err != nil {
		zap.L().Warn("redis hset acc_id failed", zap.Error(err))
		// mongo已经成功，下次会从mongo加载
	}
	for i := range accBat {
		PostEvt(Event{
			Op:    OpAfterSDKCheck,
			Login: batch[i],
			Acc:   accBat[i],
		})
	}
}
