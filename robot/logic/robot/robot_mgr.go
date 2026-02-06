package robot

import (
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
		if id%500 == 0 {
			time.Sleep(time.Second)
		}
	}
}

func LoadCfg() {
	// cfg_load.Reload("../manage/configs")
}
