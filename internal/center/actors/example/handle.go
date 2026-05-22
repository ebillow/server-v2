package example

import (
	"server/api/pb/msgid"
	"server/pkg/gnet/gctx"
	"server/pkg/gnet/router"

	"google.golang.org/protobuf/proto"
)

func init() {
	router.S().OnG(msgid.MsgIDS2S_S2SOfflineEvt, OnTest)
}

func OnTest(ctx gctx.Context, msgBase proto.Message) {
}
