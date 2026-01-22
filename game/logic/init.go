package logic

import (
	"server/game/logic/example"
	"server/game/role"
	"server/internal/pb"
)

// 一些初始化操作,不能有数据处理,否则merge会覆盖部分加载出来的数据
func CreateComps(r *role.Role) {
	r.Comps[pb.TypeComp_TCExample] = example.New(r)
	r.Comps[pb.TypeComp_TCEmptyExample] = example.NewEmpty(r)
}
