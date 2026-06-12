package onlines

import (
	"server/api/pb"
	"server/pkg/gnet/pkg"
	"server/pkg/gnet/router"
	"sync"
)

const shardCount = 640

var (
	roleShards = make([]*roleShard, shardCount)
)

type Data struct {
	SesID  uint64
	GameID uint8
}
type roleShard struct {
	mtx   sync.RWMutex
	roles map[uint64]Data
}

func init() {
	router.On(onLoginOrLogout)

	for i := 0; i < shardCount; i++ {
		roleShards[i] = &roleShard{
			roles: make(map[uint64]Data),
		}
	}
}
func getRoleShard(roleID uint64) *roleShard {
	return roleShards[roleID&(shardCount-1)]
}

func Add(roleID uint64, data Data) {
	rs := getRoleShard(roleID)
	rs.mtx.Lock()
	rs.roles[roleID] = data
	rs.mtx.Unlock()
}

func Remove(roleID uint64) {
	rs := getRoleShard(roleID)
	rs.mtx.Lock()
	delete(rs.roles, roleID)
	rs.mtx.Unlock()
}

func GetGameID(roleID uint64) (uint8, bool) {
	rs := getRoleShard(roleID)
	rs.mtx.RLock()
	defer rs.mtx.RUnlock()

	r, ok := rs.roles[roleID]
	if ok {
		return r.GameID, ok
	}
	return 0, false
}

func Count() int {
	var count int
	for _, shard := range roleShards {
		shard.mtx.RLock()
		count += len(shard.roles)
		shard.mtx.RUnlock()
	}
	return count
}

func onLoginOrLogout(_ pkg.Head, req *pb.S2SReqLoginOrLogout) {
	if req.Login {
		Add(req.RoleID, Data{SesID: req.SesID, GameID: uint8(req.GameID)})
	} else {
		Remove(req.RoleID)
	}
}
