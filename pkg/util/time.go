package util

import (
	"fmt"
	"time"
)

const TimeLayout = "2006-01-02 15:04:05"
const gTimeInterval = (int64)(24 * 3600)

var gTimeGenesisUnix = time.Date(1970, 1, 1, 0, 0, 0, 0, time.Local).Unix()
var gTimeGenesisUnixWeek = time.Date(1970, 1, 5, 0, 0, 0, 0, time.Local).Unix()

// 获取今天凌晨0点的时间
func CurDayBegin() *time.Time {
	t := time.Now()
	zero_t := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	return &zero_t
}

// 获取今天凌晨0点的时间戳
func CurDayBeginUnix() int64 {
	t := CurDayBegin()
	return t.Unix()
}

// 当前时间的字符串 精确到秒
func CurDayString() string {
	t := time.Now()
	return t.Format("20060102150405")
}

// 当前时间的字符串 精确到纳秒
func CurDayStringNano() string {
	t := time.Now()
	return fmt.Sprintf("%d%02d%02d%02d%02d%02d%d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond())
}

// 当前时间的毫秒数(mill)
func GetNowTimeM() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// 当前时间的秒数(sec)
func GetNowTimeS() int64 {
	return time.Now().Unix()
}

func NowTimeString() string {
	return time.Now().Format(TimeLayout)
}

func UnixTimeString(unix int64) string {
	return time.Unix(unix, 0).Format(TimeLayout)
}

func UnixTimeString2(unix int64) string {
	return time.Unix(unix, 0).Format("2006-01-02 15:04:05.000")
}

// 获取相对1970年1月1日的天数
func GetZeroDay(time int64, resetTime int64) int64 {
	return (time - gTimeGenesisUnix - resetTime) / gTimeInterval
}

// 获取相对1970年1月1日的周数
func GetZeroWeek(time int64, resetTime int64) int64 {
	return (time - gTimeGenesisUnixWeek - resetTime) / gTimeInterval / 7
}

// 获取相对1970年1月1日的月数
func GetZeroMonth(unix int64, resetTime int64) int64 {
	now := time.Unix(unix-resetTime, 0)
	ny, nm, _ := now.Date()
	return (int64(ny)-1970)*12 + int64(nm) - 1
}

func IsSameDay(a, b, resetTime int64) bool {
	if a == b {
		return true
	}
	ad := GetZeroDay(a, resetTime)
	if ad < 0 {
		return false
	}
	bd := GetZeroDay(b, resetTime)
	if bd < 0 {
		return false
	}
	return ad == bd
}

func GetResetTime(nowTime, resetType, resetHour, resetDay int64) int64 {
	resetTime := resetHour * 3600
	if resetType == 2 || resetType == 3 {
		resetTime += (resetDay - 1) * 24 * 3600
	}
	now := time.Unix(nowTime, 0)
	now = now.Add(time.Duration(-resetTime) * time.Second)
	ly, lm, ld := now.Date()
	resetTimeDate := time.Date(ly, lm, ld, 0, 0, 0, 0, time.Local)
	switch resetType {
	case 1:
		resetTimeDate = resetTimeDate.AddDate(0, 0, 1)
	case 2:
		lw := int64(resetTimeDate.Weekday())
		if lw == 0 {
			lw = 7
		}
		resetTimeDate = resetTimeDate.AddDate(0, 0, int(7+1-lw))
	case 3:
		resetTimeDate = resetTimeDate.AddDate(0, 1, 1-ld)
	}
	return resetTimeDate.Unix() + resetTime
}

func ParseInLocation(dt string) (ret time.Time, err error) {
	return time.ParseInLocation(TimeLayout, dt, time.Local)
}
