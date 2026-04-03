package logger

import (
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Config 日志配置
type Config struct {
	Level                int               `yaml:"Level"`
	Console              bool              `yaml:"Console"`
	ConsoleColor         bool              `yaml:"ConsoleColor"`
	MaxSize              int               `yaml:"MaxSize"`
	MaxBackups           int               `yaml:"MaxBackups"`
	MaxAge               int               `yaml:"MaxAge"`
	Compress             bool              `yaml:"Compress"`
	NoticeUrl            string            `yaml:"NoticeUrl"`
	CfgNoticeUrl         string            `yaml:"CfgNoticeUrl"`
	PayWanUrl            string            `yaml:"PayWanUrl"`
	ExtraWhiteListMsgIds map[int32][]int64 `yaml:"ExtraWhiteListMsgIds"`
	IId                  string            `yaml:"-"`
}

var globalLogger *zap.Logger

func NewZapLog(pathAndName string, conf Config) {
	hook := lumberjack.Logger{
		Filename:   pathAndName,
		MaxSize:    conf.MaxSize,
		MaxBackups: conf.MaxBackups,
		MaxAge:     conf.MaxAge,
		Compress:   conf.Compress,
	}

	encoderConfig := zapcore.EncoderConfig{
		MessageKey:     "@msg",
		LevelKey:       "@level",
		TimeKey:        "@time",
		NameKey:        "@name",
		CallerKey:      "@line",
		StacktraceKey:  "@stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.FullCallerEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}

	atomicLevel := zap.NewAtomicLevel()
	atomicLevel.SetLevel(zapcore.Level(conf.Level))

	var cores []zapcore.Core

	// 文件输出 Core (纯 JSON)
	fileCore := zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), zapcore.AddSync(&hook), atomicLevel)
	cores = append(cores, fileCore)

	// 控制台输出 Core (如果开启)
	if conf.Console {
		consoleCfg := encoderConfig
		consoleCfg.EncodeCaller = zapcore.ShortCallerEncoder
		consoleCfg.EncodeDuration = zapcore.StringDurationEncoder

		// 使用原生高亮替代自定义 ConsoleEncoder，避免 JSON 反序列化损耗
		if conf.ConsoleColor {
			consoleCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		} else {
			consoleCfg.EncodeLevel = zapcore.CapitalLevelEncoder
		}

		consoleCore := zapcore.NewCore(zapcore.NewConsoleEncoder(consoleCfg), zapcore.AddSync(os.Stdout), atomicLevel)
		cores = append(cores, consoleCore)
	}

	// 错误报警 Core (如果配置了 URL)
	if len(conf.NoticeUrl) > 0 {
		noticeEnabler := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl >= zapcore.ErrorLevel // 只处理 Error 及以上
		})

		noticeCore := &ErrorNotifierCore{
			LevelEnabler: noticeEnabler,
			IId:          conf.IId,
			NotifyUrl:    conf.NoticeUrl,
		}

		// 采样机制，防止连环报错时产生报警风暴打挂外部接口
		sampledNoticeCore := zapcore.NewSamplerWithOptions(noticeCore, time.Second, 10, 5)
		cores = append(cores, sampledNoticeCore)
	}

	// 合并所有 Core
	core := zapcore.NewTee(cores...)

	// 限制 AddStacktrace 只在 Error 级别触发
	globalLogger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))
	zap.ReplaceGlobals(globalLogger)
}
