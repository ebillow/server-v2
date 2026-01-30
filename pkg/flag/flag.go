package flag

import (
	"log"
	"os"
	"server/pkg/pb"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

// 服务启动时的flags
var (
	ConfigFile string // 配置文件路径
	SvcIndex   int    // 服务索引
	RpcPort    int    // rpc 端口
	HttpPort   int    // http 端口
	TcpPort    int
	SrvType    pb.Server
)

// Init 解析flags
func Init(serverType pb.Server, fs *pflag.FlagSet, manualParse bool) {
	fs.StringVar(&ConfigFile, "config", "", "配置文件路径")
	fs.StringVar(&ConfigFile, "cfg", "", "alias for --config")
	fs.IntVar(&SvcIndex, "index", 0, "服务索引")
	fs.IntVar(&RpcPort, "rpc-port", 0, "rpc 监听端口")
	fs.IntVar(&HttpPort, "http-port", 0, "http 监听端口")
	fs.IntVar(&TcpPort, "tcp-port", 3001, "tcp 监听端口")
	fs.SortFlags = false

	if manualParse {
		pflag.Parse()
	}

	if hostName := os.Getenv("HOSTNAME"); hostName != "" {
		if hostIdx := SplitHostName(hostName); hostIdx >= 0 {
			SvcIndex = hostIdx
		}
	}
	var err error
	if str := os.Getenv("RPC_PORT"); str != "" {
		RpcPort, err = strconv.Atoi(str)
		if err != nil {
			log.Panicf("rpc port error:%v", err.Error())
		}
	}
	if str := os.Getenv("HTTP_PORT"); str != "" {
		HttpPort, err = strconv.Atoi(str)
		if err != nil {
			log.Panicf("http port error:%v", err.Error())
		}
	}
	SrvType = serverType
}

// Debug 打印输入的flags, 需要在初始化日志后调用.
func Debug(fs *pflag.FlagSet) {
	fs.VisitAll(func(f *pflag.Flag) {
		zap.S().Debugf("FLAG: --%s=%s", f.Name, f.Value)
	})
}

func SrvName(serType pb.Server) string {
	return pb.Server_name[int32(serType)]
}

var isReady bool // 服务是否就绪

func IsReady() bool { return isReady }
func SetReady()     { isReady = true }

func splitHostNameAndID(hostName string) (string, int) {
	if strings.Contains(hostName, "-") {
		pos := strings.LastIndex(hostName, "-")
		if pos+1 >= len(hostName) {
			log.Panicf("split host name err:%s", hostName)
			return "", 99
		}
		bh := []byte(hostName)
		idStr := string(bh[pos+1:])
		id, err := strconv.Atoi(idStr)
		if err != nil {
			log.Panicf("split host name err:%s", hostName)
			return "", 99
		} else {
			return string(bh[:pos]), id
		}
	}
	// logger.Panicf("split host name err:%s", hostName)
	return "", -1
}

func SplitHostName(hostName string) int {
	_, id := splitHostNameAndID(hostName)
	return id
}
