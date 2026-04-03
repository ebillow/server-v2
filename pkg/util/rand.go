package util

import (
	"fmt"
	"golang.org/x/exp/constraints"
	"math"
	"math/rand/v2"
)

type data[T any] struct {
	Weight int32
	Value  T
}

// RandByWeight 按权重随机---------------------------
type RandByWeight[T any] struct {
	datas []data[T]
	max   int64
}

func NewRandByWeight[T any]() *RandByWeight[T] {
	r := &RandByWeight[T]{
		datas: make([]data[T], 0),
	}
	return r
}

func (r *RandByWeight[T]) Valid() bool {
	return len(r.datas) > 0 && r.max > 0
}

func (r *RandByWeight[T]) Clone() *RandByWeight[T] {
	other := &RandByWeight[T]{
		datas: make([]data[T], len(r.datas)),
		max:   r.max,
	}
	copy(other.datas, r.datas)
	return other
}

func (r *RandByWeight[T]) Add(weight int32, v T) {
	if weight <= 0 {
		return
	}
	r.datas = append(r.datas, data[T]{
		Weight: weight,
		Value:  v,
	})
	r.max += int64(weight)
}

func (r *RandByWeight[T]) Get() (T, error) {
	var zero T
	if !r.Valid() {
		return zero, fmt.Errorf("randByWeight: not valid")
	}
	rate := rand.Int64N(r.max)
	cr := int64(0)

	for _, v := range r.datas {
		cr += int64(v.Weight)
		if cr > rate {
			return v.Value, nil
		}
	}
	return zero, fmt.Errorf("RandByWeight empty")
}

// GetAndDelete	获取并删除，保证只获取一次。注意会修改RandByWeight数据。
func (r *RandByWeight[T]) GetAndDelete() (T, error) {
	var zero T
	if !r.Valid() {
		var zero T
		return zero, fmt.Errorf("randByWeight: not valid")
	}

	rate := rand.Int64N(r.max)
	cr := int64(0)

	for i, v := range r.datas {
		cr += int64(v.Weight)
		if cr > rate {
			r.max -= int64(r.datas[i].Weight)
			lastIdx := len(r.datas) - 1
			r.datas[i] = r.datas[lastIdx]
			r.datas[lastIdx] = data[T]{}
			r.datas = r.datas[:lastIdx]
			return v.Value, nil
		}
	}
	return zero, fmt.Errorf("RandByWeight empty")
}

// RandUnique 在一组数中随机，每次结果不重复------------------------------
type RandUnique[T any] struct {
	data []T
}

func NewRandUnique[T any](values ...T) *RandUnique[T] {
	clone := make([]T, len(values))
	copy(clone, values)

	return &RandUnique[T]{
		data: clone,
	}
}
func (r *RandUnique[T]) Add(v T) {
	r.data = append(r.data, v)
}
func (r *RandUnique[T]) Get() (ret T, err error) {
	var zero T
	n := len(r.data)

	if n == 0 {
		return zero, fmt.Errorf("RandUnique empty")
	}
	cur := rand.IntN(n)
	ret = r.data[cur]

	lastIdx := n - 1
	r.data[cur] = r.data[lastIdx]
	r.data[lastIdx] = zero
	r.data = r.data[:lastIdx]

	return ret, nil
}
func (r *RandUnique[T]) Reset(values ...T) {
	r.data = append(r.data[:0], values...)
}

// Remain 返回剩余可选数量
func (r *RandUnique[T]) Remain() int {
	return len(r.data)
}

// ------------------------------随机函数------------------------------------

// Happen 万分率随机[0,n)
func Happen[T constraints.Integer](v T) bool {
	if v <= 0 {
		return false
	}

	val := uint64(v)
	const MaxRate = 10000

	if val >= MaxRate {
		return true
	}

	return rand.N(uint64(MaxRate)) < val
}

// RandRange 在[min, max)，右开区间随机
func RandRange[T constraints.Integer](min, max T) T {
	if min == max {
		return min
	}
	if min > max {
		min, max = max, min
	}
	diff := uint64(max) - uint64(min)
	return T(uint64(min) + rand.N(diff))
}

// RandRangeInc 在[min, max],闭区间随机
func RandRangeInc[T constraints.Integer](min, max T) T {
	if min == max {
		return min
	}
	if min > max {
		min, max = max, min
	}
	diff := uint64(max) - uint64(min)
	if diff == math.MaxUint64 {
		return T(rand.Uint64()) // 直接随机整个 uint64 空间即可
	}

	return T(uint64(min) + rand.N(diff+1))
}
