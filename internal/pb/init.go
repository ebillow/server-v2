package pb

import (
	"fmt"
	"google.golang.org/protobuf/proto"
	"reflect"
	"server/internal/pb/msgid"
)

type messageMeta struct {
	MessageType reflect.Type
	NewMessage  func() proto.Message
}

type TypeMeta struct {
	idByType map[uint16]*messageMeta
	typeByID map[reflect.Type]uint16
}

func (t *TypeMeta) init() {
	t.idByType = make(map[uint16]*messageMeta)
	t.typeByID = make(map[reflect.Type]uint16)
}

func (t *TypeMeta) Register(id uint16, meta *messageMeta) {
	t.idByType[id] = meta
	t.typeByID[meta.MessageType] = id
}

func (t *TypeMeta) New(msgID uint16) proto.Message {
	if v, ok := t.idByType[msgID]; ok {
		return v.NewMessage()
	}
	return nil
}

func (t *TypeMeta) NewFunc(msgID uint16) func() proto.Message {
	if v, ok := t.idByType[msgID]; ok {
		return v.NewMessage
	}
	return nil
}

func (t *TypeMeta) MsgID(msg interface{}) (uint16, error) {
	itype := reflect.TypeOf(msg).Elem()
	if v, ok := t.typeByID[itype]; ok {
		return v, nil
	}

	return 0, fmt.Errorf("not found msg type %s", reflect.TypeOf(msg))
}

var (
	c2s TypeMeta
	s2c TypeMeta
	s2s TypeMeta
)

// 添加消息映射 c-s的
func init() {
	c2s.init()
	s2c.init()
	s2s.init()

	registerAllC2SMsg()
	registerAllC2SMsg()
	registerAllC2SMsg()
}

func registerC2SMsg(id msgid.MsgIDC2S, meta *messageMeta) {
	c2s.Register(uint16(id), meta)
}

func registerS2CMsg(id msgid.MsgIDS2C, meta *messageMeta) {
	s2c.Register(uint16(id), meta)
}

func registerS2SMsg(id msgid.MsgIDS2S, meta *messageMeta) {
	s2s.Register(uint16(id), meta)
}

func NewFuncC2S(msgID msgid.MsgIDC2S) func() proto.Message {
	return c2s.NewFunc(uint16(msgID))
}

func NewFuncS2C(msgID msgid.MsgIDS2C) func() proto.Message {
	return s2c.NewFunc(uint16(msgID))
}

func NewFuncS2S(msgID msgid.MsgIDS2S) func() proto.Message {
	return s2s.NewFunc(uint16(msgID))
}

func GetMsgIDC2S(msg proto.Message) (uint16, error) {
	return c2s.MsgID(msg)
}

func GetMsgIDS2C(msg proto.Message) (uint16, error) {
	return s2c.MsgID(msg)
}

func GetMsgIDS2S(msg proto.Message) (uint16, error) {
	return s2s.MsgID(msg)
}
