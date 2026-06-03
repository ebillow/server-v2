package example

import (
	"server/api/pb"
	"server/pkg/gnet/gctx"
	"server/pkg/gnet/router"
)

func init() {
	router.OnS2S(OnTest)
}

func OnTest(ctx gctx.Context, req *pb.S2SEcho) {
}
