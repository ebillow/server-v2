package logon_service

import (
	"context"
	"server/api/pb"
	"server/internal/game/component"
	"server/internal/game/role"
	"server/internal/share/model"
	"server/pkg/cfg"
	"server/pkg/db"
	"server/pkg/logger"
	"server/pkg/util"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"
)

func TestMain(m *testing.M) {
	cfg.Load("127.0.0.1:2379", "local")

	logger.NewZapLog("../../../bin/logger/test.logger", logger.Config{
		Level:   0,
		Console: true,
	})
	err := db.InitMongo("mongodb://localhost:27017", "game", 10, 16)
	if err != nil {
		panic(err)
	}

	err = db.CreateIndexIfNotExist(db.MongoDB(), "roles", map[string]mongo.IndexModel{
		"role_id": {Keys: bson.D{{"id", 1}}, Options: options.Index().SetUnique(true)},
	})
	if err != nil {
		panic(err)
	}

	err = db.InitRedis(db.RedisCfg{
		Addr: []string{"127.0.0.1:6380", "127.0.0.1:6381", "127.0.0.1:6382"},
	}, 0)
	if err != nil {
		panic(err)
	}

	role.InjectCompCreate(&component.Create)
	role.InjectLoginMgr(&Mgr)

	Mgr.Start()
	m.Run()
}

func checkSuccess() bool {
	debugWait.Wait()
	debugMtx.Lock()
	defer debugMtx.Unlock()

	ok := true
	for k, v := range debugCheck {
		if v == 0 {
			ok = false
			zap.L().Error("role login fail", zap.Uint64("role", k))
		}
	}
	return ok
}

func TestLoadBatch(t *testing.T) {
	ids := make([]uint64, 0)
	for i := uint64(0); i < 10; i++ {
		ids = append(ids, i+1)
	}
	ctx := context.Background()
	filter := bson.M{"id": bson.M{"$in": ids}}
	cursor, err := db.MongoDB().Collection("roles").Find(ctx, filter)
	if err != nil {
		zap.L().Error("find role failed", zap.Error(err))
		return
	}
	defer cursor.Close(ctx)
	var roles []*role.DataToSave
	if err = cursor.All(ctx, &roles); err != nil {
		zap.L().Error("cursor all failed", zap.Error(err))
		return
	}
	t.Log(roles)
}

func TestLoginAndOffline(t *testing.T) {
	Mgr.Login(&pb.S2SReqLogin{Req: &pb.C2SLogin{
		CliInfo: &pb.ClientInfo{Ip: "127.0.0.1"},
	},
		SesID:       222,
		RoleID:      111,
		ReConnToken: 2,
		Seq:         1,
	})

	time.Sleep(time.Second * 2)
	role.Mgr.KickAndWait(111)
	Mgr.Close()
	if !checkSuccess() {
		t.Fatal("login check fail")
	}
}

func TestDataDelete(t *testing.T) {
	roleID := uint64(111)
	db.Redis.Del(context.Background(), model.KeyRole(roleID))

	Mgr.Login(&pb.S2SReqLogin{Req: &pb.C2SLogin{
		CliInfo: &pb.ClientInfo{Ip: "127.0.0.1"},
	},
		SesID:       222,
		RoleID:      111,
		ReConnToken: 2,
		Seq:         1,
	})

	time.Sleep(time.Second * 1)
	role.Mgr.KickAndWait(111)

	if !checkSuccess() {
		t.Fatal("login check fail")
	}
}

const IDMax = 3000

func TestLoginAndOfflineContinue(t *testing.T) {
	go func() {
		ticker := time.NewTicker(time.Millisecond)
		id := uint64(1)
		for {
			select {
			case <-ticker.C:
				Mgr.Login(&pb.S2SReqLogin{Req: &pb.C2SLogin{
					CliInfo: &pb.ClientInfo{Ip: "127.0.0.1"},
				},
					SesID:       id * 2,
					RoleID:      id,
					ReConnToken: 2,
					Seq:         1,
				})
				id++
				if id == IDMax {
					return
				}
			}
		}
	}()

	time.Sleep(time.Second * 3)
	go func() {
		ticker := time.NewTicker(time.Millisecond)
		id := uint64(1)
		for {
			select {
			case <-ticker.C:
				role.Mgr.KickAndWait(id)
				id++
				if id == IDMax {
					return
				}
			}
		}
	}()
	role.Mgr.CloseAndWait()
	Mgr.Close()
	if !checkSuccess() {
		t.Fatal("login check fail")
	}
}

func TestLoginAndOfflineBatch(t *testing.T) {
	for id := uint64(1); id <= IDMax; id++ {
		Mgr.Login(&pb.S2SReqLogin{Req: &pb.C2SLogin{
			CliInfo: &pb.ClientInfo{Ip: "127.0.0.1"},
		},
			SesID:       id * 2,
			RoleID:      id,
			ReConnToken: 2,
			Seq:         1,
		})
	}
	time.Sleep(time.Second * 3)
	role.Mgr.CloseAndWait()
	Mgr.Close()
	if !checkSuccess() {
		t.Fatal("login check fail")
	}
}

func TestOnlineOffline(t *testing.T) {
	for i := 0; i < 50; i++ {
		Mgr.Login(&pb.S2SReqLogin{Req: &pb.C2SLogin{
			CliInfo: &pb.ClientInfo{Ip: "127.0.0.1"},
		},
			SesID:       1 * 3,
			RoleID:      1,
			ReConnToken: 2,
			Seq:         1,
		})
		role.Mgr.KickAndWait(1)
	}
	time.Sleep(time.Second * 3)
	role.Mgr.CloseAndWait()
	Mgr.Close()
}

func TestLoginOtherDev(t *testing.T) {
	for i := 0; i < 1000; i++ {
		Mgr.Login(&pb.S2SReqLogin{Req: &pb.C2SLogin{
			CliInfo: &pb.ClientInfo{Ip: "127.0.0.1"},
		},
			SesID:       1 * 3,
			RoleID:      1,
			ReConnToken: 2,
			Seq:         1,
		})

		time.Sleep(time.Millisecond * time.Duration(util.RandRange(0, 5)))
		Mgr.Login(&pb.S2SReqLogin{Req: &pb.C2SLogin{
			CliInfo: &pb.ClientInfo{Ip: "127.0.0.1"},
		},
			SesID:       1 * 2,
			RoleID:      1,
			ReConnToken: 2,
			Seq:         1,
		})
		time.Sleep(time.Millisecond * time.Duration(util.RandRange(0, 10)))
	}

	role.Mgr.CloseAndWait()
	Mgr.Close()
}

func TestDrain(t *testing.T) {
	c := make(chan int, 1024)
	var wait sync.WaitGroup

	wait.Add(1)
	go func() {
		defer wait.Done()
		for {
			select {
			case d, ok := <-c:
				if !ok {
					t.Log("recv exit")
					return
				} else {
					t.Log(d)
					time.Sleep(time.Second)
				}
			}
		}
	}()

	for i := 0; i < 10; i++ {
		c <- i
	}

	close(c)
	t.Log("send close")
	wait.Wait()
}
