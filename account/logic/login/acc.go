package login

import (
	"context"
	"errors"
	"go.uber.org/zap"
	"server/account/acc_db"
	"server/pkg/db"
	"server/pkg/model"
	"server/pkg/pb"
	"time"
)

var (
	errSelfIsConnected = errors.New("acc is connected")
	errTargetConnected = errors.New("target acc already connect")
	errAccIDErr        = errors.New("target accid is err")
)

type Account struct {
	Account string `redis:"account"`
	AccID   uint64 `redis:"acc_id"`
	Freeze  bool   `redis:"freeze"`
	GameID  int32  `redis:"game_id"`
	Time    int64  `redis:"time"`
	Seq     uint32 `redis:"seq"`
	Passwd  uint64 `redis:"passwd"`
}

func newAccount(ctx context.Context, req *pb.S2SReqLogin) (*Account, error) {
	id := db.Redis.HIncrBy(ctx, model.RedisKeyIDs, "acc_id", 1).Val()
	acc := &Account{
		Account: req.Req.Account,
		AccID:   uint64(id),
	}

	_, err := db.MongoDB.Collection(acc_db.AccountTable).InsertOne(ctx, acc)
	if err != nil {
		zap.L().Error("mongo insert acc failed", zap.Error(err))
		return nil, err
	}

	pipe := db.Redis.Pipeline()
	key := model.KeyAccount(req.Req.Account)
	pipe.HSet(ctx, key, "account", acc.Account, "acc_id", acc.AccID)
	pipe.Expire(ctx, key, time.Hour*24*7)

	_, err = pipe.Exec(ctx)
	if err != nil {
		zap.L().Error("redis hset acc_id failed", zap.Error(err))
		return nil, err
	}

	return acc, nil
}

func (acc *Account) Update(ctx context.Context, req *pb.S2SReqLogin) {
	pipe := db.Redis.Pipeline()
	key := model.KeyAccount(req.Req.Account)
	pipe.HSet(ctx, key, "account", acc.Account, "acc_id", acc.AccID)
	pipe.Expire(ctx, key, time.Hour*24*7)

	_, err := pipe.Exec(ctx)
	if err != nil {
		zap.L().Error("redis hset acc_id failed", zap.Error(err))
		return
	}
}

func (acc *Account) SaveLoginData(ctx context.Context) error {
	return db.Redis.HSet(ctx, model.KeyAccount(acc.Account), "game_id", acc.GameID, "time", acc.Time, "seq", acc.Seq, "passwd", acc.Passwd).Err()
}

func (acc *Account) LoadSeq(ctx context.Context) int32 {
	v, err := db.Redis.HGet(ctx, model.KeyAccount(acc.Account), "seq").Int()
	if err != nil {
		return 0
	}
	return int32(v)
}

//	func ConnectAcc(msg *pb2.MsgBindAcc, s *snet.Session) {
//		rawAccToBind := share.GetRawAcc(msg.SdkToBind, 0, msg.AccToBind)
//
//		ctx, cancel := context.WithTimeout(context.Background(), time.Second*300)
//		defer cancel()
//		data, err := connectAcc(ctx, rawAccToBind, msg.GuestAccID)
//		if err != nil {
//			if errors.Is(err, errSelfIsConnected) {
//				msg.Ret = pb2.MsgBindAcc_AccIsBind
//			} else if errors.Is(err, errTargetConnected) {
//				msg.Ret = pb2.MsgBindAcc_GuestIsBind
//			} else {
//				msg.Ret = pb2.MsgBindAcc_GuestNotFind
//			}
//			logger.Warnf("connect acc err:%v", err)
//		} else { // 成功
//			msg.Ret = pb2.MsgBindAcc_BindSuccess
//			// 用户账号绑定成功
//			logMsg := &pb2.MsgDBLog{TaLogs: []*pb2.TaLog{{
//				EventName:  "AccountBind",
//				Account:    msg.RoleGuid,
//				DistinctId: "",
//				Properties: component.BsonMarshal(map[string]interface{}{
//					"ia_add": true,
//					"acc":    rawAccToBind,
//					"accID":  msg.GuestAccID,
//				}),
//			}}}
//			network.SendToRandLog(pb.MsgIDS2S_S2SLog, logMsg)
//		}
//		msg.AccData = data
//		s.Send(pb.MsgIDS2S_Acc2GmBindAcc, msg)
//	}
// func connectAcc(ctx context.Context, acc string, accID uint64) (data *pb.AccInfo, err error) {
// 	acc_db.GetPrimaryDB().SyncExe(func(cli *mongo.Database) {
// 		err = cli.Client().UseSession(ctx, func(sessionContext mongo.SessionContext) error {
// 			if err = sessionContext.StartTransaction(); err != nil {
// 				return err
// 			}
//
// 			table := cli.Collection("acc")
// 			data = &pb2.AccInfo{}
// 			err = table.FindOne(sessionContext, bson.M{"acc": acc}).Decode(data)
// 			if err == nil {
// 				return errSelfIsConnected
// 			} else if !errors.Is(err, mongo.ErrNoDocuments) {
// 				return err
// 			}
//
// 			connInfo := []string{acc}
// 			curSor, err2 := table.Find(sessionContext, bson.M{"accid": accID})
// 			if err2 != nil {
// 				err = err2
// 				return err2
// 			}
// 			for curSor.Next(sessionContext) {
// 				target := &pb2.AccInfo{}
// 				err = curSor.Decode(target)
// 				if err != nil {
// 					continue
// 				}
// 				if target.Acc[0:1] == acc[0:1] {
// 					return errTargetConnected
// 				}
// 				if target.Acc[0:1] == util.ToString(pb2.ESdkNumber_GuestOverSeas) {
// 					data = target
// 				}
// 				connInfo = append(connInfo, target.Acc)
// 			}
//
// 			data.Acc = acc
// 			data.ConnectAcc = connInfo
// 			_, err = table.InsertOne(sessionContext, data)
// 			if err != nil {
// 				_ = sessionContext.AbortTransaction(context.Background())
// 				return err
// 			}
//
// 			update := bson.M{"$set": bson.M{"connectacc": connInfo}}
// 			_, err = table.UpdateMany(sessionContext, bson.M{"accid": accID}, update)
// 			if err != nil {
// 				_ = sessionContext.AbortTransaction(context.Background())
// 				return err
// 			}
//
// 			return sessionContext.CommitTransaction(context.Background())
// 		})
// 	})
// 	if err == nil {
// 		for _, v := range data.ConnectAcc {
// 			ClearAccCache(v)
// 		}
// 	}
// 	return data, err
// }

//
// func DisconnectAcc(roleGuid uint64, accID uint64, acc string) (connectAcc []string, err error) {
// 	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
// 	defer cancel()
// 	acc_db.GetPrimaryDB().SyncExe(func(cli *mongo.Database) {
// 		err = cli.Client().UseSession(ctx, func(sessionContext mongo.SessionContext) error {
// 			if err = sessionContext.StartTransaction(); err != nil {
// 				return err
// 			}
//
// 			table := cli.Collection("acc")
// 			data := pb2.AccInfo{}
// 			err = table.FindOne(sessionContext, bson.M{"acc": acc}).Decode(&data)
// 			if err != nil {
// 				return err
// 			}
//
// 			if accID != data.AccID {
// 				return errAccIDErr
// 			}
//
// 			_, err = table.DeleteOne(sessionContext, bson.M{"acc": acc})
// 			if err != nil {
// 				if err = sessionContext.AbortTransaction(context.Background()); err != nil {
// 					return err
// 				}
// 				return err
// 			}
// 			for i := range data.ConnectAcc {
// 				if data.ConnectAcc[i] == acc {
// 					data.ConnectAcc = sliceDelete(data.ConnectAcc, i)
// 					break
// 				}
// 			}
// 			update := bson.M{"$set": bson.M{"connectacc": data.ConnectAcc}}
// 			_, err = table.UpdateMany(sessionContext, bson.M{"accid": data.AccID}, update)
// 			if err != nil {
// 				if err = sessionContext.AbortTransaction(context.Background()); err != nil {
// 					return err
// 				}
// 				return err
// 			}
// 			connectAcc = data.ConnectAcc
//
// 			return sessionContext.CommitTransaction(context.Background())
// 		})
// 	})
// 	if err == nil { // 在游戏里操作的，这是即使有其它设备登录，也不会添加acccache
// 		for _, v := range connectAcc {
// 			ClearAccCache(v)
// 		}
// 		// 用户账号解绑成功
// 		logMsg := &pb2.MsgDBLog{TaLogs: []*pb2.TaLog{{
// 			EventName:  "AccountBind",
// 			Account:    roleGuid,
// 			DistinctId: "",
// 			Properties: component.BsonMarshal(map[string]interface{}{
// 				"ia_add": false,
// 				"acc":    acc,
// 				"accID":  accID,
// 			}),
// 		}}}
// 		network.SendToRandLog(pb.MsgIDS2S_S2SLog, logMsg)
// 	}
//
// 	ClearAccCache(acc)
// 	return connectAcc, err
// }
//
// func sliceDelete(s []string, delIndex int) []string {
// 	switch {
// 	case delIndex == 0:
// 		return s[1:]
// 	case delIndex >= len(s):
// 		return s
// 	case delIndex+1 == len(s):
// 		return s[0:delIndex]
// 	default:
// 		s1 := s[0:delIndex]
// 		return append(s1, s[delIndex+1:]...)
// 	}
// }
//
// func DeleteAcc(accID uint64) (err error) {
// 	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
// 	defer cancel()
//
// 	accs := make([]string, 0)
// 	acc_db.GetPrimaryDB().SyncExe(func(cli *mongo.Database) {
// 		err = cli.Client().UseSession(ctx, func(sessionContext mongo.SessionContext) error {
// 			if err = sessionContext.StartTransaction(); err != nil {
// 				return err
// 			}
//
// 			table := cli.Collection("acc")
//
// 			var devAcc *pb2.AccInfo
// 			curSor, err := table.Find(sessionContext, bson.M{"accid": accID})
// 			if err != nil {
// 				return err
// 			}
// 			for curSor.Next(sessionContext) {
// 				target := &pb2.AccInfo{}
// 				err = curSor.Decode(target)
// 				if err != nil {
// 					continue
// 				}
//
// 				if target.Acc[0:1] == util.ToString(pb2.ESdkNumber_GuestOverSeas) {
// 					devAcc = target
// 				}
//
// 				accs = append(accs, target.Acc)
// 			}
//
// 			_, err = table.DeleteMany(sessionContext, bson.M{"accid": accID})
// 			if err != nil {
// 				if err = sessionContext.AbortTransaction(context.Background()); err != nil {
// 					return err
// 				}
// 				return err
// 			}
//
// 			devAcc.Acc = devAcc.Acc + "_delete" + util.ToString(time.Now().Unix())
// 			_, err = table.InsertOne(sessionContext, devAcc)
// 			if err != nil {
// 				if err = sessionContext.AbortTransaction(context.Background()); err != nil {
// 					return err
// 				}
// 				return err
// 			}
//
// 			return sessionContext.CommitTransaction(context.Background())
// 		})
// 	})
//
// 	for i := range accs { // 同时在登录的，并且需要写缓存，有极低可能需要重新删一次
// 		ClearAccCache(accs[i])
// 	}
//
// 	return err
// }
