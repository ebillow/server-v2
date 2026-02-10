package login

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/bson"
	"server/account/acc_db"
	"server/pkg/db"
	"server/pkg/discovery"
	"server/pkg/logger"
	"server/pkg/model"
	"server/pkg/pb"
	"server/pkg/util"
	"strconv"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	logger.NewZapLog("../../../bin/logger/test.logger", logger.Config{
		Level:   0,
		Console: true,
	})
	err := db.InitMongo(db.MongoCfg{
		URI:    "mongodb://localhost:27017",
		DbName: "account",
	}, 10, 16)
	if err != nil {
		panic(err)
	}

	acc_db.CreateAccIndex()

	err = db.InitRedis(db.RedisCfg{
		Addr: []string{"127.0.0.1:6380", "127.0.0.1:6381", "127.0.0.1:6382"},
	}, 0)
	if err != nil {
		panic(err)
	}

	err = discovery.Init([]string{"127.0.0.1:2379"})
	if err != nil {
		panic(err)
	}
	discovery.Watch()
	err = discovery.Register(pb.Server_Game, 1)
	if err != nil {
		panic(err)
	}

	Start(context.Background())
	m.Run()
}

func checkSuccess() {
	fmt.Println("start check success")
	for k, v := range debugAcc {
		if !v.Ok {
			fmt.Println("login fail", k, v.AccID)
		}
	}
}

func TestDBAndRedis(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	sor, err := db.MongoDB.Collection(acc_db.AccountTable).Find(ctx, bson.M{})
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
				SdkNo:     pb.ESdkNumber_Apple,
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
	time.Sleep(time.Second * 10)
	checkSuccess()
}

func TestLogin(t *testing.T) {
	for i := 0; i < 5000; i++ {
		Login(&pb.S2SReqLogin{
			Req: &pb.C2SLogin{
				SdkNo:     pb.ESdkNumber_Guest,
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
	time.Sleep(time.Second * 10)
	checkSuccess()
}

func TestLoginRandSdk(t *testing.T) {
	for i := 10000; i < 15000; i++ {
		Login(&pb.S2SReqLogin{
			Req: &pb.C2SLogin{
				SdkNo:     pb.ESdkNumber(util.RandInt(4)),
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
	time.Sleep(time.Second * 10)
	checkSuccess()
}
