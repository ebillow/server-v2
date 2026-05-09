package util

import (
	"strconv"

	"golang.org/x/exp/constraints"
)

// IToString 整数转字符串
func IToString[T constraints.Integer](n T) string {
	if n < 0 {
		return strconv.FormatInt(int64(n), 10)
	}
	return strconv.FormatUint(uint64(n), 10)
}

// FToString 浮点数转字符串，避免Sprint写成科学计数，并且小数位数刚好够
func FToString[T constraints.Float](n T) string {
	return strconv.FormatFloat(float64(n), 'f', -1, 64)
}

// ParseFloat 字符串转浮点数
func ParseFloat[T constraints.Float](s string) (T, error) {
	n, err := strconv.ParseFloat(s, 64)
	return T(n), err
}

// ParseInt 字符串转整数
// 不经过 float64 转换，避免大整数（如雪花 ID）精度丢失
func ParseInt[T constraints.Integer](s string) (T, error) {
	n, err := strconv.ParseInt(s, 10, 64)
	return T(n), err
}

// ParseBool 字符串转布尔
func ParseBool(s string) (bool, error) {
	return strconv.ParseBool(s)
}

// ParseIntDef 字符串转整数，失败返回默认值
func ParseIntDef[T constraints.Integer](s string, def T) T {
	if s == "" {
		return def
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return def
	}
	return T(n)
}

// ParseFloatDef 字符串转浮点数，失败返回默认值
func ParseFloatDef[T constraints.Float](s string, def T) T {
	if s == "" {
		return def
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return T(n)
}

// ParseBoolDef 字符串转布尔，失败返回默认值
func ParseBoolDef(s string, def bool) bool {
	if s == "" {
		return def
	}
	v, err := strconv.ParseBool(s)
	if err != nil {
		return def
	}
	return v
}
