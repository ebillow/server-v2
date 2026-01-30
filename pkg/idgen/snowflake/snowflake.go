package snowflake

import (
	"fmt"
	"strconv"
	"sync"
	"time"
)

// 标准雪花算法生成的 ID 构成:
//
// [ 符号位(1bit) | 毫秒时间戳(41bit) | 机器码(10bit) | 自增序列(12bit) ]
//
// 41bit 的毫秒时间戳可以覆盖 2199023255551/(365*86400000) = 69.73057 年
//
// @see: https://en.wikipedia.org/wiki/Snowflake_ID
//
// **项目中可对构成进行调整**

const (
	timeUnit = 1e6 // 1ms; 基于 nanoseconds

	timestampBit = 41 // 时间戳位数
	machineBit   = 10 // 机器码位数
	seqBit       = 12 // 自增序列位数

	machineShift   = seqBit              // 机器码偏移
	timestampShift = seqBit + machineBit // 时间戳偏移

	maxYears     = (-1 ^ (-1 << timestampBit)) / (365 * 24 * time.Hour / timeUnit) // 时间位可用多少年
	maxMachineId = -1 ^ (-1 << machineBit)                                         // 支持的最大机器码
	seqMask      = -1 ^ (-1 << seqBit)                                             // 自增序列掩码
)

var beginEpoch = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixNano() / timeUnit // 开始时间戳 2024-01-01T00:00:00Z

var generator = &Generator{}

type Generator struct {
	mu        sync.Mutex
	machineID int64 // 机器码
	seq       int64 // 自增序列
	lastTs    int64 // 上一次的毫秒时间戳
}

func (g *Generator) Generate() int64 {
	g.mu.Lock()
	defer g.mu.Unlock()

	nowTs := time.Now().UnixNano() / timeUnit

	if g.lastTs == nowTs {
		g.seq = (g.seq + 1) & seqMask
		if g.seq == 0 {
			// 如果自增序列溢出就等下一帧
			for nowTs <= g.lastTs {
				time.Sleep(time.Duration((g.lastTs+1-nowTs)*timeUnit) - time.Duration(time.Now().UTC().UnixNano()%timeUnit))
				nowTs = time.Now().UnixNano() / timeUnit
			}
		}
	} else {
		g.seq = 0
	}

	g.lastTs = nowTs

	return (nowTs-beginEpoch)<<timestampShift | g.machineID<<machineShift | g.seq
}

// InitGenerator 初始化id生成器
func InitGenerator(machineID int) {
	if machineID < 0 || machineID > maxMachineId {
		panic(fmt.Errorf("machineID out of range [0~%d]", maxMachineId))
	}
	generator.seq = 0
	generator.lastTs = 0
	generator.machineID = int64(machineID)
}

// Gen 生成唯一id
func Gen() int64 {
	return generator.Generate()
}

// Extract 提取ID, 剥离出元数据
func Extract(id int64) (genTime time.Time, machineId, seq int64) {
	bin := fmt.Sprintf("%064b", id)
	tsOffset := timestampBit + 1
	tsBin, machineIdBin, seqBin := bin[:tsOffset], bin[tsOffset:tsOffset+machineBit], bin[tsOffset+machineBit:]

	ts, _ := strconv.ParseInt(tsBin, 2, 64)
	genTime = time.Unix(0, (beginEpoch+ts)*timeUnit)
	machineId, _ = strconv.ParseInt(machineIdBin, 2, 64)
	seq, _ = strconv.ParseInt(seqBin, 2, 64)

	return genTime, machineId, seq
}

func ExtractMachineId(id int64) int64 {
	bin := fmt.Sprintf("%064b", id)
	tsOffset := timestampBit + 1
	machineIdBin := bin[tsOffset : tsOffset+machineBit]
	machineId, _ := strconv.ParseInt(machineIdBin, 2, 64)
	return machineId
}
