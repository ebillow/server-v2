package robot

import (
	"server/pkg/util"
	"strings"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
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

func AddSendCnt() {
	m.AddSendCnt()
}

func AddRecvCnt() {
	m.AddRecvCnt()
}

type Monitor struct {
	c      chan uint64
	onLine map[uint64]time.Time

	toc     chan uint32
	timeOut map[uint32]uint32
	name    string

	sendCnt atomic.Uint64
	recvCnt atomic.Uint64
}

func NewMonitor(name string) *Monitor {
	m := &Monitor{
		c:       make(chan uint64, 1000),
		onLine:  make(map[uint64]time.Time),
		name:    name,
		toc:     make(chan uint32, 3000),
		timeOut: make(map[uint32]uint32),
	}

	return m
}

func Start() {
	go m.run()
}

func (m *Monitor) TimeOut(id uint32) {
	m.toc <- id
}

func (m *Monitor) Active(id uint64) {
	m.c <- id
}

func (m *Monitor) AddSendCnt() {
	m.sendCnt.Add(1)
}

func (m *Monitor) AddRecvCnt() {
	m.recvCnt.Add(1)
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
							ss.WriteString(util.ToString(key.(uint32)))
							ss.WriteString("服,")
						}
						return true
					})
					zap.S().Info(ss.String())
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
				zap.S().Infof("%s active:%d time out:%d send:%d recv:%d", m.name, len(m.onLine), cnt, m.sendCnt.Swap(0), m.recvCnt.Swap(0))
				m.onLine = make(map[uint64]time.Time)
			}
		}
	}
}
