package login

import (
	"context"
	"fmt"
	"server/account/acc_db"
	"server/pkg/db"
	"server/pkg/discovery"
	"server/pkg/logger"
	"server/pkg/pb"
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
