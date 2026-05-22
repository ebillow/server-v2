package pb

import (
	msgid2 "server/api/pb/msgid"
	"sync"

	"google.golang.org/protobuf/proto"
)

var (
	C2SUsePool = make(map[uint32]bool)
	S2CUsePool = make(map[uint32]bool)
)

func init() {
	C2SUsePool[uint32(msgid2.MsgIDC2S_C2SEcho)] = true
	S2CUsePool[uint32(msgid2.MsgIDS2C_S2CEcho)] = true
}

var C2SEchoPool = sync.Pool{
	New: func() any {
		return &C2SEcho{}
	},
}

func GetC2SEcho() *C2SEcho {
	return C2SEchoPool.Get().(*C2SEcho)
}

func PutC2SEcho(msg *C2SEcho) {
	if msg == nil {
		return
	}
	proto.Reset(msg)
	C2SEchoPool.Put(msg)
}

var S2CEchoPool = sync.Pool{
	New: func() any {
		return &S2CEcho{}
	},
}

func GetS2CEcho() *S2CEcho {
	return S2CEchoPool.Get().(*S2CEcho)
}

func PutS2CEcho(msg *S2CEcho) {
	if msg == nil {
		return
	}
	proto.Reset(msg)
	S2CEchoPool.Put(msg)
}
