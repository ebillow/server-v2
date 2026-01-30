package idgen

import (
	"encoding/binary"
	"encoding/hex"
	snowflakeNG "server/pkg/idgen/snowflake"
	"strconv"
	"strings"
)

// Init 初始化雪花ID算法
func Init(serverIdx int) {
	snowflakeNG.InitGenerator(serverIdx)
}

func Gen() int64 {
	return snowflakeNG.Gen()
}

func GenUint64() uint64 {
	return uint64(Gen())
}
func GenString() string {
	return strconv.FormatInt(Gen(), 10)
}

func ExtractServerId(id int64) int64 {
	return snowflakeNG.ExtractMachineId(id)
}

// ConvertToUpperHex 将 uint64 转换为大写的 hex 字符串[A-F,0-9], 长度为16
func ConvertToUpperHex(id uint64) string {
	var buf = make([]byte, 8)           // uint64 占 8 个字节
	binary.BigEndian.PutUint64(buf, id) // 将 uint64 转换为 []byte
	hexStr := hex.EncodeToString(buf)   // 每个字节转换为两个十六进制字符
	hexStr = strings.ToUpper(hexStr)    // 转为大写
	return hexStr
}
