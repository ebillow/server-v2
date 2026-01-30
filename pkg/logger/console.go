package logger

import (
	"fmt"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
	"os"
	"strings"
)

// implementation check
var _ zapcore.Encoder = &ConsoleEncoder{}

// ConsoleEncoder 控制台 encoder, 用来输出人类可读的日志. 底层继承 zapcore.jsonEncoder
type ConsoleEncoder struct {
	zapcore.Encoder                       // based on zapcore.jsonEncoder
	config          zapcore.EncoderConfig // 配置
	enableColor     bool                  // 是否打印颜色
}

// NewConsoleEncoder 创建控制台 encoder
func NewConsoleEncoder(cfg zapcore.EncoderConfig, colorful bool) *ConsoleEncoder {
	jsonEncoder := zapcore.NewJSONEncoder(cfg)
	return &ConsoleEncoder{
		Encoder:     jsonEncoder,
		config:      cfg,
		enableColor: colorful,
	}
}

// Clone 实现 zapcore.Encoder 的 Clone
func (ce *ConsoleEncoder) Clone() zapcore.Encoder {
	return &ConsoleEncoder{
		Encoder:     ce.Encoder.Clone(),
		config:      ce.config,
		enableColor: ce.enableColor,
	}
}

// EncodeEntry 实现 zapcore.Encoder 的 EncodeEntry
func (ce *ConsoleEncoder) EncodeEntry(ent zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	buf, err := ce.Encoder.EncodeEntry(ent, fields)
	if err != nil {
		return nil, err
	}

	jsonLogContent := buf.String()
	jsonRet := gjson.Parse(jsonLogContent)

	buf.Reset()

	// write timestamp
	ts := jsonRet.Get(ce.config.TimeKey).String()
	_, _ = buf.WriteString(ce.colorize(BrightBlack, ts) + " ")

	// write level
	level := jsonRet.Get(ce.config.LevelKey).String()
	_, _ = buf.WriteString(ce.formatLevel(level) + " ")

	// write caller
	if caller := jsonRet.Get(ce.config.CallerKey).String(); caller != "" {
		_, _ = buf.WriteString(ce.formatCaller(caller) + " ")
	}

	// write message with color
	if msg := jsonRet.Get(ce.config.MessageKey).String(); msg != "" {
		var color = ColorNone
		if v := jsonRet.Get(FieldColor); v.Exists() && v.Type == gjson.Number {
			color = Color(v.Uint())
		}
		msg = strings.TrimSpace(msg)
		_, _ = buf.WriteString(ce.colorize(color, msg) + " ")
		// _, _ = buf.WriteString(msg + " ")
	}

	// write error
	if v := jsonRet.Get("error"); v.Exists() {
		_, _ = buf.WriteString(ce.colorize(Cyan, "error=") + ce.colorize(Red, quoteIfNeeded(v.String())) + " ")

		if vv := jsonRet.Get("errorVerbose"); vv.Exists() {
			_, _ = buf.WriteString(ce.colorize(Cyan, "errorVerbose=") + quoteIfNeeded(vv.String()) + " ")
		}
	}

	// write other fields, the order is preserved.
	jsonRet.ForEach(func(key, value gjson.Result) bool {
		k := key.String()
		switch k {
		case ce.config.LevelKey, ce.config.TimeKey, ce.config.MessageKey, ce.config.CallerKey, FieldColor, "error", "errorVerbose":
			// 忽略已经写过的字段
		default:
			var val string
			if value.Type == gjson.JSON {
				val = value.String()
			} else {
				val = quoteIfNeeded(value.String())
			}

			_, _ = buf.WriteString(ce.colorize(Cyan, quoteIfNeeded(k)+"=") + val + " ")
		}
		return true
	})

	// EOL
	buf.TrimNewline()
	_ = buf.WriteByte('\n')

	return buf, nil
}

// func (ce *ConsoleEncoder) quoteIfNeeded(s string) string {
// 	if ce.needsQuote(s) {
// 		return strconv.Quote(s)
// 	}
// 	return s
// }

// func (ce *ConsoleEncoder) needsQuote(s string) bool {
// 	for i := range s {
// 		if s[i] < 0x20 || s[i] > 0x7e || s[i] == ' ' || s[i] == '\\' || s[i] == '"' {
// 			return true
// 		}
// 	}
// 	return false
// }

func (ce *ConsoleEncoder) formatCaller(c string) string {
	if len(c) > 0 {
		c = ce.colorize(Blue, c) + " " + ce.colorize(Red, ">")
	}
	return c
}

func (ce *ConsoleEncoder) formatLevel(level string) string {
	if level == "" {
		level = "??????"
	}
	l := fmt.Sprintf("%-6s", strings.ToUpper(level)) // 对齐

	if !ce.enableColor {
		return l
	}

	// 上色
	var color = Bold
	switch strings.ToLower(level) {
	case zapcore.DebugLevel.String():
		color = Magenta
	case zapcore.InfoLevel.String():
		color = Blue
	case zapcore.WarnLevel.String():
		color = Yellow
	case zapcore.ErrorLevel.String(), zapcore.PanicLevel.String(), zapcore.DPanicLevel.String(), zapcore.FatalLevel.String():
		color = Red
	}
	return color.Wrap(l)
}

func (ce *ConsoleEncoder) colorize(color Color, content string) string {
	if !ce.enableColor {
		return content
	}
	return color.Wrap(content)
}

// Color 终端颜色代码
// @see https://en.wikipedia.org/wiki/ANSI_escape_code for colors code
type Color uint8

const (
	ColorNone Color = iota
	Bold

	Black Color = iota + 28
	Red
	Green
	Yellow
	Blue
	Magenta // purple
	Cyan
	White

	BrightBlack Color = iota + 80 // Gray
	BrightRed
	BrightGreen
	BrightYellow
	BrightBlue
	BrightMagenta
	BrightCyan
	BrightWhite
)

func (c Color) Wrap(content string) string {
	if c == ColorNone || os.Getenv("NO_COLOR") != "" {
		return content
	}
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", uint8(c), content)
}

// Field 将颜色构建成 zap.Field
func (c Color) Field() zap.Field {
	return zap.Any(FieldColor, c)
}
