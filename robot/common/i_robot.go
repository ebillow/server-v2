package common

import (
	"google.golang.org/protobuf/proto"
	"server/pkg/pb"
	"server/pkg/pb/msgid"
)

type IRobot interface {
	GetData() *pb.RoleData                        // 获取玩家数据
	Send(msgId msgid.MsgIDC2S, msg proto.Message) // 发送消息
}
