package login

import (
	"context"
	"encoding/json"
	"math/rand/v2"
	"server/pkg/db"
	"server/pkg/logger"
	"server/pkg/model"
	"server/pkg/pb"
	"server/pkg/share"
	"server/pkg/util"
	"time"
)

func getGameState(gameID int32) (*pb.S2SGameState, bool) {
	b, err := db.Redis.HGet(context.Background(),
		model.RedisKeyServerState, util.ToString(gameID)).Bytes()
	if err != nil {
		logger.Warnf("getGameState id=%d err:%v", gameID, err)
		return nil, false
	}
	data := &pb.S2SGameState{}
	err = json.Unmarshal(b, data)
	if err != nil {
		logger.Warnf("getGameState id=%d err:%v", gameID, err)
		return nil, false
	}
	return data, true
}

func findMinGame(ss map[string]string) int32 {
	var min *pb.S2SGameState
	now := time.Now().Unix()
	for i := range ss {
		data := &pb.S2SGameState{}
		err := json.Unmarshal([]byte(ss[i]), data)
		if err != nil {
			logger.Warnf("getGameState err:%v", err)
			continue
		}
		if now-data.Time > 10*2 {
			continue
		}
		if min == nil {
			min = data
		} else if min.RoleCnt > data.RoleCnt {
			min = data
		}
	}
	if min == nil {
		return 0
	}
	return min.GetGameID()
}

func randNewOne(ss map[string]string) int32 {
	now := time.Now().Unix()
	cnt := util.RandInt(len(ss))
	i := 0
	for _, v := range ss {
		if i < cnt {
			continue
		}
		data := &pb.S2SGameState{}
		err := json.Unmarshal([]byte(v), data)
		if err != nil {
			logger.Warnf("getGameState err:%v", err)
			continue
		}
		if now-data.Time > 10*2 {
			continue
		}
		return data.GameID
	}
	return 0
}

func randGameID(world uint32) int32 {
	ss := db.Redis.HGetAll(context.Background(), model.RedisKeyServerState).Val() // 机器多了要优化
	gameID := randNewOne(ss)
	if gameID == 0 {
		gameID = findMinGame(ss)
	}
	return gameID
}

func choseGame(lastGameID int32, world uint32) int32 {
	if lastGameID != 0 { // 已登录过
		serInfo, ok := getGameState(lastGameID)
		if ok {
			logger.Tracef("game state:%s", serInfo)
			switch serInfo.State {
			case pb.S2SGameState_GmRunning:
				if time.Now().Unix()-serInfo.Time > share.GameStateUpdateTime*6 {
					return randGameID(world)
				} else {
					return lastGameID
				}
			case pb.S2SGameState_GmClose:
				return randGameID(world)
			default: // timeout saveData
				return 0
			}
		} else { //
			return randGameID(world)
		}
	} else { // 第一次登录
		return randGameID(world)
	}
}

func getGameCanEnter(acc *Account, req *pb.S2SReqLogin) (*Account, pb.LoginCode) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	if acc.Passwd != 0 && req.ReConnToken != acc.Passwd {
		return acc, pb.LoginCode_LCCanNotReConn
	}

	now := time.Now().Unix()
	if now < acc.Time+LoginCD {
		return acc, pb.LoginCode_LCCD
	}

	acc.Time = now
	if acc.Passwd == 0 {
		acc.Passwd = rand.Uint64()
	}

	// 获取 game id
	gameID := choseGame(acc.GameID, 0)
	if gameID == 0 {
		return acc, pb.LoginCode_LCNoGame
	}
	acc.Seq = rand.Uint32()
	acc.GameID = gameID
	err := acc.SaveLoginData(ctx)
	if err != nil {
		logger.Warnf("save acc Login data err:%v", err)
		return nil, pb.LoginCode_LCServerErr
	}
	return acc, pb.LoginCode_LCSuccess
}

func onRoleLogout(account string, sn int32) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	acc := Account{Account: account}
	seq := acc.LoadSeq(ctx)
	if seq == sn {
		acc.GameID = 0
		err := acc.SaveLoginData(ctx)
		if err != nil {
			logger.Warnf("save acc Login data err:%v", err)
		}
	}
}
