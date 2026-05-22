package example

import (
	"server/internal/share/actor"
	"time"
)

var _ actor.ISubActor = &Example{}

type Example struct {
}

func (e *Example) Init() error {
	return nil
}

func (e *Example) OnTick(now time.Time) {

}

func (e *Example) Exit() {

}
