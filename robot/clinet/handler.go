package clinet

import (
	"errors"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"server/pkg/pb/msgid"
)

var (
	errAPINotRegist = errors.New("api not regist")
)

type msgHandler struct {
	createFunc func() proto.Message
	handleFunc func(msg proto.Message, s *Session)
}

// Route 消息处理器
type Route struct {
	handlers []*msgHandler
}

// NewRoute createRoute
func NewRoute(size int) *Route {
	r := &Route{}
	r.init(size)
	return r
}

// Register 注册消息
func (r *Route) Register(msgID uint32, cf func() proto.Message, df func(msg proto.Message, s *Session)) {
	n := &msgHandler{
		createFunc: cf,
		handleFunc: df,
	}
	r.handlers[msgID] = n
}

// Handle 处理消息
func (r *Route) Handle(id uint32, data []byte, s *Session) bool {
	node, err := r.getHandler(id)
	if err != nil {
		return false
	}

	msg, err := r.parseMsg(node, data)
	if err != nil {
		zap.S().Warnf("%s parser msg %d error:%v", s.String(), id, err)
		s.Close()
		return false
	}

	if id != uint32(msgid.MsgIDS2C_S2CHeartBeat) {
		zap.S().Debugf("%s recv [%d]%s msg:%s", s.String(), id, msgid.MsgIDS2C_name[int32(id)], msg)
	}

	node.handleFunc(msg, s)

	return true
}

func (r *Route) init(size int) {
	r.handlers = make([]*msgHandler, size)
}

func (r *Route) getHandler(id uint32) (n *msgHandler, err error) {
	if int(id) >= len(r.handlers) {
		err = errAPINotRegist
		return
	}

	n = r.handlers[id]
	if nil == n || nil == n.createFunc {
		err = errAPINotRegist
		return
	}
	return
}

func (r *Route) parseMsg(n *msgHandler, data []byte) (msg proto.Message, err error) {
	msg = n.createFunc()
	if msg == nil { // 允许只有消息id没内容
		return
	}
	err = proto.Unmarshal(data, msg)
	return
}
