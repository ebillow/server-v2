package example

import (
	"google.golang.org/protobuf/proto"
	"server/game/role"
	"server/internal/pb/msgid"
)

func init() {
	role.CRouter().Register(msgid.MsgIDC2S_C2SBindAcc, onGetData)
}

func onGetData(msg proto.Message, r *role.Role) {

}
