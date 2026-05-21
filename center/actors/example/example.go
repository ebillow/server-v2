package example

import (
	"server/pkg/gnet/gctx"
	"server/pkg/gnet/router"
	"server/pkg/pb/msgid"
	"server/pkg/share/app"
	"time"

	"google.golang.org/protobuf/proto"
)

var _ app.ISubActor = &Example{}

func init() {
	router.S().OnG(msgid.MsgIDS2S_S2SOfflineEvt, OnTest)
}

type Example struct {
}

func (e *Example) Init() error {
	return nil
}

func (e *Example) OnTick(now time.Time) {

}

func (e *Example) Exit() {

}

func OnTest(ctx gctx.Context, msgBase proto.Message) {

}
