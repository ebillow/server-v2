package util

import (
	"math/rand"
)

const (
	MAX_RATE = 10000
)

type RanderData struct {
	Rate  uint32
	Value interface{}
}

// Rander add概率和interface，get会按概率随机返回interface
type Rander struct {
	Data []*RanderData
	max  int
}

func NewRander() *Rander {
	r := &Rander{
		Data: make([]*RanderData, 0),
	}
	return r
}

func (r *Rander) Valid() bool {
	return len(r.Data) > 0 && r.max > 0
}

func (r *Rander) Clone() *Rander {
	other := &Rander{
		Data: make([]*RanderData, len(r.Data)),
		max:  r.max,
	}
	for i, v := range r.Data {
		other.Data[i] = &RanderData{
			Rate:  v.Rate,
			Value: v.Value,
		}
	}
	return other
}

func (r *Rander) Add(rate uint32, v interface{}) {
	d := &RanderData{
		Rate:  rate,
		Value: v,
	}
	r.Data = append(r.Data, d)
	r.max += int(rate)
}

func (r *Rander) Get() interface{} {
	rate := uint32(rand.Intn(r.max))
	cr := uint32(0)
	var ret interface{}
	for _, v := range r.Data {
		cr += v.Rate
		ret = v.Value
		if cr > rate {
			return ret
		}
	}
	return ret
}

// GetAndDelete	获取并删除，保证只获取一次。注意会修改rander数据。一般先要clone
func (r *Rander) GetAndDelete() interface{} {
	rate := uint32(rand.Intn(r.max))
	cr := uint32(0)
	var ret interface{}
	for i, v := range r.Data {
		cr += v.Rate
		ret = v.Value
		if cr > rate {
			r.max -= int(r.Data[i].Rate)
			r.Data[i] = r.Data[len(r.Data)-1]
			r.Data = r.Data[:len(r.Data)-1]
			return ret
		}
	}
	return ret
}

/*------------------------------------------------
 */
//Rand 万分率随机[0,n)
func Rand(v int) bool {
	if v < 0 {
		return false
	} else if v >= 10000 {
		return true
	} else {
		return rand.Intn(10000) < v
	}
}

// RandRangeFloat 在[min, max)随机
func RandRangeFloat(min float64, max float64) float64 {
	if min == max {
		return min
	}
	if min > max {
		min, max = max, min
	}
	return min + (max-min)*rand.Float64()
}

// RandRangeInt 在[min, max)随机
func RandRangeInt(min int, max int) int {
	return rand.Intn(max-min) + min
}

// RandRangeIntCloseInterval 在[min, max]随机
func RandRangeIntCloseInterval(min int, max int) int {
	return rand.Intn(max-min+1) + min
}

func RandInt(v int) int {
	if v == 0 {
		return 0
	}
	return rand.Intn(v)
}

func RandToken() uint32 {
	ret := uint32(0)
	for ret == 0 {
		ret = rand.Uint32()
	}
	return ret
}

// RandNotRepeated 在一组数中随机，每次结果不重复
type RandNotRepeated struct {
	data []uint64
	cnt  int
}

func NewRandNotRepeated(values ...uint64) *RandNotRepeated {
	r := &RandNotRepeated{data: values, cnt: len(values)}
	return r
}
func (r *RandNotRepeated) Add(v uint64) {
	r.data = append(r.data, v)
	r.cnt++
}
func (r *RandNotRepeated) Rand() (ret uint64, ok bool) {
	if r.cnt == 0 {
		return 0, false
	}
	cur := rand.Intn(r.cnt)
	ret = r.data[cur]
	r.data[cur] = r.data[r.cnt-1]
	r.cnt--
	return ret, true
}
