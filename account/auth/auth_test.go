package auth

import (
	"context"
	"fmt"
	"server/account/acc_db"
	"server/pkg/db"
	"server/pkg/discovery"
	"server/pkg/flag"
	"server/pkg/logger"
	"server/pkg/model"
	"server/pkg/pb"
	"server/pkg/util"
	"strconv"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestMain(m *testing.M) {
	logger.NewZapLog("../../../bin/logger/test.logger", logger.Config{
		Level:   -1,
		Console: true,
	})
	err := db.InitMongo("mongodb://localhost:27017", "account", 10, 16)
	if err != nil {
		panic(err)
	}

	acc_db.CreateIndex()

	err = db.InitRedis(db.RedisCfg{
		Addr: []string{"127.0.0.1:6380", "127.0.0.1:6381", "127.0.0.1:6382"},
	}, 0)
	if err != nil {
		panic(err)
	}

	err = discovery.Init([]string{"127.0.0.1:2379"}, db.Redis)
	if err != nil {
		panic(err)
	}
	discovery.Watch()
	err = discovery.RegisterDefault(flag.SrvName(pb.Server_Game), &discovery.Node{SvcName: flag.SrvName(pb.Server_Game), NodeID: 1})
	if err != nil {
		panic(err)
	}

	Start(context.Background())
	m.Run()
}

func checkSuccess() bool {
	debugWait.Wait()
	ok := true
	fmt.Println("start check success,total:", len(debugAcc))
	for k, v := range debugAcc {
		if !v.Ok {
			fmt.Println("check fail", k, v.AccID)
			ok = false
		}
	}
	fmt.Println("finish check success,total:", len(debugAcc))
	return ok
}

func TestLoginBatch(t *testing.T) {
	// 正常情况
	for i := 1; i <= 10000; i++ {
		Login(&pb.S2SReqLogin{
			Req: &pb.C2SLogin{
				SdkType:   pb.SdkType(i % 4),
				Account:   "test" + strconv.Itoa(i),
				Reconnect: false,
				CliInfo:   &pb.ClientInfo{Ip: "127.0.0.1"},
			},
			SesID:  uint64(i),
			RoleID: uint64(i),
			Seq:    uint32(util.RandRange(0, 10)),
		})
	}
	if !checkSuccess() {
		t.Fatal("check fail")
	}
}

func TestLoginRedisExpire(t *testing.T) {
	ctx := context.Background()
	for i := 1; i <= 10000; i++ {
		keyBind := model.KeyAccBind(RealAcc(pb.SdkType(i%4), "test"+strconv.Itoa(i)))
		if util.Happen(5000) {
			accID, err := db.Redis.Get(ctx, keyBind).Result()
			if err != nil && err != redis.Nil {
				t.Fatal("redis get fail")
			}
			db.Redis.Del(ctx, model.KeyAccount(util.ParseUintDef(accID, uint64(0))))
		}
		if util.Happen(5000) {
			db.Redis.Del(ctx, keyBind)
		}
	}

	for i := 1; i <= 10000; i++ {
		Login(&pb.S2SReqLogin{
			Req: &pb.C2SLogin{
				SdkType:   pb.SdkType(i % 4),
				Account:   "test" + strconv.Itoa(i),
				Reconnect: false,
				CliInfo:   &pb.ClientInfo{Ip: "127.0.0.1"},
			},
			SesID:  uint64(i),
			RoleID: uint64(i),
			Seq:    uint32(util.RandRange(0, 10)),
		})
	}
	if !checkSuccess() {
		t.Fatal("check fail")
	}
}

func TestDBAndRedis(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	sor, err := db.MongoDB().Collection(AccountCollection).Find(ctx, bson.M{})
	if err != nil {
		panic(err)
	}
	defer sor.Close(ctx)
	for sor.Next(ctx) {
		acc := Account{}
		err = sor.Decode(&acc)
		if acc.AccID == 0 {
			t.Error("acc id is 0")
		}
		var accIDStr, appleID, googleID, fbID string
		if len(acc.Device) > 0 {
			accIDStr, err = db.Redis.Get(ctx, model.KeyAccBind(acc.Device)).Result()
			if err != nil && err != redis.Nil {
				t.Error(err)
				continue
			}

			accID, err := strconv.Atoi(accIDStr)
			if err != nil {
				t.Error(err)
			}
			if acc.AccID != uint64(accID) {
				t.Error("acc id is wrong")
			}
		}

		if len(acc.AppleID) > 0 {
			appleID, err = db.Redis.Get(ctx, model.KeyAccBind(acc.AppleID)).Result()
			if err != nil && err != redis.Nil {
				t.Error(err)
				continue
			}

			accID, err := strconv.Atoi(appleID)
			if err != nil {
				t.Error(err)
			}
			if acc.AccID != uint64(accID) {
				t.Error("acc id is wrong")
			}
		}

		if len(acc.GoogleID) > 0 {
			googleID, err = db.Redis.Get(ctx, model.KeyAccBind(acc.GoogleID)).Result()
			if err != nil && err != redis.Nil {
				t.Error(err)
				continue
			}
			accID, err := strconv.Atoi(googleID)
			if err != nil {
				t.Error(err)
			}
			if acc.AccID != uint64(accID) {
				t.Error("acc id is wrong")
			}
		}

		if len(acc.FbID) > 0 {
			fbID, err = db.Redis.Get(ctx, model.KeyAccBind(acc.FbID)).Result()
			if err != nil && err != redis.Nil {
				t.Error(err)
				continue
			}
			accID, err := strconv.Atoi(fbID)
			if err != nil {
				t.Error(err)
			}
			if acc.AccID != uint64(accID) {
				t.Error("acc id is wrong")
			}
		}
	}
}

func TestLoginApple(t *testing.T) {
	for i := 20000; i < 20001; i++ {
		Login(&pb.S2SReqLogin{
			Req: &pb.C2SLogin{
				SdkType:   pb.SdkType_Apple,
				Account:   "test" + strconv.Itoa(i),
				Token:     "",
				Channel:   0,
				Dev:       "",
				Area:      0,
				Version:   "",
				Reconnect: false,
				CliInfo:   &pb.ClientInfo{Ip: "127.0.0.1"},
			},
			SesID:        1,
			RoleID:       0,
			ReConnToken:  0,
			Seq:          0,
			ConnectedAcc: nil,
		})
	}
	if !checkSuccess() {
		t.Fatal("check fail")
	}
}

func TestLogin(t *testing.T) {
	for i := 0; i < 5000; i++ {
		Login(&pb.S2SReqLogin{
			Req: &pb.C2SLogin{
				SdkType:   pb.SdkType_Guest,
				Account:   "test" + strconv.Itoa(i),
				Token:     "",
				Channel:   0,
				Dev:       "",
				Area:      0,
				Version:   "",
				Reconnect: false,
				CliInfo:   &pb.ClientInfo{Ip: "127.0.0.1"},
			},
			SesID:        1,
			RoleID:       0,
			ReConnToken:  0,
			Seq:          0,
			ConnectedAcc: nil,
		})
	}
	if !checkSuccess() {
		t.Fatal("check fail")
	}
}

func TestLoginRandSdk(t *testing.T) {
	for i := 10000; i < 15000; i++ {
		Login(&pb.S2SReqLogin{
			Req: &pb.C2SLogin{
				SdkType:   pb.SdkType(util.RandRange(0, 4)),
				Account:   "test" + strconv.Itoa(i),
				Token:     "",
				Channel:   0,
				Dev:       "",
				Area:      0,
				Version:   "",
				Reconnect: false,
				CliInfo:   &pb.ClientInfo{Ip: "127.0.0.1"},
			},
			SesID:        1,
			RoleID:       0,
			ReConnToken:  0,
			Seq:          0,
			ConnectedAcc: nil,
		})
	}
	if !checkSuccess() {
		t.Fatal("check fail")
	}
}

func TestLoginDup(t *testing.T) {
	for i := 1; i <= 100; i++ {
		Login(&pb.S2SReqLogin{
			Req: &pb.C2SLogin{
				SdkType:   pb.SdkType(i % 4),
				Account:   "test" + strconv.Itoa(i),
				Reconnect: false,
				CliInfo:   &pb.ClientInfo{Ip: "127.0.0.1"},
			},
			SesID:  uint64(i),
			RoleID: uint64(i),
			Seq:    uint32(util.RandRange(0, 10)),
		})
		Login(&pb.S2SReqLogin{
			Req: &pb.C2SLogin{
				SdkType:   pb.SdkType(i % 4),
				Account:   "test" + strconv.Itoa(i),
				Reconnect: false,
				CliInfo:   &pb.ClientInfo{Ip: "127.0.0.1"},
			},
			SesID:  uint64(i),
			RoleID: uint64(i),
			Seq:    uint32(util.RandRange(0, 10)),
		})
	}
	// if !checkSuccess() {
	// 	t.Fatal("check fail")
	// }
}
