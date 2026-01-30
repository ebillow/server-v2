package logic

import (
	"go.uber.org/zap"
	"server/pkg/logger"
	"server/pkg/util"
	"strings"
	"time"
)

var (
	m = NewMonitor("game")
	// world = NewMonitor("world")
	// gate  = NewMonitor("gate")
)

func Active(id uint64) {
	m.Active(id)
}

func TimeOut(id uint32) {
	m.TimeOut(id)
}

//
// func WorldTimeOut(id uint32) {
//	world.TimeOut(id)
// }
//
// func WorldSendCnt(id uint64) {
//	world.SendCnt(id)
// }
//
// func WorldRecvCnt(id uint64) {
//	world.RecvCnt(id)
// }

func GateTimeOut(id uint32) {
	// gate.TimeOut(id)
}

type Monitor struct {
	c      chan uint64
	onLine map[uint64]time.Time

	toc     chan uint32
	timeOut map[uint32]uint32
	name    string

	msgSend map[uint64]int
	cSend   chan uint64
	msgRecv map[uint64]int
	cRecv   chan uint64
}

func NewMonitor(name string) *Monitor {
	m := &Monitor{
		c:       make(chan uint64, 1000),
		onLine:  make(map[uint64]time.Time),
		name:    name,
		toc:     make(chan uint32, 3000),
		timeOut: make(map[uint32]uint32),
		cSend:   make(chan uint64, 3000),
		cRecv:   make(chan uint64, 3000),
		msgRecv: make(map[uint64]int),
		msgSend: make(map[uint64]int),
	}

	go m.run()
	return m
}

func (m *Monitor) TimeOut(id uint32) {
	m.toc <- id
}

func (m *Monitor) Active(id uint64) {
	m.c <- id
}

func (m *Monitor) SendCnt(id uint64) {
	m.cSend <- id
}

func (m *Monitor) RecvCnt(id uint64) {
	m.cRecv <- id
}

func (m *Monitor) run() {
	t := time.NewTicker(time.Second * 60)
	tLogin := time.NewTicker(time.Second * 10)
	defer func() {
		t.Stop()
		tLogin.Stop()
	}()
	now := time.Now()
	for {
		now = time.Now()
		select {
		case id := <-m.c:
			m.onLine[id] = now
		case id := <-m.toc:
			m.timeOut[id]++
		case id := <-m.cRecv:
			m.msgRecv[id]++
		case id := <-m.cSend:
			m.msgSend[id]++
		case <-tLogin.C:
			if Setup.LoginOnly {
				success := 0
				Robots.Range(func(key, value any) bool {
					if value.(bool) == true {
						success++
					}
					return true
				})
				total := Setup.WorldEnd - Setup.WorldBegin + 1
				zap.S().Infof("登录测试：[%d->%d] 共%d个服, 成功:%d个服", Setup.WorldBegin, Setup.WorldEnd, total, success)
				if success < int(total) {
					ss := strings.Builder{}
					ss.WriteString("还未登录成功:")
					Robots.Range(func(key, value any) bool {
						if value.(bool) == false {
							ss.WriteString(" ")
							ss.WriteString(util.ToString(key))
							ss.WriteString("服,")
						}
						return true
					})
					logger.Info(ss.String())
				}
			}
		case <-t.C:
			if !Setup.LoginOnly {
				cnt := 0
				for k, v := range m.timeOut {
					if v > 5 {
						cnt++
					}
					m.timeOut[k] = 0
				}

				sendCnt := 0
				if len(m.msgSend) > 0 {
					for _, v := range m.msgSend {
						sendCnt += v
					}
					sendCnt = sendCnt / len(m.msgSend)
				}
				recvCnt := 0
				if len(m.msgRecv) > 0 {
					for _, v := range m.msgRecv {
						recvCnt += v
					}
					recvCnt = recvCnt / len(m.msgRecv)
				}
				logger.Infof("%s active player cnt:%d time out cnt:%d SendPB:%d Recv:%d\n", m.name, len(m.onLine), cnt, sendCnt, recvCnt)
				m.msgSend = make(map[uint64]int)
				m.msgRecv = make(map[uint64]int)
				m.onLine = make(map[uint64]time.Time)
			}
		}
	}
}
