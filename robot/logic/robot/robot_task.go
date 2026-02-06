package robot

import (
	"server/pkg/pb"
	"server/pkg/pb/msgid"
	"server/pkg/util"
	"time"
)

func InitTask(r *Robot) {
	r.taskMgr = NewTimeEvter()
	r.taskMgr.Add(20, heartbeat) //

	InitEcho(r)
}

func TaskRun(r *Robot) {
	now := time.Now()
	if now.Sub(r.lastActTime).Seconds() > (time.Second * 5).Seconds() {
		r.lastActTime = now
		r.heartBeat(now)
	}
	if !Setup.LoginOnly {
		r.taskMgr.Run(r)
	}
}

func heartbeat(r *Robot) {
	msg := pb.C2SHeartBeat{
		CliTime: util.GetNowTimeM(),
	}
	r.Send(msgid.MsgIDC2S_C2SHeartBeat, &msg)
}
