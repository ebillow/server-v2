package role_mgr

import (
	"server/pkg/gnet/gctx"
	"server/pkg/gnet/router"
	"server/pkg/pb"
	"server/pkg/pb/msgid"
	"sync"

	"google.golang.org/protobuf/proto"
)

var (
	roles sync.Map
)

func Init() {
	router.S().OnG(msgid.MsgIDS2S_S2SReqLoginOrLogout, onLoginOrLogout)
}

type Data struct {
	SesID  uint64
	GameID uint32
}

func Add(roleID uint64, data Data) {
	roles.Store(roleID, data)
}

func Del(roleID uint64) {
	roles.Delete(roleID)
}

func GetGameID(roleID uint64) (uint32, bool) {
	n, ok := roles.Load(roleID)
	if !ok {
		return 0, false
	}
	return n.(Data).GameID, true
}

func onLoginOrLogout(ctx gctx.Context, msgBase proto.Message) {
	msg := msgBase.(*pb.S2SReqLoginOrLogout)
	if msg.Login {
		Add(msg.RoleID, Data{SesID: msg.RoleID, GameID: msg.GameID})
	} else {
		Del(msg.RoleID)
	}
}
