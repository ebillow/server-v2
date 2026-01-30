package util

import "strconv"

func ParseUint32(s string) uint32 {
	if ret, err := strconv.ParseUint(s, 10, 32); err == nil {
		return uint32(ret)
	} else {
		return uint32(0)
	}
}

func ParseInt32(s string) int32 {
	if ret, err := strconv.ParseInt(s, 10, 32); err == nil {
		return int32(ret)
	} else {
		return 0
	}
}

func ParseFloat32(s string) float32 {
	if ret, err := strconv.ParseFloat(s, 32); err == nil {
		return float32(ret)
	} else {
		return 0
	}
}

func ParseBool(s string) bool {
	if ret, err := strconv.ParseBool(s); err == nil {
		return ret
	} else {
		return false
	}
}

func ParseUint64(s string) uint64 {
	if ret, err := strconv.ParseUint(s, 10, 64); err == nil {
		return ret
	} else {
		return 0
	}
}

func ParseInt64(s string) int64 {
	if ret, err := strconv.ParseInt(s, 10, 64); err == nil {
		return ret
	} else {
		return 0
	}
}

func ParseFloat64(s string) float64 {
	if ret, err := strconv.ParseFloat(s, 64); err == nil {
		return ret
	} else {
		return 0
	}
}
