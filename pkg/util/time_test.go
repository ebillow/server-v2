package util

import (
	// "github.com/jinzhu/now"
	"testing"
	"time"
)

func TestNow(t *testing.T) {
	// tb := now.BeginningOfDay()
	// t.Log(tb)
	// tmrb := now.BeginningOfDay().Add(time.Hour * 24)
	// t.Log("下一天", tmrb)
}

func TestTime(t *testing.T) {
	month := []int64{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	for i := 0; i < 11; i++ {
		month[i+1] = month[i] + month[i+1]
	}
	bt := time.Date(2021, 1, 1, 2, 0, 0, 0, time.Local)
	bd := GetZeroDay(bt.Unix(), 7200)
	bw := GetZeroWeek(bt.Unix(), 7200)
	bm := GetZeroMonth(bt.Unix(), 7200)
	for i := int64(0); i < 1000; i++ { // 1000天
		abt := bt.AddDate(0, 0, int(i))
		abtU := abt.Unix()
		for j := int64(0); j < 82800; j++ { // 每天24*3600秒
			abd := GetZeroDay(abtU+j, 7200)
			if abd != bd+i {
				t.Fail()
			}
			abw := GetZeroWeek(abtU+j, 7200)
			if i < 3 {
				if abw != bw {
					t.Fail()
				}
			} else {
				if abw != bw+(i-3)/7+1 {
					t.Fail()
				}
			}
			abm := GetZeroMonth(abtU+j, 7200)
			for k, v := range month {
				if i < v {
					if abm != bm+int64(k) {
						t.Fail()
					}
					break
				}
			}
		}
	}

}
