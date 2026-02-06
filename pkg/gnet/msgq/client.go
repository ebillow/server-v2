package msgq

import "server/pkg/pb"

// Send 指定发送
func (bs *DataBus) Send(serName string, serID int32, msgID uint32, data []byte, roleID uint64, sesID uint64) error {
	out, err := encode(&pb.NatsMsg{
		MsgID:   msgID,
		Data:    data,
		SerID:   serID,
		SerType: bs.serType,
		RoleID:  roleID,
		SesID:   sesID,
		Forward: false,
	})
	if err != nil {
		return err
	}
	return bs.conn.Publish(getIndexSubject(serName, serID), out)
}

func (bs *DataBus) ForwardToRole(serName string, serID int32, msgID uint32, data []byte, roleID uint64, sesID uint64) error {
	out, err := encode(&pb.NatsMsg{
		MsgID:   msgID,
		Data:    data,
		SerID:   serID,
		SerType: bs.serType,
		RoleID:  roleID,
		SesID:   sesID,
		Forward: true,
	})
	if err != nil {
		return err
	}
	return bs.conn.Publish(getIndexSubject(serName, serID), out)
}

// SendAny 组发送. 随机一个能收到
func (bs *DataBus) SendAny(serName string, msgID uint32, data []byte, roleID uint64, sesID uint64) error {
	out, err := encode(&pb.NatsMsg{
		MsgID:   msgID,
		Data:    data,
		SerType: bs.serType,
		RoleID:  roleID,
		SesID:   sesID,
		Forward: false,
	})
	if err != nil {
		return err
	}
	return bs.conn.Publish(getGroupSubject(serName), out)
}

// SendAll 所有的 serName 服节点都能收到
func (bs *DataBus) SendAll(serName string, msgID uint32, data []byte, roleID uint64, sesID uint64) error {
	out, err := encode(&pb.NatsMsg{
		MsgID:   msgID,
		Data:    data,
		SerType: bs.serType,
		RoleID:  roleID,
		SesID:   sesID,
		Forward: false,
	})
	if err != nil {
		return err
	}
	return bs.conn.Publish(getAllSubject(serName), out)
}
