package robot

import (
	"server/pkg/util"
	"sync"
	"time"
)

var (
	Robots sync.Map
)

func InitRobots(cnt int, bg int) {
	if Setup.LoginOnly {
		for i := Setup.WorldBegin; i <= Setup.WorldEnd; i++ {
			Robots.Store(i, false)
			go NewUnitRobot(int(i), i)
		}
		return
	}
	LoadCfg()
	for i := 0; i != cnt; i++ {
		id := bg + i
		go NewUnitRobot(id, Setup.WorldBegin)
		if id%100 == 0 {
			time.Sleep(time.Microsecond * time.Duration(util.RandRange(10000, 20000)))
		}
	}
	Start()
}

func LoadCfg() {
	// cfg_load.Reload("../manage/configs")
}
