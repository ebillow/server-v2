package logic

import "server/game/role"

// 一些初始化操作,不能有数据处理,否则merge会覆盖部分加载出来的数据
func CreateComps(r *role.Role) {
	// r.Comps[pb.TypeComp_TCBuilding] = city.NewData(r)
}
