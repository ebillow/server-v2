package global

import (
	"server/internal/share/actor"
	"time"
)

var _ actor.ISubActor = &Global{}

type Global struct {
}

func (e *Global) Init() error {
	return nil
}

func (e *Global) OnTick(now time.Time) {

}

func (e *Global) Exit() {

}
