package clinet

import (
	"crypto/cipher"
	"encoding/binary"
	"errors"
	"server/api/pb/msgid"
	"server/pkg/crypt/gaes"
)

// ---------------------------------------------------
func newReader(data []byte, deCyp cipher.BlockMode) (*pkgReader, error) {
	dataLen := len(data)
	if dataLen < 4 {
		return nil, errors.New("packet head < 2")
	}

	if deCyp != nil {
		var err error
		data, err = gaes.DeCrypt(data, deCyp)
		if err != nil {
			return nil, err
		}
	}

	return &pkgReader{data: data}, nil
}

//msgid:2/data
type pkgReader struct {
	data []byte
}

func (p *pkgReader) GetMsgID() uint32 {
	return binary.BigEndian.Uint32(p.data[0:4])
}

func (p *pkgReader) GetData() []byte {
	return p.data[4:]
}

// ---------------------------------------------------
type pkgWriter struct {
	msgId uint32
	data  []byte
}

func newPkgWriter(msgId uint32, data []byte) *pkgWriter {
	return &pkgWriter{
		msgId: msgId,
		data:  data,
	}
}

func (p *pkgWriter) Write(retCache []byte, enCyp cipher.BlockMode) int {
	binary.BigEndian.PutUint32(retCache[0:4], p.msgId)
	copy(retCache[4:], p.data)
	endPos := len(p.data) + 4

	if enCyp != nil && p.msgId != uint32(msgid.MsgIDC2S_C2SInit) {
		gaes.EnCrypt(retCache[0:endPos], enCyp)
	}

	return endPos
}
