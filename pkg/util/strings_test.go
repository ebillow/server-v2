package util

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIToString(t *testing.T) {
	if got := IToString(int(-123)); got != "-123" {
		t.Errorf("IToString(-123) = %v, want -123", got)
	}
	if got := IToString(uint64(18446744073709551615)); got != "18446744073709551615" {
		t.Errorf("IToString(MaxUint64) = %v, want 18446744073709551615", got)
	}
	if got := IToString(int8(0)); got != "0" {
		t.Errorf("IToString(0) = %v, want 0", got)
	}
}

func TestFToString(t *testing.T) {
	if got := FToString(3.1415926); got != "3.1415926" {
		t.Errorf("FToString(3.1415926) = %v, want 3.1415926", got)
	}
	if got := FToString(float32(1.5)); got != "1.5" {
		t.Errorf("FToString(1.5) = %v, want 1.5", got)
	}
}

func TestParseInt(t *testing.T) {
	t.Run("Valid Signed", func(t *testing.T) {
		got, err := ParseInt[int64]("-123456")
		if err != nil || got != -123456 {
			t.Errorf("ParseInt() got = %v, err = %v", got, err)
		}
	})
	t.Run("Invalid Signed", func(t *testing.T) {
		_, err := ParseInt[int]("abc")
		if err == nil {
			t.Errorf("ParseInt() expected error for invalid string")
		}
	})
}

func TestParseUint(t *testing.T) {
	t.Run("Valid Unsigned Max", func(t *testing.T) {
		got, err := ParseUint[uint64]("18446744073709551615")
		if err != nil || got != math.MaxUint64 {
			t.Errorf("ParseUint() got = %v, err = %v", got, err)
		}
	})
	t.Run("Invalid Unsigned Negative", func(t *testing.T) {
		_, err := ParseUint[uint64]("-1")
		if err == nil {
			t.Errorf("ParseUint() expected error for negative string")
		}
	})
}

func TestParseIntDef(t *testing.T) {
	// 成功解析
	if got := ParseIntDef("100", int(200)); got != 100 {
		t.Errorf("ParseIntDef(100) = %v, want 100", got)
	}
	// 解析失败，返回指定默认值
	if got := ParseIntDef("abc", int64(200)); got != 200 {
		t.Errorf("ParseIntDef(abc, 200) = %v, want 200", got)
	}
	// 解析失败，返回零值
	if got := ParseIntDef[int]("abc"); got != 0 {
		t.Errorf("ParseIntDef(abc) = %v, want 0", got)
	}
}

func TestParseUintDef(t *testing.T) {
	if got := ParseUintDef("18446744073709551615", uint64(0)); got != math.MaxUint64 {
		t.Errorf("ParseUintDef(MaxUint64) = %v, want %v", got, uint64(math.MaxUint64))
	}
	if got := ParseUintDef("-1", uint(10)); got != 10 {
		t.Errorf("ParseUintDef(-1, 10) = %v, want 10", got)
	}
}

func TestParseFloatDef(t *testing.T) {
	if got := ParseFloatDef("3.14", float64(0)); got != 3.14 {
		t.Errorf("ParseFloatDef(3.14) = %v, want 3.14", got)
	}
	if got := ParseFloatDef("invalid", float32(1.23)); got != 1.23 {
		t.Errorf("ParseFloatDef(invalid, 1.23) = %v, want 1.23", got)
	}
}

func TestParseBoolDef(t *testing.T) {
	tests := []struct {
		input string
		def   []bool
		want  bool
	}{
		{"true", nil, true},
		{"1", nil, true},
		{"false", nil, false},
		{"", nil, false},
		{"", []bool{true}, true},
		{"invalid", nil, false},
		{"invalid", []bool{true}, true},
	}

	for _, tt := range tests {
		got := ParseBoolDef(tt.input, tt.def...)
		if got != tt.want {
			t.Errorf("ParseBoolDef(%q, %v) = %v, want %v", tt.input, tt.def, got, tt.want)
		}
	}
}
func TestItoString(t *testing.T) {
	i := 12312312312312313
	str := IToString(i)
	require.Equal(t, "12312312312312313", str)

	i = 2
	str = IToString(i)
	require.Equal(t, "2", str)

	i2 := uint64(math.MaxUint64)
	str = IToString(i2)
	require.Equal(t, "18446744073709551615", str)

	ii := -1231231231233
	str = IToString(ii)
	require.Equal(t, "-1231231231233", str)
}

func TestFtoString(t *testing.T) {
	f := 123123123123.12312
	str := FToString(f)
	require.Equal(t, "123123123123.12312", str)

	f = 1.12312231231
	str = FToString(f)
	require.Equal(t, "1.12312231231", str)

	f = -1.12312231231
	str = FToString(f)
	require.Equal(t, "-1.12312231231", str)
}
