package common

import (
	"google.golang.org/protobuf/proto"
	"server/pkg/pb/msgid"
)

type MDType uint32

const (
	MdBattle MDType = 1
)

type IModule interface {
	Test()                                                     // 模块测试
	HandleMessage(msgID msgid.MsgIDS2C, message proto.Message) // 模块消息处理函数
}
