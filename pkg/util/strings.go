package util

import (
	"golang.org/x/exp/constraints"
	"strconv"
)

type number interface {
	constraints.Integer | constraints.Float
}

// ToStr 整数转字符串
func ToStr[T constraints.Integer](n T) string { return strconv.Itoa(int(n)) }

// FtoStr 浮点数转字符串
func FtoStr[T constraints.Float](n T) string {
	return strconv.FormatFloat(float64(n), 'f', -1, 64)
}

// Parse 字符串转数字
func Parse[T number](s string) T { n, _ := strconv.ParseFloat(s, 64); return T(n) }

func ParseBool(s string) bool {
	v, _ := strconv.ParseBool(s)
	return v
}
