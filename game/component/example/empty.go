package example

import "server/game/role"

type Empty struct {
	cur int32
	tmp map[int32]bool
}

func NewEmpty(r *role.Role) *Empty {
	return &Empty{
		tmp: make(map[int32]bool),
	}
}
