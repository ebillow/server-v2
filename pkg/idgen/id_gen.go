package idgen

import (
	"github.com/sony/sonyflake/v2"
)

var (
	sf *sonyflake.Sonyflake
)

// Init 初始化雪花ID算法
func Init(serverIdx int) (err error) {
	sf, err = sonyflake.New(sonyflake.Settings{
		MachineID: func() (int, error) {
			return serverIdx, nil
		},
	})
	return nil
}

func Gen() (int64, error) {
	return sf.NextID()
}

func MachineID(id int64) int64 {
	parts := sf.Decompose(id)
	return parts["machine"]
}
