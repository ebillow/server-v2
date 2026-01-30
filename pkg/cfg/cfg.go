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

func configPath(version string, iid string) string {
	return fmt.Sprintf("/%s/%s/config/config", iid, version)
}

type ClickHouse struct {
	Addr    []string `yaml:"Addr"`
	DBName  string   `yaml:"DBName"`
	Account string   `yaml:"Account"`
	Pwd     string   `yaml:"Pwd"`
	Count   int      `yaml:"Count"`
}

type Redis struct {
	Address          string `yaml:"Address"`
	DB               int    `yaml:"DB"`
	Password         string `yaml:"Password"`
	IsCluster        bool   `yaml:"IsCluster"`
	SlowLogThreshold int    `yaml:"SlowLogThreshold"` // 打印慢日志阈值 毫秒; <= 0 表示不打印
}

type MsgQueue struct {
	SAddr      string `yaml:"SAddr"`
	Timeout    int    `yaml:"Timeout"`
	MaxTryConn int    `yaml:"MaxTryConn"`
	User       string `yaml:"User"`
	Pwd        string `yaml:"Pwd"`
}

// Etcd 服务发现
type Etcd struct {
	AddrList    []string `yaml:"addrList"`
	Username    string   `yaml:"username"`
	Password    string   `yaml:"password"`
	DialTimeout int      `yaml:"dialTimeout"`
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
	IId        string        `yaml:"iid"`      // 整合商ID
	LogInfo    logger.Config `yaml:"LogInfo"`  // 日志配置
	MsgQueue   MsgQueue      `yaml:"MsgQueue"` // 全局Queue
	LogQueue   MsgQueue      `yaml:"LogQueue"` // 日志服Queue
	Flag       Flag
	Redis      Redis      `yaml:"Redis"`      // 服务器Redis
	Proxy      string     `yaml:"Proxy"`      // 代理
	Etcd       Etcd       `yaml:"etcd"`       // 服务发现
	ClickHouse ClickHouse `yaml:"ClickHouse"` // 日志库
	Time       Time       `yaml:"Time"`       // 时间间隔配置
	Aes        Aes        `yaml:"Aes"`        // aes加密配置
}
