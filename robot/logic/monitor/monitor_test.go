package monitor

import (
	"server/pkg/util"
	"testing"
	"time"
)

func TestMonitor(t *testing.T) {
	Start()
	out := time.After(time.Minute * 5)
	tk := time.NewTicker(time.Millisecond * 10)
	for {
		select {
		case <-tk.C:
			Add(util.RandRange(300, time.Millisecond*3000))
		case <-out:
			return
		}
	}
}
