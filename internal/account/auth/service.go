package auth

import (
	"context"
	pb "server/api/pb"
	"server/pkg/discovery"
	"server/pkg/flag"
	"time"

	"go.uber.org/zap"
)

func gameExists(gameID uint8) bool {
	return discovery.Exists(flag.SrvName(pb.Server_Game), int32(gameID))
}

func randGameID(_ uint32) (uint8, pb.LoginCode) {
	id, ok := discovery.Select(flag.SrvName(pb.Server_Game))
	if !ok {
		return 0, pb.LoginCode_LCNoGame
	}
	return uint8(id), pb.LoginCode_LCSuccess
}

func chooseGame(lastGameID uint8, world uint32) (uint8, pb.LoginCode) {
	if lastGameID != 0 { // 已登录过
		if gameExists(lastGameID) {
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
			zap.S().Warnf("save acc Login data err:%v", err)
		}
	}
}
