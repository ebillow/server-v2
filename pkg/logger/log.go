package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"strconv"
	"time"
)

// Config 日志配置
type Config struct {
	Level                int               `yaml:"Level"`                // 日志级别
	Console              bool              `yaml:"Console"`              // 是否输出到控制台 (stdout)
	ConsoleColor         bool              `yaml:"ConsoleColor"`         // 是否打印颜色
	MaxSize              int               `yaml:"MaxSize"`              // 单个文件大小 单位（M）
	MaxBackups           int               `yaml:"MaxBackups"`           // 最大备份
	MaxAge               int               `yaml:"MaxAge"`               // 最大保存天数
	Compress             bool              `yaml:"Compress"`             // 是否压缩
	NoticeUrl            string            `yaml:"NoticeUrl"`            // 通知url
	CfgNoticeUrl         string            `yaml:"CfgNoticeUrl"`         // 启动时配置报错或者热更配置报错 到打包群url
	PayWanUrl            string            `yaml:"PayWanUrl"`            // 启动时配置报错或者热更配置报错 到url
	ExtraWhiteListMsgIds map[int32][]int64 `yaml:"ExtraWhiteListMsgIds"` // 消息白名单额外规则; msgId => userIds; read-only
	IId                  string            `yaml:"-"`
}

func NewZapLog(pathAndName string, conf Config) {
	hook := lumberjack.Logger{
		Filename:   pathAndName,     // 日志文件路径
		MaxSize:    conf.MaxSize,    // 每个日志文件保存的最大尺寸 单位：M
		MaxBackups: conf.MaxBackups, // 日志文件最多保存多少个备份
		MaxAge:     conf.MaxAge,     // 文件最多保存多少天
		Compress:   conf.Compress,   // 是否压缩
	}

	encoderConfig := zapcore.EncoderConfig{
		MessageKey:     "@msg",
		LevelKey:       "@level",
		TimeKey:        "@time",
		NameKey:        "@name",
		CallerKey:      "@line",
		StacktraceKey:  "@stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder, // 小写编码器
		EncodeTime:     zapcore.ISO8601TimeEncoder,    // ISO8601 UTC 时间格式
		EncodeDuration: zapcore.MillisDurationEncoder, // 将 duration 显示为毫秒
		EncodeCaller:   zapcore.FullCallerEncoder,     // 全路径编码器
		EncodeName:     zapcore.FullNameEncoder,
	}

	// 设置日志级别
	atomicLevel := zap.NewAtomicLevel()
	atomicLevel.SetLevel(zapcore.Level(conf.Level))

	var cores = []zapcore.Core{
		zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), zapcore.AddSync(&hook), atomicLevel),
	}
	if conf.Console {
		// 如果开了 console, 则往控制台输出人类可读的日志
		cfg := encoderConfig
		cfg.EncodeCaller = zapcore.ShortCallerEncoder
		cfg.EncodeDuration = zapcore.StringDurationEncoder
		encoder := NewConsoleEncoder(cfg, conf.ConsoleColor)
		core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), atomicLevel)

		// 添加日志采样:
		// 如果一次 tick 内出现相同等级内容的日志，则打印前 N 条.
		// 如果当前 tick 内后续还有日志, 则每 M 条打印一次.
		// core = zapcore.NewSamplerWithOptions(core, time.Second, 10, 5)
		cores = append(cores, core)
	}
	if len(conf.NoticeUrl) > 0 {
		errorOnlyCore := zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			// zapcore.AddSync(&errorHook{}), // 你的自定义hook
			zapcore.AddSync(&ErrorNotifierV1{IId: conf.IId, NotifyUrl: conf.NoticeUrl}),
			zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
				return lvl >= zapcore.ErrorLevel // 只处理Error及以上级别
			}),
		)

		// 添加日志采样:
		// 如果一次 tick 内出现相同等级内容的日志，则打印前 N 条.
		// 如果当前 tick 内后续还有日志, 则每 M 条打印一次.
		errorOnlyCore = zapcore.NewSamplerWithOptions(errorOnlyCore, time.Second, 10, 5)
		cores = append(cores, errorOnlyCore)
	}

	core := zapcore.NewTee(cores...)

	// 开启开发模式，堆栈跟踪
	log := zap.New(core, zap.AddCaller())
	zap.ReplaceGlobals(log)
}

const (
	FieldColor = "_color"
)

func quoteIfNeeded(s string) string {
	if needsQuote(s) {
		return strconv.Quote(s)
	}
	return s
}

func needsQuote(s string) bool {
	for i := range s {
		if s[i] < 0x20 || s[i] > 0x7e || s[i] == ' ' || s[i] == '\\' || s[i] == '"' {
			return true
		}
	}
	return false
}

// 兼容
// Debug logs a message at level Debug on the standard logger.
func Debug(args ...interface{}) {
	zap.S().Debug(args...)
}

// Info logs a message at level Info on the standard logger.
func Info(args ...interface{}) {
	zap.S().Info(args...)
}

// Warn logs a message at level Warn on the standard logger.
func Warn(args ...interface{}) {
	zap.S().Warn(args...)
}

// Error logs a message at level Error on the standard logger.
func Error(args ...interface{}) {
	zap.S().Error(args...)
}

// Panic logs a message at level Panic on the standard logger.
func Panic(args ...interface{}) {
	zap.S().Panic(args...)
}

// Tracef logs a message at level Trace on the standard logger.
func Tracef(format string, args ...interface{}) {
	zap.S().Debugf(format, args...)
}

// Debugf logs a message at level Debug on the standard logger.
func Debugf(format string, args ...interface{}) {
	zap.S().Debugf(format, args...)
}

// Infof logs a message at level Info on the standard logger.
func Infof(format string, args ...interface{}) {
	zap.S().Infof(format, args...)
}

// Warnf logs a message at level Warn on the standard logger.
func Warnf(format string, args ...interface{}) {
	zap.S().Warnf(format, args...)
}

// Errorf logs a message at level Error on the standard logger.
func Errorf(format string, args ...interface{}) {
	zap.S().Errorf(format, args...)
}

// Panicf logs a message at level Panic on the standard logger.
func Panicf(format string, args ...interface{}) {
	zap.S().Panicf(format, args...)
}
