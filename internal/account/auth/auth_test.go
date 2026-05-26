package auth

import (
	"context"
	"fmt"
	pb "server/api/pb"
	"server/internal/share/model"
	"server/pkg/cfg"
	"server/pkg/db"
	"server/pkg/discovery"
	"server/pkg/flag"
	"server/pkg/idgen"
	"server/pkg/logger"
	"server/pkg/util"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestMain(m *testing.M) {
	cfg.Load("127.0.0.1:2379", "local")
	idgen.Init(1)

	logger.NewZapLog("../../../bin/logs/test.logger", logger.Config{
		Level:   -1,
		Console: true,
	})
	err := db.InitMongo("mongodb://localhost:27017", "local_account", 10, 16)
	if err != nil {
		panic(err)
	}

	// acc_db.CreateIndex()

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
	time.Sleep(time.Second * 2)
	StartService(context.Background())
	m.Run()
}

func checkSuccess() int {
	debugWait.Wait()
	failed := 0
	fmt.Println("start check success,total:", len(debugAcc))
	for k, v := range debugAcc {
		if !v.Ok {
			fmt.Println("check fail", k, v.AccID)
			failed++
		}
	}
	fmt.Println("finish check success,total:", len(debugAcc))
	return failed
}

func TestLogin_Concurrency_SameAccount(t *testing.T) {
	ctx := context.Background()

	time.Sleep(time.Millisecond * 100) // 等待协程启动

	// 模拟极端并发：100 个客户端同时用同一个 AppleID 登录
	const concurrentCount = 100
	var wg sync.WaitGroup

	appleID := "apple_test_888"

	reqs := make([]*pb.S2SReqLogin, concurrentCount)
	for i := 0; i < concurrentCount; i++ {
		req := &pb.S2SReqLogin{
			SesID: uint64(i), // 不同的会话ID
			Req: &pb.C2SLogin{
				Account: appleID,
				SdkType: pb.SdkType_Apple,
				Dev:     "iPhone14",
			},
		}
		reqs[i] = req
		DebugAdd(req)
	}

	for _, v := range reqs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// 直接跳过网络层，把事件塞进队列
			// 此时由于没有真实 SDK，我们模拟 SDK 验证成功，直接推入 Loader
			PushToLoader(v)
		}()
	}

	// 等待所有请求投递完毕
	wg.Wait()

	// 给 Loader 足够的时间处理这 100 个并发请求
	time.Sleep(time.Second * 2)

	// 验证结果：
	// 1. MongoDB 中该 AppleID 只能有一条记录（AccID）
	count, err := db.MongoDB().Collection(AccountCollection).CountDocuments(ctx, bson.M{"apple_id": appleID})
	if err != nil {
		t.Fatalf("mongo count error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 account created, got %d. Cross-talk (串号) occurred!", count)
	}

	// 2. Redis 中的 Seq 必须是 1（只成功处理了一次登录流程）
	acc := &Account{}
	err = db.MongoDB().Collection(AccountCollection).FindOne(ctx, bson.M{"apple_id": appleID}).Decode(acc)
	if err != nil {
		t.Fatalf("mongo find error: %v", err)
	}

	redisSeq, err := db.Redis.HGet(ctx, model.KeyAccount(acc.AccID), "seq").Int()
	if err != nil {
		t.Fatalf("redis get seq error: %v", err)
	}
	if redisSeq != 1 {
		t.Fatalf("expected redis seq to be 1, got %d. CAS protection failed!", redisSeq)
	}
}

func TestLogin_RateLimit(t *testing.T) {
	req := &pb.S2SReqLogin{
		Req: &pb.C2SLogin{
			Account: "test_cd_user",
			SdkType: pb.SdkType_Guest,
		},
	}

	// 第一次登录，应该成功
	code := checkLoginRateLimit(req)
	if code != pb.LoginCode_LCSuccess {
		t.Fatalf("first login should succeed, got %v", code)
	}

	// 立刻第二次登录，应该触发 CD
	code = checkLoginRateLimit(req)
	if code != pb.LoginCode_LCCD {
		t.Fatalf("second login should hit CD, got %v", code)
	}

	// 等待 CD 结束 (LoginCD 常量为 3 秒)
	time.Sleep(time.Second * 4)

	// 第三次登录，应该成功
	code = checkLoginRateLimit(req)
	if code != pb.LoginCode_LCSuccess {
		t.Fatalf("login after CD should succeed, got %v", code)
	}
}
func TestLoginOne(t *testing.T) {
	HandleLoginRequest(&pb.S2SReqLogin{
		Req: &pb.C2SLogin{
			SdkType:   pb.SdkType_Google,
			Account:   "test" + strconv.Itoa(86),
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
	checkSuccess()
}

func TestLoginOneFromDB(t *testing.T) {
	ctx := context.Background()
	keyBind := model.KeyAccBind(pb.SdkType_Guest, "test"+strconv.Itoa(1))

	accID, err := db.Redis.Get(ctx, keyBind).Result()
	if err != nil && err != redis.Nil {
		t.Fatal("redis get fail")
	}
	db.Redis.Del(ctx, model.KeyAccount(util.ParseUintDef(accID, uint64(0))))
	db.Redis.Del(ctx, keyBind)

	HandleLoginRequest(&pb.S2SReqLogin{
		Req: &pb.C2SLogin{
			SdkType:   pb.SdkType_Guest,
			Account:   "test" + strconv.Itoa(1),
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
	checkSuccess()
	time.Sleep(time.Second * 3)
	db.Redis.Del(ctx, model.KeyAccount(util.ParseUintDef(accID, uint64(0))))
	HandleLoginRequest(&pb.S2SReqLogin{
		Req: &pb.C2SLogin{
			SdkType:   pb.SdkType_Guest,
			Account:   "test" + strconv.Itoa(1),
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
	checkSuccess()
	time.Sleep(time.Second * 3)
	db.Redis.Del(ctx, keyBind)
	HandleLoginRequest(&pb.S2SReqLogin{
		Req: &pb.C2SLogin{
			SdkType:   pb.SdkType_Guest,
			Account:   "test" + strconv.Itoa(1),
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
	checkSuccess()
}

func TestLoginBatch(t *testing.T) {
	// 正常情况
	for i := 1; i <= 10000; i++ {
		HandleLoginRequest(&pb.S2SReqLogin{
			Req: &pb.C2SLogin{
				SdkType:   pb.SdkType_Guest,
				Account:   "test" + strconv.Itoa(i),
				Reconnect: false,
				CliInfo:   &pb.ClientInfo{Ip: "127.0.0.1"},
			},
			SesID:  uint64(i),
			RoleID: uint64(i),
		})
	}
	checkSuccess()
}

func TestLoginBatchRandType(t *testing.T) {
	// 正常情况
	for i := 1; i <= 10000; i++ {
		HandleLoginRequest(&pb.S2SReqLogin{
			Req: &pb.C2SLogin{
				SdkType:   pb.SdkType(i % 4),
				Account:   "test" + strconv.Itoa(i),
				Reconnect: false,
				CliInfo:   &pb.ClientInfo{Ip: "127.0.0.1"},
			},
			SesID:  uint64(i),
			RoleID: uint64(i),
		})
	}
	checkSuccess()
}

func TestLoginRedisExpire(t *testing.T) {
	ctx := context.Background()
	for i := 1; i <= 10000; i++ {
		keyBind := model.KeyAccBind(pb.SdkType(i%4), "test"+strconv.Itoa(i))
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
		HandleLoginRequest(&pb.S2SReqLogin{
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
	checkSuccess()
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
			accIDStr, err = db.Redis.Get(ctx, model.KeyAccBind(pb.SdkType_Guest, acc.Device)).Result()
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
			appleID, err = db.Redis.Get(ctx, model.KeyAccBind(pb.SdkType_Apple, acc.AppleID)).Result()
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
			googleID, err = db.Redis.Get(ctx, model.KeyAccBind(pb.SdkType_Google, acc.GoogleID)).Result()
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
			fbID, err = db.Redis.Get(ctx, model.KeyAccBind(pb.SdkType_Facebook, acc.FbID)).Result()
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
		HandleLoginRequest(&pb.S2SReqLogin{
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
	checkSuccess()
}

func TestLogin(t *testing.T) {
	for i := 0; i < 5000; i++ {
		HandleLoginRequest(&pb.S2SReqLogin{
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
	checkSuccess()
}

func TestLoginRandSdk(t *testing.T) {
	for i := 10000; i < 15000; i++ {
		HandleLoginRequest(&pb.S2SReqLogin{
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
	checkSuccess()
}

func TestLoginDup(t *testing.T) {
	for i := 1; i <= 100; i++ {
		HandleLoginRequest(&pb.S2SReqLogin{
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
		HandleLoginRequest(&pb.S2SReqLogin{
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
	checkSuccess()
}
