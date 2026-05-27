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
				cacheKey := FormatBindKey(req.Req.SdkType, req.Req.Account)
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
		pipeBind.Get(ctx, model.KeyAccBind(FormatBindKey(op.Req.SdkType, op.Req.Account)))
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
					acc.UnmarshalBinds()
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

type accWrap struct {
	BindKey string
	AccData *Account
}

func (l *AccountLoader) loadAccountsFromDB(ctx context.Context, batch []*pb.S2SReqLogin) {
	bindKeys := make([]string, 0, len(batch))
	for _, op := range batch {
		bindKey := FormatBindKey(op.Req.SdkType, op.Req.Account)
		bindKeys = append(bindKeys, bindKey)
	}
	filter := bson.M{"binds": bson.M{"$in": bindKeys}}

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
			cmds[i] = pipe.HMGet(ctx, model.KeyAccount(acc.AccID), StateFields()...)
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

	// 建立反向映射表，方便快速匹配
	result := make(map[string]*Account)
	for _, acc := range accDatas {
		// 遍历该账号绑定的所有渠道
		for _, bind := range acc.Binds {
			result[bind] = acc
		}
	}

	newAccBatch := make([]*pb.S2SReqLogin, 0, len(batch))
	updateAccBatch := make([]accWrap, 0, len(batch))
	for _, op := range batch {
		bindKey := FormatBindKey(op.Req.SdkType, op.Req.Account)
		if r, ok := result[bindKey]; ok {
			dispatchEvent(Event{Op: OpAfterSDKCheck, Login: op, Acc: r})
			updateAccBatch = append(updateAccBatch, accWrap{AccData: r, BindKey: bindKey})
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

func (l *AccountLoader) syncAccountsToCache(ctx context.Context, batch []accWrap) {
	const expiration = time.Hour * 24 * 7
	pipe := db.Redis.Pipeline()
	for _, b := range batch {
		keyAcc := model.KeyAccount(b.AccData.AccID)
		pipe.HSet(ctx, keyAcc, b.AccData.FieldAccID(), b.AccData.AccID,
			b.AccData.FieldFreeze(), b.AccData.Freeze,
			b.AccData.FieldBinds(), b.AccData.MarshalBinds())
		pipe.Expire(ctx, keyAcc, expiration)

		pipe.Set(ctx, model.KeyAccBind(b.BindKey), b.AccData.AccID, expiration)
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

		bindKey := FormatBindKey(req.Req.SdkType, req.Req.Account)
		acc := &Account{
			AccID: id,
			Binds: []string{bindKey},
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
		keyBind := model.KeyAccBind(FormatBindKey(req.Req.SdkType, req.Req.Account))

		pipe.HSet(ctx, keyAcc, acc.FieldAccID(), acc.AccID, acc.FieldBinds(), acc.MarshalBinds())
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
