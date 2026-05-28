package msgq

import (
	"server/api/pb"
	"server/pkg/gnet/gctx"
)

// 发送函数 会接管 data 的生命周期，调用结束后请勿修改 data

// Send 指定发送
func (bs *DataBus) Send(serType pb.Server, serID uint8, msgID uint32, data []byte, actorID uint64, sesID uint64) error {
	if bs.closed.Load() {
		return ErrClosed
	}
	pbt, err := bs.getIdxPubBatcher(serType, serID)
	if err != nil {
		return err
	}
	return pbt.Add(gctx.Context{
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

func (bs *DataBus) ForwardToRole(serType pb.Server, serID uint8, msgID uint32, data []byte, actorID uint64, sesID uint64) error {
	if bs.closed.Load() {
		return ErrClosed
	}

	pbt, err := bs.getIdxPubBatcher(serType, serID)
	if err != nil {
		return err
	}
	return pbt.Add(gctx.Context{
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
func (bs *DataBus) SendAny(serType pb.Server, msgID uint32, data []byte, actorID uint64, sesID uint64) error {
	if bs.closed.Load() {
		return ErrClosed
	}

	pbt, err := bs.getGroupPubBatcher(serType)
	if err != nil {
		return err
	}

	return pbt.Add(gctx.Context{
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
func (bs *DataBus) SendAll(serType pb.Server, msgID uint32, data []byte, actorID uint64, sesID uint64) error {
	if bs.closed.Load() {
		return ErrClosed
	}

	pbt, err := bs.getAllPubBatcher(serType)
	if err != nil {
		return err
	}

	return pbt.Add(gctx.Context{
		MsgID:     msgID,
		Data:      data,
		FromSer:   bs.serType,
		FromSerID: bs.serID,
		ToSer:     uint8(serType),
		ActorID:   actorID,
		SesID:     sesID,
	})
}

// Relay 转发给指定服
func (bs *DataBus) Relay(serType pb.Server, serID uint8, msgID uint32, data []byte, actorID uint64, sesID uint64) error {
	if bs.closed.Load() {
		return ErrClosed
	}

	pbt, err := bs.getGroupPubBatcher(pb.Server_Center)
	if err != nil {
		return err
	}

	return pbt.Add(gctx.Context{
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
