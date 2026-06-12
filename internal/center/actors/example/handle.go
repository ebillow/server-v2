package example

import (
	"server/api/pb"
	"server/pkg/gnet/gmsg"
	"server/pkg/gnet/router"
)

func init() {
	router.On(OnTest)
}

func OnTest(h gmsg.Head, req *pb.S2SEcho) {
}
