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
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
)

type AccountLoader struct {
	loading chan *pb.S2SReqLogin
}

func newAccountLoader() *AccountLoader {
	return &AccountLoader{
		loading: make(chan *pb.S2SReqLogin, 4096),
	}
}

func (l *AccountLoader) push(op *pb.S2SReqLogin) {
	l.loading <- op
}

func (l *AccountLoader) run(ctx context.Context) {
	const (
		batchSize     = 500
		flushInterval = 50 * time.Millisecond
	)

	batch := make([]*pb.S2SReqLogin, 0, batchSize)
	uniqueRequests := make([]*pb.S2SReqLogin, 0, batchSize)
	t := time.NewTicker(flushInterval)
	defer func() {
		t.Stop()
	}()

	flush := func() {
		if len(batch) > 0 {
			seen := make(map[string]bool)

			for _, req := range batch {
				cacheKey := model.KeyAccBind(req.Req.SdkType, req.Req.Account)
				if seen[cacheKey] {
					sendLoginFailure(req, pb.LoginCode_LCServerBusy)
					continue
				}
				seen[cacheKey] = true
				uniqueRequests = append(uniqueRequests, req)
			}
			l.loadAccountsFromCache(uniqueRequests)
			batch = batch[:0]
			uniqueRequests = uniqueRequests[:0]
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

func (l *AccountLoader) loadAccountsFromCache(batch []*pb.S2SReqLogin) {
	// ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	// defer cancel()
	ctx := context.Background()

	pipeBind := db.Redis.Pipeline()
	for _, op := range batch {
		// zap.L().Debug("loadAccountsFromCache", zap.String("op", op.String()))
		pipeBind.Get(ctx, model.KeyAccBind(op.Req.SdkType, op.Req.Account))
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
					dispatchEvent(Event{
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
		l.loadAccountsFromDB(ctx, batchFromDB)
	}
}

func (l *AccountLoader) loadAccountsFromDB(ctx context.Context, all []*pb.S2SReqLogin) {
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
			filter = bson.M{acc.FieldGoogleID(): bson.M{"$in": bt.accs}}
		case pb.SdkType_Facebook:
			filter = bson.M{acc.FieldFBID(): bson.M{"$in": bt.accs}}
		default:
		}
		l.queryAccountsBySDKType(ctx, filter, bt.batch, k)
	}
}

func (l *AccountLoader) queryAccountsBySDKType(ctx context.Context, filter bson.M, batch []*pb.S2SReqLogin, typ pb.SdkType) {
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

	// 可能redis中bind过期，acc状态还在，需要状态回填
	if len(accDatas) > 0 {
		pipe := db.Redis.Pipeline()
		cmds := make([]*redis.SliceCmd, len(accDatas))
		for i, acc := range accDatas {
			cmds[i] = pipe.HMGet(ctx, model.KeyAccount(acc.AccID), AccFields()...)
		}

		if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
			zap.L().Warn("enrich account from redis failed", zap.Error(err))
		} else {
			for i, acc := range accDatas {
				if cmds[i].Err() == nil {
					_ = cmds[i].Scan(acc)
				}
			}
		}
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
			dispatchEvent(Event{
				Op:    OpAfterSDKCheck,
				Login: op,
				Acc:   r,
			})
			updateAccBatch = append(updateAccBatch, accWrap{AccData: r, Account: op.Req.Account}) // 延后了点，需要mgr保证不重进
		} else {
			newAccBatch = append(newAccBatch, op)
		}
	}

	if len(newAccBatch) > 0 {
		l.registerNewAccounts(ctx, newAccBatch)
	}
	if len(updateAccBatch) > 0 {
		l.syncAccountsToCache(ctx, updateAccBatch)
	}
}

type accWrap struct {
	Account string
	AccData *Account
}

func (l *AccountLoader) syncAccountsToCache(ctx context.Context, batch []accWrap) {
	const expiration = time.Hour * 24 * 7
	pipe := db.Redis.Pipeline()
	for _, b := range batch {
		keyAcc := model.KeyAccount(b.AccData.AccID)
		pipe.HSet(ctx, keyAcc, b.AccData.FieldAccID(), b.AccData.AccID,
			"freeze", b.AccData.Freeze,
			b.AccData.FieldDevice(), b.AccData.Device,
			b.AccData.FieldAppleID(), b.AccData.AppleID,
			b.AccData.FieldGoogleID(), b.AccData.GoogleID,
			b.AccData.FieldFBID(), b.AccData.FbID)
		pipe.Expire(ctx, keyAcc, expiration)

		// 绑定信息
		if len(b.AccData.Device) > 0 {
			keyBind := model.KeyAccBind(pb.SdkType_Guest, b.AccData.Device)
			pipe.Set(ctx, keyBind, b.AccData.AccID, expiration)
		}
		if len(b.AccData.AppleID) > 0 {
			keyBind := model.KeyAccBind(pb.SdkType_Apple, b.AccData.AppleID)
			pipe.Set(ctx, keyBind, b.AccData.AccID, expiration)
		}
		if len(b.AccData.GoogleID) > 0 {
			keyBind := model.KeyAccBind(pb.SdkType_Google, b.AccData.GoogleID)
			pipe.Set(ctx, keyBind, b.AccData.AccID, expiration)
		}
		if len(b.AccData.FbID) > 0 {
			keyBind := model.KeyAccBind(pb.SdkType_Facebook, b.AccData.FbID)
			pipe.Set(ctx, keyBind, b.AccData.AccID, expiration)
		}
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		zap.L().Error("redis hset acc_id failed", zap.Error(err))
		return
	}
}

func (l *AccountLoader) registerNewAccounts(ctx context.Context, batch []*pb.S2SReqLogin) {
	pipe := db.Redis.Pipeline()
	const expiration = time.Hour * 24 * 7

	for _, req := range batch {
		id, err := GenerateNextAccID(ctx)
		if err != nil {
			zap.L().Error("generate acc id failed", zap.Error(err))
			sendLoginFailure(req, pb.LoginCode_LCServerErr)
			continue
		}

		acc := &Account{
			AccID:  id,
			Device: req.Req.Dev,
		}

		switch req.Req.SdkType {
		case pb.SdkType_Apple:
			acc.AppleID = req.Req.Account
		case pb.SdkType_Google:
			acc.GoogleID = req.Req.Account
		case pb.SdkType_Facebook:
			acc.FbID = req.Req.Account
		default:
			acc.Device = req.Req.Account
		}
		zap.L().Debug("db insert", zap.Any("acc", acc))
		_, err = db.MongoDB().Collection(AccountCollection).InsertOne(ctx, acc)
		if err != nil {
			// 如果是唯一索引冲突（说明其他节点刚好注册了这个账号）
			if mongo.IsDuplicateKeyError(err) {
				zap.L().Warn("concurrent register detected, ignore", zap.String("acc", req.Req.Account))
				sendLoginFailure(req, pb.LoginCode_LCServerBusy)
				continue
			}
			zap.L().Error("[login] insert account failed", zap.Error(err))
			sendLoginFailure(req, pb.LoginCode_LCServerErr)
			continue
		}

		keyAcc := model.KeyAccount(acc.AccID)
		keyBind := model.KeyAccBind(req.Req.SdkType, req.Req.Account)

		pipe.HSet(ctx, keyAcc, acc.FieldAccID(), acc.AccID)
		if acc.AppleID != "" {
			pipe.HSet(ctx, keyAcc, acc.FieldAppleID(), acc.AppleID)
		}
		if acc.GoogleID != "" {
			pipe.HSet(ctx, keyAcc, acc.FieldGoogleID(), acc.GoogleID)
		}
		if acc.FbID != "" {
			pipe.HSet(ctx, keyAcc, acc.FieldFBID(), acc.FbID)
		}
		if acc.Device != "" {
			pipe.HSet(ctx, keyAcc, acc.FieldDevice(), acc.Device)
		}

		pipe.Expire(ctx, keyAcc, expiration)
		pipe.Set(ctx, keyBind, acc.AccID, expiration)

		dispatchEvent(Event{
			Op:    OpAfterSDKCheck,
			Login: req,
			Acc:   acc,
		})
	}

	if _, err := pipe.Exec(ctx); err != nil {
		zap.L().Warn("redis batch set failed, will fallback to mongo next time", zap.Error(err))
	}
}
