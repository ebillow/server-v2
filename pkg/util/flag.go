package util

// Flag flag操作
type Flag struct {
	v int
}

// Add 添加flag标志
func (f *Flag) Add(flag int) {
	f.v |= flag
}

// Has 判断是否有flag标志
func (f *Flag) Has(flag int) bool {
	return f.v&flag != 0
}

// Del 删除flag标志
func (f *Flag) Del(flag int) {
	f.v &= ^flag
}
