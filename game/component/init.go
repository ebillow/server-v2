package component

import (
	"server/game/component/debug"
	"server/game/component/example"
	"server/game/role"
	"server/pkg/pb"
)

// CreateComps 一些初始化操作,不能有数据处理,否则会被覆盖
func CreateComps(r *role.Role) {
	r.Comps[pb.TypeComp_TCExample] = example.New(r)           // 示例，有数据存储
	r.Comps[pb.TypeComp_TCEmptyExample] = example.NewEmpty(r) // 示例，没有数据存储
	r.Comps[pb.TypeComp_TCDebug] = debug.New(r)               // 测试用
}
