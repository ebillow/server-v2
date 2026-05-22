package common

import (
	"server/api/pb"
	"server/api/pb/msgid"

	"google.golang.org/protobuf/proto"
)

type IRobot interface {
	GetData() *pb.RoleData                        // 获取玩家数据
	Send(msgId msgid.MsgIDC2S, msg proto.Message) // 发送消息
}
