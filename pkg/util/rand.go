package util

import (
	"fmt"
	"golang.org/x/exp/constraints"
	"math/rand/v2"
)

type data[T any] struct {
	Weight uint32
	Value  T
}

// RandByWeight 按权重随机---------------------------
type RandByWeight[T any] struct {
	datas []*data[T]
	max   int
}

func NewRandByWeight[T any]() *RandByWeight[T] {
	r := &RandByWeight[T]{
		datas: make([]*data[T], 0),
	}
	return r
}

func (r *RandByWeight[T]) Valid() bool {
	return len(r.datas) > 0 && r.max > 0
}

func (r *RandByWeight[T]) Clone() *RandByWeight[T] {
	other := &RandByWeight[T]{
		datas: make([]*data[T], len(r.datas)),
		max:   r.max,
	}
	for i, v := range r.datas {
		other.datas[i] = &data[T]{
			Weight: v.Weight,
			Value:  v.Value,
		}
	}
	return other
}

func (r *RandByWeight[T]) Add(weight uint32, v T) {
	r.datas = append(r.datas, &data[T]{
		Weight: weight,
		Value:  v,
	})
	r.max += int(weight)
}

func (r *RandByWeight[T]) Get() (T, error) {
	rate := uint32(rand.N(r.max))
	cr := uint32(0)
	var ret T
	for _, v := range r.datas {
		cr += v.Weight
		ret = v.Value
		if cr > rate {
			return ret, nil
		}
	}
	return ret, fmt.Errorf("RandByWeight empty")
}

// GetAndDelete	获取并删除，保证只获取一次。注意会修改RandByWeight数据。
func (r *RandByWeight[T]) GetAndDelete() (T, error) {
	rate := uint32(rand.N(r.max))
	cr := uint32(0)
	var ret T
	for i, v := range r.datas {
		cr += v.Weight
		ret = v.Value
		if cr > rate {
			r.max -= int(r.datas[i].Weight)
			r.datas[i] = r.datas[len(r.datas)-1]
			r.datas = r.datas[:len(r.datas)-1]
			return ret, nil
		}
	}
	return ret, fmt.Errorf("RandByWeight empty")
}

// RandUnique 在一组数中随机，每次结果不重复------------------------------
type RandUnique[T any] struct {
	data []T
	cnt  int
}

func NewRandUnique[T any](values ...T) *RandUnique[T] {
	r := &RandUnique[T]{data: values, cnt: len(values)}
	return r
}
func (r *RandUnique[T]) Add(v T) {
	r.data = append(r.data, v)
	r.cnt++
}
func (r *RandUnique[T]) Get() (ret T, err error) {
	if r.cnt == 0 {
		return ret, fmt.Errorf("RandUnique empty")
	}
	cur := rand.N(r.cnt)
	ret = r.data[cur]
	r.data[cur] = r.data[r.cnt-1]
	r.cnt--
	return ret, nil
}

// Happen 万分率随机[0,n)
func Happen(v int) bool {
	const MaxRate = 10000
	if v < 0 {
		return false
	} else if v >= MaxRate {
		return true
	} else {
		return rand.N(MaxRate) < v
	}
}

// RandRange 在[min, max)随机
func RandRange[T constraints.Integer](min, max T) T {
	if min == max {
		return min
	}
	if min > max {
		min, max = max, min
	}
	return min + rand.N(max-min)
}

// RandRangeIntCloseInterval 在[min, max]随机
func RandRangeIntCloseInterval[T constraints.Integer](min, max T) T {
	if min > max {
		min, max = max, min
	}
	return rand.N(max-min+1) + min
}
