package offline_evt

import (
	"os"
	"server/api/pb"
	"server/pkg/db"
	"testing"
)

func TestMain(m *testing.M) {
	err := db.InitRedis(db.RedisCfg{
		Addr: []string{"127.0.0.1:6380", "127.0.0.1:6381", "127.0.0.1:6382"},
	}, 0)
	if err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

func TestAdd(t *testing.T) {
	roleID := uint64(5000)
	for i := uint64(0); i <= MaxEvents+1; i++ {
		Add(roleID, &pb.S2SReqLogin{
			SesID:  i,
			RoleID: i * 2,
		})
	}
}

type U struct {
}

func (u *U) GetID() uint64 {
	return 5000
}
func TestDo(t *testing.T) {
	RegisterHandler(func(user IRole, msg *pb.S2SReqLogin) {
		t.Log("handle", msg)
	})
	u := &U{}
	Do(u)
}
