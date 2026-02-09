package login

import (
	"context"
	"server/pkg/discovery"
	"server/pkg/flag"
	"server/pkg/logger"
	"server/pkg/pb"
	"time"
)

func gameExist(gameID int32) bool {
	return discovery.Exist(flag.SrvName(pb.Server_Game), gameID)
}

func randGameID(world uint32) (int32, pb.LoginCode) {
	id, ok := discovery.Pick(flag.SrvName(pb.Server_Game))
	if !ok {
		return 0, pb.LoginCode_LCNoGame
	}
	return id, pb.LoginCode_LCSuccess
}

func choseGame(lastGameID int32, world uint32) (int32, pb.LoginCode) {
	if lastGameID != 0 { // 已登录过
		if gameExist(lastGameID) {
			return lastGameID, pb.LoginCode_LCSuccess
		}
	}
	return randGameID(world)
}

func onRoleLogout(accID uint64, sn uint32) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	acc := Account{AccID: accID}
	seq := acc.LoadSeq(ctx)
	if seq == sn {
		acc.GameID = 0
		err := acc.SaveLoginData(ctx)
		if err != nil {
			logger.Warnf("save acc Login data err:%v", err)
		}
	}
}
