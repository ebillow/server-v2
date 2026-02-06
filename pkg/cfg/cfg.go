package cfg

import (
	"fmt"
	"server/pkg/logger"
	"sync/atomic"
)

var cfg atomic.Value

func Get() *Config {
	return cfg.Load().(*Config)
}

func configPath(iid string, version string) string {
	return fmt.Sprintf("/%s/%s/config/config", iid, version)
}

type ClickHouse struct {
	Addr    []string `yaml:"Addr"`
	DBName  string   `yaml:"DBName"`
	Account string   `yaml:"Account"`
	Pwd     string   `yaml:"Pwd"`
	Count   int      `yaml:"Count"`
}

type Mongo struct {
	URL string `yaml:"URL"`
}

type Redis struct {
	Address  []string `yaml:"Address"`
	DB       int      `yaml:"DB"`
	Password string   `yaml:"Password"`
}

type MsgQueue struct {
	SAddr      string `yaml:"SAddr"`
	Timeout    int    `yaml:"Timeout"`
	MaxTryConn int    `yaml:"MaxTryConn"`
	User       string `yaml:"User"`
	Pwd        string `yaml:"Pwd"`
}

type Time struct {
	AutoSave int64 `yaml:"AutoSave"`
}

type Flag struct {
	TestPay            bool `yaml:"TestPay"`            // 不走支付流程，直接成功
	TraceMsg           bool `yaml:"TraceMsg"`           // 是否打印消息日志
	CheckClientVersion bool `yaml:"CheckClientVersion"` // 检查客户端版本
	GmEnable           bool `yaml:"GMFeature"`          // GM命令
}

type Aes struct {
	Key string `yaml:"Key"`
	IV  string `yaml:"IV"`
}

// Config 服务端总配置
type Config struct {
	MsgQueue MsgQueue `yaml:"MsgQueue"` // 全局Queue
	LogQueue MsgQueue `yaml:"LogQueue"` // 日志服Queue

	Mongo      Mongo      `yaml:"Mongo"`      // MongoDB配置
	Redis      Redis      `yaml:"Redis"`      // 服务器Redis
	ClickHouse ClickHouse `yaml:"ClickHouse"` // 日志库

	LogInfo logger.Config `yaml:"LogInfo"` // 日志配置

	Flag  Flag   `yaml:"Flag"`  // 开关
	Proxy string `yaml:"Proxy"` // 代理
	Time  Time   `yaml:"Time"`  // 时间间隔配置
	Aes   Aes    `yaml:"Aes"`   // aes加密配置
}
