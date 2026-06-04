package example

import (
	"server/internal/game/role"
	"time"
)

type Data struct {
	OnlineCnt  int32
	OfflineCnt int32
	Today      bool
	Name       string
	Award      map[int32]struct{}
	Info       []int32
	Cnt        int64

	cur   int32
	tmp   map[int32]bool
	dirty bool
}

func New(r *role.Role) *Data {
	return &Data{
		Award: make(map[int32]struct{}),
		Info:  make([]int32, 0),
		tmp:   make(map[int32]bool),
	}
}

func (d *Data) IsDirty() bool {
	return d.dirty
}

func (d *Data) ClearDirty() {
	d.dirty = false
}

func (d *Data) Online(r *role.Role) {
	d.OnlineCnt++
	d.cur = d.OnlineCnt
	d.Today = true
	d.Name = "testName"
	d.Award[d.cur] = struct{}{}
	d.Info = append(d.Info, d.cur)
	d.tmp[d.cur] = true
}

func (d *Data) Offline(r *role.Role) {
	d.OfflineCnt++
	if d.OnlineCnt != d.OfflineCnt {
		panic("d.OnlineCnt != d.OfflineCnt")
	}
}

func (d *Data) OnTick(now time.Time, r *role.Role) {
	d.Cnt++
	// d.dirty = true
}
