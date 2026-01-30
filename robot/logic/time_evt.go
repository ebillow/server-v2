package logic

import (
	"time"
)

type Item struct {
	cb      func(*Robot)
	runTime int64
	period  int64
}

type TimeEvter struct {
	items []*Item
	now   int64
}

func NewTimeEvter() *TimeEvter {
	return &TimeEvter{
		items: make([]*Item, 0),
		now:   time.Now().Unix(),
	}
}

func (t *TimeEvter) Add(period int64, cb func(*Robot)) {
	t.items = append(t.items, &Item{cb: cb, runTime: t.now + period, period: period})
}

func (t *TimeEvter) Run(r *Robot) {
	t.now = time.Now().Unix()
	for _, v := range t.items {
		if v.runTime < t.now {
			v.cb(r)
			v.runTime += v.period
		}
	}
}
