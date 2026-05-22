package component

import (
	"server/api/pb"
	"server/internal/game/component/debug"
	example2 "server/internal/game/component/example"
	"server/internal/game/role"
)

var Create create

type create struct {
}

// Create 创建组件，初始化结构,不能有数据处理,否则会被覆盖
func (*create) Create(r *role.Role) {
	r.Comps[pb.TypeComp_TCExample] = example2.New(r)           // 示例，有数据存储
	r.Comps[pb.TypeComp_TCEmptyExample] = example2.NewEmpty(r) // 示例，没有数据存储
	r.Comps[pb.TypeComp_TCDebug] = debug.New(r)                // 测试用
}
