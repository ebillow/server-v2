package global

import (
	"server/pkg/share/app"
	"time"
)

var _ app.ISubActor = &Global{}

type Global struct {
}

func (e *Global) Init() error {
	return nil
}

func (e *Global) OnTick(now time.Time) {

}

func (e *Global) Exit() {

}
