package login

import (
	"server/pkg/pb"
)

// new gm order
// func OnGmOrder(msgBase proto.Message, s *snet.Session) {
// 	order := msgBase.(*pb2.MsgGMOrder)
// 	logger.Infof("recv GM order:%s", order.String())
// 	switch order.Command {
// 	case "add local chk list":
// 		PostEvt(EvtParam{Op: OpAddLocalChkList, Str: order.Params})
// 	case "clear local chk list":
// 		PostEvt(EvtParam{Op: OpClearLocalChkList})
// 	case "GmOrderAccountFreezeReq":
// 		req := &def.GmOrderAccountFreezeReq{}
// 		err := json.Unmarshal([]byte(order.Params), &req)
// 		if err != nil {
// 			logger.Warnf("GmOrderAccountFreezeReq json unmarshal err:%v", err)
// 			return
// 		}
// 		Freeze(req.OP, req.Args, req.Other.EndTime)
// 	case "GmOrderSetInnerReq":
// 		req := &def.GmOrderSetInnerReq{}
// 		err := json.Unmarshal([]byte(order.Params), &req)
// 		if err != nil {
// 			logger.Warnf("GmOrderSetInnerReq json unmarshal err:%v", err)
// 			return
// 		}
//
// 		isInner := false
// 		if req.OP == "do_inner" {
// 			isInner = true
// 		}
// 		SetInnerAcc(req.OP, req.Args, isInner)
// 	}
// }

// 维护时可以登录的设备
func isInWhiteList(rawAcc string, dev string) bool {
	return false
}

// 全服关闭时，设置key=0的这项
func onServerState(msg *pb.MsgServerClose) {

}

//
// func serverCloseInfo(world uint32) (*pb.MsgServerCloseInfo, bool) {
// 	if serverClose == nil {
// 		return nil, false
// 	}
//
// 	info, ok := serverClose.Data[0]
// 	if ok && info.Close {
// 		return info, info.Close
// 	}
//
// 	info, ok = serverClose.Data[world]
// 	if !ok {
// 		return nil, false
// 	}
// 	return info, info.Close
// }
//
// func KickOut(accID uint64) {
// 	mtx := locker.NewMutex(lockerAccLogin(accID))
// 	err := mtx.Lock()
// 	if err != nil {
// 		logger.Warnf("get acc Login data err:%v", err)
// 		return
// 	}
// 	defer func() { _, _ = mtx.Unlock() }()
//
// 	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
// 	defer cancel()
//
// 	data, err := getAccLoginData(ctx, accID)
// 	if err != nil {
// 		logger.Warnf("get acc Login data err:%v", err)
// 		return
// 	}
//
// 	data.Token = util.RandToken()
// 	err = saveAccLoginData(ctx, accID, data)
//
// 	network.SendToGame(data.GameID, pb.MsgIDS2S_S2SKickAcc, &pb2.MsgBigUInt{Value: accID})
// }

func Freeze(op string, accIDStrList []string, endTime string) {
	// // todo:
	// accIDList := make([]uint64, 0)
	// for _, v := range accIDStrList {
	// 	accId, _ := strconv.ParseUint(v, 10, 64)
	// 	// if accId > MinGuid && accId < MaxAccCntOnThisWorld {
	// 	accIDList = append(accIDList, accId)
	// 	// }
	// }
	// switch op {
	// case "doban": // 封禁
	// 	endTimeUnix := int64(0)
	// 	if endTime != "" {
	// 		endTimeUnixTime, err := util.ParseInLocation(endTime)
	// 		if err == nil {
	// 			endTimeUnix = endTimeUnixTime.Unix()
	// 		} else {
	// 			logger.Warn(err)
	// 		}
	// 	} else {
	// 		endTime = time.Now().AddDate(1000, 0, 0).Format(util.TimeLayout)
	// 	}
	// 	okList := make([]string, 0)
	// 	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	// 	defer cancel()
	// 	mgo.SyncExe(func(cli *mongo.Database) {
	// 		update := bson.D{{"$set", bson.D{
	// 			{"freeze", true},
	// 			{"freezeendtime", endTimeUnix},
	// 		}}}
	//
	// 		for _, v := range accIDList {
	// 			_, err := cli.Collection("acc").UpdateMany(ctx, bson.M{"accid": v}, update)
	// 			logger.Info("doban", v, err)
	// 			if err == nil {
	// 				okList = append(okList, util.ToString(v))
	// 			}
	// 			clearCacheByAccID(ctx, cli, v)
	// 			KickOut(v)
	// 		}
	// 	})
	// 	// 更新日志
	// 	if len(okList) > 0 {
	// 		dbLog := &pb2.MsgDBLog{SqlDiy: []string{}}
	// 		if len(okList) == 1 {
	// 			dbLog.SqlDiy = append(dbLog.SqlDiy, fmt.Sprintf("UPDATE g_log_acc_sets SET IsFreeze = 1, FreezeTime = %d WHERE ID = %s", endTimeUnix, okList[0]))
	// 		} else {
	// 			accs := strings.Join(okList, ",")
	// 			dbLog.SqlDiy = append(dbLog.SqlDiy, fmt.Sprintf("UPDATE g_log_acc_sets SET IsFreeze = 1, FreezeTime = %d WHERE ID IN (%s)", endTimeUnix, accs))
	// 		}
	// 		network.SendToRandLog(pb.MsgIDS2S_S2SLog, dbLog)
	// 	}
	// case "unban": // 解封
	// 	okList := make([]string, 0)
	// 	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	// 	defer cancel()
	// 	mgo.SyncExe(func(cli *mongo.Database) {
	// 		update := bson.D{{"$set", bson.D{
	// 			{"freeze", false},
	// 			{"freezeendtime", 0},
	// 		}}}
	//
	// 		for _, v := range accIDList {
	// 			_, err := cli.Collection("acc").UpdateMany(ctx, bson.M{"accid": v}, update)
	// 			logger.Info("unban", v, err)
	// 			if err == nil {
	// 				okList = append(okList, util.ToString(v))
	// 			}
	// 			clearCacheByAccID(ctx, cli, v)
	// 		}
	// 	})
	// 	// 更新日志
	// 	if len(okList) > 0 {
	// 		dbLog := &pb2.MsgDBLog{SqlDiy: []string{}}
	// 		if len(okList) == 1 {
	// 			dbLog.SqlDiy = append(dbLog.SqlDiy, fmt.Sprintf("UPDATE g_log_acc_sets SET IsFreeze = 0, FreezeTime = 0 WHERE ID = %s", okList[0]))
	// 		} else {
	// 			accs := strings.Join(okList, ",")
	// 			dbLog.SqlDiy = append(dbLog.SqlDiy, fmt.Sprintf("UPDATE g_log_acc_sets SET IsFreeze = 0, FreezeTime = 0 WHERE ID IN (%s)", accs))
	// 		}
	// 		network.SendToRandLog(pb.MsgIDS2S_S2SLog, dbLog)
	// 	}
	// case "quit": // 下线
	// 	for _, v := range accIDList {
	// 		KickOut(v)
	// 	}
	// }
}

//
// func clearCacheByAccID(ctx context.Context, cli *mongo.Database, accID uint64) {
// 	sor, err := cli.Collection("acc").Find(ctx, bson.M{"accid": accID})
// 	if err != nil {
// 		logger.Warnf("find acc %d err:%v", accID, err)
// 		return
// 	}
// 	for sor.Next(ctx) {
// 		info := &pb2.AccInfo{}
// 		err = sor.Decode(info)
// 		if err != nil {
// 			logger.Warnf("clear cache err:%v", err)
// 			continue
// 		}
// 		ClearAccCache(info.Acc)
// 	}
// }
//
// // SetInnerAcc	设置accid内部账号,isInner=true标识是，否则标识不是
// func SetInnerAcc(op string, accIDStrList []string, isInner bool) {
// 	accIDList := make([]uint64, 0)
// 	for _, v := range accIDStrList {
// 		accId, _ := strconv.ParseUint(v, 10, 64)
// 		accIDList = append(accIDList, accId)
// 	}
//
// 	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
// 	defer cancel()
//
// 	mgo.SyncExe(func(cli *mongo.Database) {
// 		update := bson.D{{"$set", bson.D{
// 			{"inner", isInner},
// 		}}}
// 		for _, v := range accIDList {
// 			_, err := cli.Collection("acc").UpdateMany(ctx, bson.M{"accid": v}, update)
// 			if err != nil {
// 				logger.Warnf("set inner acc setp=1 %d=%t err:%v", v, isInner, err)
// 				return
// 			}
// 			clearCacheByAccID(ctx, cli, v)
// 			logger.Infof("set inner acc %d = %t", v, isInner)
// 		}
// 	})
// }
//
// // BindDev	设置账号绑定设备
// func BindDev(req *pb2.MsgBindDev) {
// 	dbLog := &pb2.MsgDBLog{SqlDiy: []string{}}
// 	mgo.SyncExe(func(cli *mongo.Database) {
// 		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
// 		defer cancel()
// 		for rid, dev := range req.AccBindDev {
// 			accId := share.GetAccIDFromRoleID(rid)
// 			v := dev
// 			if !req.Bind {
// 				v = ""
// 			}
// 			update := bson.D{{"$set", bson.D{
// 				{"binddev", v},
// 			}}}
//
// 			_, err := cli.Collection("acc").UpdateOne(ctx, bson.M{"accid": accId}, update)
// 			if err != nil {
// 				logger.Warnf("bind dev, acc%d->[%s] err:%v", accId, v, err)
// 				return
// 			}
// 			logger.Infof("bind dev, acc%d->[%s]", accId, v)
// 			// 日志
// 			dbLog.SqlDiy = append(dbLog.SqlDiy, fmt.Sprintf("UPDATE g_log_acc_sets SET BindDev = '%s' WHERE ID = %d", v, accId))
//
// 			// dbLog := &pb.MsgDBLog{TableName: "g_log_account"}
// 			// dbLog.SqlDiy = fmt.Sprintf("update g_log_account set bind_dev='%s' where accid = %d", dev, accId)
// 			// dblog.SendToCentLog(pb.MsgIDS2S_S2SLog, dbLog)
// 		}
// 	})
// 	if len(dbLog.SqlDiy) > 0 {
// 		network.SendToRandLog(pb.MsgIDS2S_S2SLog, dbLog)
// 	}
// }

func ShutUp(op string, accIDStrList []uint64, endTime string) {
	// // todo:
	// accIDList := make([]uint64, 0)
	// for _, accId := range accIDStrList {
	// 	accIDList = append(accIDList, accId)
	// }
	// switch op {
	// case "doban": // 封禁
	// 	endTimeUnix := int64(0)
	// 	if endTime != "" {
	// 		endTimeUnixTime, err := util.ParseInLocation(endTime)
	// 		if err == nil {
	// 			endTimeUnix = endTimeUnixTime.Unix()
	// 		} else {
	// 			logger.Warn(err)
	// 		}
	// 	} else {
	// 		endTime = time.Now().AddDate(1000, 0, 0).Format(util.TimeLayout)
	// 	}
	// 	okList := make([]string, 0)
	// 	mgo.SyncExe(func(cli *mongo.Database) {
	// 		update := bson.D{{"$set", bson.D{
	// 			{"shutup", true},
	// 			{"shutupendtime", endTimeUnix},
	// 		}}}
	//
	// 		for _, v := range accIDList {
	// 			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	// 			_, err := cli.Collection("acc").UpdateOne(ctx, bson.M{"accid": v}, update)
	// 			cancel()
	// 			logger.Info("shut up doban", v, err)
	// 			if err == nil {
	// 				okList = append(okList, util.ToString(v))
	// 			}
	// 		}
	// 	})
	// 	// 更新日志
	// 	if len(okList) > 0 {
	// 		dbLog := &pb2.MsgDBLog{SqlDiy: []string{}}
	// 		if len(okList) == 1 {
	// 			dbLog.SqlDiy = append(dbLog.SqlDiy, fmt.Sprintf("UPDATE g_log_acc_sets SET IsShutUp = 1, ShutUpTime = %d WHERE ID = %s", endTimeUnix, okList[0]))
	// 		} else {
	// 			accs := strings.Join(okList, ",")
	// 			dbLog.SqlDiy = append(dbLog.SqlDiy, fmt.Sprintf("UPDATE g_log_acc_sets SET IsShutUp = 1, ShutUpTime = %d WHERE ID IN (%s)", endTimeUnix, accs))
	// 		}
	// 		network.SendToRandLog(pb.MsgIDS2S_S2SLog, dbLog)
	// 	}
	// case "unban": // 解封
	// 	okList := make([]string, 0)
	// 	mgo.SyncExe(func(cli *mongo.Database) {
	// 		update := bson.D{{"$set", bson.D{
	// 			{"shutup", false},
	// 			{"shutupendtime", 0},
	// 		}}}
	// 		for _, v := range accIDList {
	// 			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	// 			_, err := cli.Collection("acc").UpdateOne(ctx, bson.M{"accid": v}, update)
	// 			cancel()
	// 			logger.Info("shut up unban", v, err)
	// 			if err == nil {
	// 				okList = append(okList, util.ToString(v))
	// 			}
	// 		}
	// 	})
	// 	// 更新日志
	// 	if len(okList) > 0 {
	// 		dbLog := &pb2.MsgDBLog{SqlDiy: []string{}}
	// 		if len(okList) == 1 {
	// 			dbLog.SqlDiy = append(dbLog.SqlDiy, fmt.Sprintf("UPDATE g_log_acc_sets SET IsShutUp = 0, ShutUpTime = 0 WHERE ID = %s", okList[0]))
	// 		} else {
	// 			accs := strings.Join(okList, ",")
	// 			dbLog.SqlDiy = append(dbLog.SqlDiy, fmt.Sprintf("UPDATE g_log_acc_sets SET IsShutUp = 0, ShutUpTime = 0 WHERE ID IN (%s)", accs))
	// 		}
	// 		network.SendToRandLog(pb.MsgIDS2S_S2SLog, dbLog)
	// 	}
	// }
}
