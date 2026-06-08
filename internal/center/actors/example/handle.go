package example

import (
	"server/api/pb"
	"server/pkg/gnet/gctx"
	"server/pkg/gnet/router"
)

func init() {
	router.On(OnTest)
}

func OnTest(h gctx.Head, req *pb.S2SEcho) {
}
