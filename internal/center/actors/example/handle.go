package example

import (
	"server/api/pb"
	"server/pkg/gnet/pkg"
	"server/pkg/gnet/router"
)

func init() {
	router.On(OnTest)
}

func OnTest(h pkg.Head, req *pb.S2SEcho) {
}
