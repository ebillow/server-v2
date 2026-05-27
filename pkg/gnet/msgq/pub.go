package msgq

import (
	"server/api/pb"
	"server/pkg/gnet/gctx"
)

// Send 指定发送
func (bs *DataBus) Send(serType pb.Server, serID uint8, msgID uint32, data []byte, actorID uint64, sesID uint64) {
	bs.getIdxPubBatcher(serType, serID).Add(gctx.Context{
		MsgID:     msgID,
		Data:      data,
		FromSerID: bs.serID,
		FromSer:   bs.serType,
		ToSer:     uint8(serType),
		ToSerID:   serID,
		ActorID:   actorID,
		SesID:     sesID,
	})
}

func (bs *DataBus) ForwardToRole(serType pb.Server, serID uint8, msgID uint32, data []byte, actorID uint64, sesID uint64) {
	bs.getIdxPubBatcher(serType, serID).Add(gctx.Context{
		MsgID:     msgID,
		Data:      data,
		FromSerID: bs.serID,
		FromSer:   bs.serType,
		ToSer:     uint8(serType),
		ToSerID:   serID,
		ActorID:   actorID,
		SesID:     sesID,
		Flag:      gctx.Forward,
	})
}

// SendAny 组发送. 随机一个能收到
func (bs *DataBus) SendAny(serType pb.Server, msgID uint32, data []byte, actorID uint64, sesID uint64) {
	bs.getGroupPubBatcher(serType).Add(gctx.Context{
		MsgID:     msgID,
		Data:      data,
		FromSer:   bs.serType,
		FromSerID: bs.serID,
		ToSer:     uint8(serType),
		ActorID:   actorID,
		SesID:     sesID,
	})
}

// SendAll 所有的 serName 服节点都能收到
func (bs *DataBus) SendAll(serType pb.Server, msgID uint32, data []byte, actorID uint64, sesID uint64) {
	bs.getAllPubBatcher(serType).Add(gctx.Context{
		MsgID:     msgID,
		Data:      data,
		FromSer:   bs.serType,
		FromSerID: bs.serID,
		ToSer:     uint8(serType),
		ActorID:   actorID,
		SesID:     sesID,
	})
}

func (bs *DataBus) Relay(serType pb.Server, serID uint8, msgID uint32, data []byte, actorID uint64, sesID uint64) {
	bs.getGroupPubBatcher(pb.Server_Center).Add(gctx.Context{
		MsgID:     msgID,
		Data:      data,
		FromSer:   bs.serType,
		FromSerID: bs.serID,
		ToSer:     uint8(serType),
		ToSerID:   serID,
		ActorID:   actorID,
		SesID:     sesID,
		Flag:      gctx.Forward,
	})
}
