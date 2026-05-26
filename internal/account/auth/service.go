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

func selectGameServer(_ uint32) (uint8, pb.LoginCode) {
	id, ok := discovery.Select(flag.SrvName(pb.Server_Game))
	if !ok {
		return 0, pb.LoginCode_LCNoGame
	}
	return uint8(id), pb.LoginCode_LCSuccess
}

func allocateGameServer(lastGameID uint8, world uint32) (uint8, pb.LoginCode) {
	if lastGameID != 0 { // 已登录过
		if gameExists(lastGameID) {
			return lastGameID, pb.LoginCode_LCSuccess
		}
	}
	return selectGameServer(world)
}

func HandleRoleLogout(accID uint64, sn uint32) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	acc := Account{AccID: accID}
	success, err := acc.ClearGameID(ctx, sn)
	if err != nil {
		zap.S().Warnf("clear game id err:%v", err)
		return
	}

	if !success {
		// 如果不成功，说明玩家已经重新登录并产生了新的 Seq
		// 此时不应该清除 GameID，直接忽略本次登出请求即可
		zap.S().Debugf("ignore logout for accID:%d, seq has changed (old sn:%d)", accID, sn)
	}
}
