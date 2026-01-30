package session

import (
	"crypto/cipher"
	"encoding/binary"
	"errors"
	"server/pkg/crypt/gaes"
	"server/pkg/pb/msgid"
)

func Decode(src []byte, deCyp cipher.BlockMode) (msgID uint32, seq uint32, data []byte, err error) {
	dataLen := len(src)
	const minDataLen = 8
	if dataLen < minDataLen {
		return 0, 0, nil, errors.New("packet head < 6")
	}

	if deCyp != nil {
		src = gaes.DeCrypt(src, deCyp)
	}
	msgID = binary.BigEndian.Uint32(src[0:4])
	seq = binary.BigEndian.Uint32(src[4:8])
	data = src[8:]
	return
}

func Encode(msgID uint32, src []byte, enCyp cipher.BlockMode, cache []byte) []byte {
	binary.BigEndian.PutUint32(cache[0:4], msgID)
	copy(cache[4:], src)

	endPos := len(src) + 4

	if enCyp != nil && msgID != uint32(msgid.MsgIDS2C_S2CInit) {
		return gaes.EnCrypt(cache[0:endPos], enCyp)
	}
	return cache[:endPos]
}
