package login_mgr

import (
	"context"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"
	"server/game/logic"
	"server/game/role"
	"server/game/role/role_mgr"
	"server/internal/db"
	"server/internal/log"
	"server/internal/model"
	"server/internal/pb"
	"server/internal/util"
	"sync"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	log.Init("../../../bin/log/test.log", &log.Config{
		Level:   "debug",
		Console: true,
	})
	err := db.InitMongo(&db.MongoCfg{
		URI:    "mongodb://localhost:27017",
		DbName: "game",
	}, 10, 16)
	if err != nil {
		panic(err)
	}

	err = db.CreateIndexIfNotExist(db.MongoDB, "roles", map[string]mongo.IndexModel{
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

	role.CreateComps = logic.CreateComps
	role.SetLoginMgr(&Mgr)
	role.SetRoleMgr(role_mgr.Mgr)
	Mgr.Start()
	m.Run()
}

type LoginCheck struct {
	loginCnt map[uint64]int64
	mtx      sync.Mutex
}

func NewLoginCheck() *LoginCheck {
	ctx := context.Background()

	cur := uint64(0)
	var ss []string
	for {
		ss, cur = db.Redis.Scan(ctx, cur, "role:*", 1000).Val()
		for _, s := range ss {
			db.Redis.Del(context.Background(), s)
		}
		if cur == 0 {
			break
		}
	}

	return &LoginCheck{
		loginCnt: make(map[uint64]int64),
	}
}

func (l *LoginCheck) Add(id uint64) {
	l.mtx.Lock()
	l.loginCnt[id]++
	l.mtx.Unlock()
}

func (l *LoginCheck) CheckResult() {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	ctx := context.Background()
	for k, v := range l.loginCnt {
		datas := db.Redis.HGetAll(ctx, model.KeyRole(k)).Val()
		data := role.DataToSave{Data: datas}
		r := pb.RoleData{}
		err := jsoniter.UnmarshalFromString(data.Get(pb.TypeComp_TCBase), &r)
		if err != nil {
			panic(err)
		}
		if int64(r.Country) != int64(r.Exp) {
			fmt.Printf("role=%d cnt=%d login=%d offline=%d\n", k, v, r.Exp, int64(r.Country))
			panic(v)
		}
		// if int64(r.Exp) != v {
		// 	fmt.Printf("role=%d cnt=%d login=%d offline=%d\n", k, v, r.Exp, int64(r.Country))
		// }
	}
	fmt.Println("check result finished")
}

func TestLoadBatch(t *testing.T) {
	ids := make([]uint64, 0)
	for i := uint64(0); i < 10; i++ {
		ids = append(ids, i+1)
	}
	ctx := context.Background()
	filter := bson.M{"id": bson.M{"$in": ids}}
	cursor, err := db.MongoDB.Collection("roles").Find(ctx, filter)
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
	c := NewLoginCheck()
	Mgr.Online(&pb.C2SLogin{
		CliInfo:     &pb.ClientInfo{Ip: "127.0.0.1"},
		SesID:       222,
		RoleID:      111,
		PlatType:    0,
		ReConnToken: 2,
		Seq:         1,
	})
	c.Add(111)

	time.Sleep(time.Second * 2)
	role.GetRoleMgr().PostCloseAndWait(111)
	Mgr.Close()
	c.CheckResult()
}

func TestDataDelete(t *testing.T) {
	c := NewLoginCheck()
	Mgr.Online(&pb.C2SLogin{
		CliInfo:     &pb.ClientInfo{Ip: "127.0.0.1"},
		SesID:       222,
		RoleID:      111,
		PlatType:    0,
		ReConnToken: 2,
		Seq:         1,
	})
	c.Add(111)

	time.Sleep(time.Second * 1)
	role.GetRoleMgr().PostCloseAndWait(111)
	time.Sleep(time.Second * 10)
	c.CheckResult()

	time.Sleep(time.Minute * 6)
}

func TestBson(t *testing.T) {
	d := pb.RoleData{
		ID:    2,
		Level: 100,
		Exp:   9999,
		Name:  "testName",
		Items: map[string]int64{"Gold": 888, "ItemA": 5555},
	}
	b, err := bson.Marshal(&d)
	require.NoError(t, err)
	t.Log(string(b))

	db.Redis.Set(context.Background(), "test:bson", string(b), 0)

	b2 := db.Redis.Get(context.Background(), "test:bson").Val()
	d2 := pb.RoleData{}
	err = bson.Unmarshal([]byte(b2), &d2)
	require.NoError(t, err)
	t.Log(&d2)
}

const IDMax = 3000

func TestLoginAndOfflineContinue(t *testing.T) {
	c := NewLoginCheck()
	go func() {
		ticker := time.NewTicker(time.Millisecond)
		id := uint64(1)
		for {
			select {
			case <-ticker.C:
				Mgr.Online(&pb.C2SLogin{
					CliInfo:     &pb.ClientInfo{Ip: "127.0.0.1"},
					SesID:       id * 2,
					RoleID:      id,
					PlatType:    0,
					ReConnToken: 2,
					Seq:         1,
				})
				c.Add(id)
				id++
				if id == IDMax {
					return
				}
			}
		}
	}()

	time.Sleep(time.Second * 10)
	go func() {
		ticker := time.NewTicker(time.Millisecond)
		id := uint64(1)
		for {
			select {
			case <-ticker.C:
				role.GetRoleMgr().PostCloseAndWait(id)
				id++
				if id == IDMax {
					return
				}
			}
		}
	}()
	role.GetRoleMgr().CloseAndWait()
	Mgr.Close()
	c.CheckResult()
}

func TestLoginAndOfflineBatch(t *testing.T) {
	c := NewLoginCheck()
	for id := uint64(1); id <= IDMax; id++ {
		Mgr.Online(&pb.C2SLogin{
			CliInfo:     &pb.ClientInfo{Ip: "127.0.0.1"},
			SesID:       id * 2,
			RoleID:      id,
			PlatType:    0,
			ReConnToken: 2,
			Seq:         1,
		})
		c.Add(id)
	}

	MockMsg()

	role.GetRoleMgr().CloseAndWait()
	Mgr.Close()
	c.CheckResult()
}

func MockMsg() {
	t := time.NewTicker(time.Millisecond * 100)
	out := time.After(time.Second * 10)
	for {
		select {
		case <-t.C:
			for id := uint64(1); id <= IDMax; id++ {
				role.GetRoleMgr().PostEvent(id, role.Event{
					MsgID: 1,
					Data:  []byte("hello"),
				})
			}
		case <-out:
			return
		}
	}
}

func TestOnlineOffline(t *testing.T) {
	c := NewLoginCheck()
	for i := 0; i < 50; i++ {
		Mgr.Online(&pb.C2SLogin{
			CliInfo:     &pb.ClientInfo{Ip: "127.0.0.1"},
			SesID:       1 * 3,
			RoleID:      1,
			PlatType:    0,
			ReConnToken: 2,
			Seq:         1,
		})
		c.Add(1)
		role.GetRoleMgr().PostCloseAndWait(1)
	}
	time.Sleep(time.Second * 3)
	role.GetRoleMgr().CloseAndWait()
	Mgr.Close()
	c.CheckResult()
}

func TestLoginOtherDev(t *testing.T) {
	c := NewLoginCheck()
	for i := 0; i < 1000; i++ {
		Mgr.Online(&pb.C2SLogin{
			CliInfo:     &pb.ClientInfo{Ip: "127.0.0.1"},
			SesID:       1 * 3,
			RoleID:      1,
			PlatType:    0,
			ReConnToken: 2,
			Seq:         1,
		})
		c.Add(1)

		time.Sleep(time.Millisecond * time.Duration(util.RandInt(5)))
		Mgr.Online(&pb.C2SLogin{
			CliInfo:     &pb.ClientInfo{Ip: "127.0.0.1"},
			SesID:       1 * 2,
			RoleID:      1,
			PlatType:    0,
			ReConnToken: 2,
			Seq:         1,
		})
		c.Add(1)
		time.Sleep(time.Millisecond * time.Duration(util.RandInt(10)))
	}

	role.GetRoleMgr().CloseAndWait()
	Mgr.Close()
	c.CheckResult()
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
