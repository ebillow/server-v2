package clinet

import (
	"google.golang.org/protobuf/proto"
)

type EvtType int

const (
	EvtLoginSuccess EvtType = iota
)

type Evt struct {
	Type EvtType
	Data proto.Message
}

func (s *Session) PostEvt(evtType EvtType, param proto.Message) {
	evt := Evt{
		Type: evtType,
		Data: param,
	}
	s.evt <- evt
}

func (s *Session) onEvent(evt Evt) {

}
