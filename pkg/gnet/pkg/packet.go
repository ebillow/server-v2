package pkg

import (
	"encoding/binary"
	"server/api/pb"
	"server/pkg/flag"
	"server/pkg/gerror"
	"server/pkg/gnet/gmetrics"

	"go.uber.org/zap/zapcore"
)

const (
	Normal    = 0
	Forward   = 1
	Broadcast = 1 << 1
)

const FrameLenSize = 4
const FrameBodyHeadSize = 4 + 8 + 8 + 1 + 1 + 1 + 1 + 1

type Unity interface{}

type Head struct {
	ActorID   uint64
	SesID     uint64
	MsgID     uint32
	Flag      uint8
	FromSer   uint8
	FromSerID uint8
	ToSer     uint8
	ToSerID   uint8
}

func (head *Head) EncodeTo(dst []byte, bodySize int) {
	binary.LittleEndian.PutUint32(dst[0:], uint32(bodySize))
	binary.LittleEndian.PutUint32(dst[4:], head.MsgID)
	binary.LittleEndian.PutUint64(dst[8:], head.ActorID)
	binary.LittleEndian.PutUint64(dst[16:], head.SesID)
	dst[24] = head.FromSer
	dst[25] = head.FromSerID
	dst[26] = head.ToSer
	dst[27] = head.ToSerID
	dst[28] = head.Flag
}

func (head *Head) Decode(buf []byte) (err error) {
	if len(buf) < FrameBodyHeadSize {
		return gerror.New("decode error: buffer too small for header")
	}

	offset := 0

	head.MsgID = binary.LittleEndian.Uint32(buf[offset:])
	offset += 4

	head.ActorID = binary.LittleEndian.Uint64(buf[offset:])
	offset += 8

	head.SesID = binary.LittleEndian.Uint64(buf[offset:])
	offset += 8

	head.FromSer = buf[offset]
	offset += 1

	head.FromSerID = buf[offset]
	offset += 1

	head.ToSer = buf[offset]
	offset += 1

	head.ToSerID = buf[offset]
	offset += 1

	head.Flag = buf[offset]

	return nil
}

type Packet struct {
	Head  Head
	Data  []byte
	U     Unity
	reply string
}

func (s *Packet) SetReply(str string) {
	s.reply = str
}

func (s *Packet) Reply() string {
	return s.reply
}

func (s *Packet) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddUint32("msgID", s.Head.MsgID)
	encoder.AddUint64("actorID", s.Head.ActorID)
	encoder.AddUint64("sesID", s.Head.SesID)
	encoder.AddString("from", flag.SrvName(pb.Server(s.Head.FromSer)))
	encoder.AddUint8("fromID", s.Head.FromSerID)
	encoder.AddString("to", flag.SrvName(pb.Server(s.Head.ToSer)))
	encoder.AddUint8("toID", s.Head.ToSerID)
	return nil
}

func (s *Packet) EncodeTo(dst []byte, bodySize int) {
	s.Head.EncodeTo(dst, bodySize)
	copy(dst[FrameLenSize+FrameBodyHeadSize:], s.Data)
}

func (s *Packet) Decode(buf []byte) error {
	if len(buf) < FrameBodyHeadSize {
		return gerror.New("decode error: buffer too small for header")
	}

	err := s.Head.Decode(buf)
	if err != nil {
		return err
	}

	// ctx.Data = bytes.Clone(buf[pkg.FrameBodyHeadSize:])
	s.Data = buf[FrameBodyHeadSize:]

	return nil
}

// DecodeManyAndHandle 批量解码大包
func DecodeManyAndHandle(buf []byte, subName string, reply string, callback func(msg Packet)) error {
	offset := 0

	sm := gmetrics.GetSubMetrics(subName)

	for offset < len(buf) {
		// 读取 4 字节的长度前缀
		if len(buf)-offset < 4 {
			sm.DecodeErr.Inc()
			return gerror.New("batch decode error: missing length prefix")
		}
		subSize := int(binary.LittleEndian.Uint32(buf[offset:]))
		offset += 4

		// 截取单条消息的数据段
		if len(buf)-offset < subSize {
			sm.DecodeErr.Inc()
			return gerror.New("batch decode error: buffer too small for sub-message")
		}
		subBuf := buf[offset : offset+subSize]

		msg := Packet{}
		err := msg.Decode(subBuf)
		if err != nil {
			sm.DecodeErr.Inc()
			offset += subSize
			continue // 或者记录错误并 continue
		}

		sm.GetMsgCounter(msg.Head.MsgID).Inc()

		msg.SetReply(reply)
		callback(msg)
		offset += subSize
	}

	return nil
}
