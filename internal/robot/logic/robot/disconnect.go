package robot

import (
	"server/pkg/util"
)

func InitDisconnect(r *Robot) {
	r.AddTask(int64(util.RandRange(600, 1800)), taskDisconnect)
}

func taskDisconnect(r *Robot) {
	r.s.Close()
}
