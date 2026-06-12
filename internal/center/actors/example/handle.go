package example

import (
	"server/api/pb"
	"server/pkg/gnet/router"
)

func init() {
	router.On(OnTest)
}

func OnTest(req *pb.S2SEcho) {
}
