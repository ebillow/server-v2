package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
)

type Config struct {
	Level      string `yaml:"Level"`      // 日志级别
	Console    bool   `yaml:"Console"`    // 是否输出到控制台 (stdout)
	MaxSize    int    `yaml:"MaxSize"`    // 单个文件大小 单位（M）
	MaxBackups int    `yaml:"MaxBackups"` // 最大备份
	MaxAge     int    `yaml:"MaxAge"`     // 最大保存天数
	Compress   bool   `yaml:"Compress"`   // 是否压缩
}

// Init init log
func Init(pathName string, conf *Config) {
	if conf == nil {
		conf = &Config{
			Level:      "debug",
			Console:    true,
			MaxSize:    100,
			MaxBackups: 30,
			MaxAge:     7,
			Compress:   false,
		}
	}
	zap.ReplaceGlobals(NewProductionLogger(pathName, conf))

}
func NewProductionLogger(pathAndName string, conf *Config) *zap.Logger {
	// 1. 配置Lumberjack切割器
	hook := lumberjack.Logger{
		Filename:   pathAndName,     // 日志文件路径
		MaxSize:    conf.MaxSize,    // 每个日志文件保存的最大尺寸 单位：M
		MaxBackups: conf.MaxBackups, // 日志文件最多保存多少个备份
		MaxAge:     conf.MaxAge,     // 文件最多保存多少天
		Compress:   conf.Compress,   // 是否压缩
	}

	// 2. 创建Zap核心
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	lv, err := zapcore.ParseLevel(conf.Level)
	if err != nil {
		panic(err)
	}

	cores := []zapcore.Core{
		zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			zapcore.AddSync(&hook),
			lv,
		),
	}

	if conf.Console {
		// 如果开了 console, 则往控制台输出人类可读的日志
		cfg := encoderConfig
		cfg.EncodeCaller = zapcore.ShortCallerEncoder
		cfg.EncodeDuration = zapcore.StringDurationEncoder
		encoder := NewConsoleEncoder(cfg, true)
		core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), lv)

		cores = append(cores, core)
	}

	// 3. 添加调用栈信息
	return zap.New(zapcore.NewTee(cores...), zap.AddCaller())
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
