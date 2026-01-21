package version

import (
	"fmt"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"runtime"
)

var (
	BuildTime string // 编译时间
	GitCommit string // git 提交
	GitTag    string // git tag
)

func LogVersion() {
	zap.S().Infof("BuildTime: %s", BuildTime)
	zap.S().Infof("GitTag: %s", GitTag)
	zap.S().Infof("GitCommit: %s", GitCommit)
	zap.S().Infof("GoVersion: %s", runtime.Version())
}

// String 以字符串方式返回版本信息.
func String() string {
	str := "BuildTime: %s\n" +
		"GitTag: %s\n" +
		"GitCommit: %s\n" +
		"GoVersion: %s\n"
	return fmt.Sprintf(str, BuildTime, GitTag, GitCommit, runtime.Version())
}

// CobraCmd 返回 cobra 子命令
func CobraCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "show server version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(String())
		},
	}
}
