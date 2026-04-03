package logger

import (
	"fmt"
	"go.uber.org/zap/zapcore"
	"server/pkg/flag" // 请根据你的实际项目路径调整
	"server/pkg/notice"
	"strings"
)

// noticeMsgTpl 通知模板
const noticeMsgTpl = `<font color="warning">实时报警，请相关同事注意。</font>
> 环境: <font color="green">%s</font>
> 服务: <font color="green">%s</font>
> 时间: <font color="comment">%s</font>
> 行号: <font color="comment">%s</font>
> 等级: <font color="red">%s</font>
> 消息: <font color="red">%s</font>
> kv: <font color="comment">%s</font>
> 堆栈: <font color="red">%s</font>
`

// ErrorNotifierCore 错误日志报警核心，实现 zapcore.Core
// 直接处理结构化日志，达到零 JSON 反序列化开销
type ErrorNotifierCore struct {
	zapcore.LevelEnabler
	IId       string
	NotifyUrl string
	fields    []zapcore.Field // 保存 With 方法附加的上下文信息
}

// With 实现 zapcore.Core 的 With 接口，用于附加上下文
func (c *ErrorNotifierCore) With(fields []zapcore.Field) zapcore.Core {
	return &ErrorNotifierCore{
		LevelEnabler: c.LevelEnabler,
		IId:          c.IId,
		NotifyUrl:    c.NotifyUrl,
		fields:       append(c.fields, fields...),
	}
}

// Check 检查是否应该记录此级别的日志
func (c *ErrorNotifierCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

// Write 执行实际的报警逻辑
func (c *ErrorNotifierCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	if c.NotifyUrl == "" {
		return nil
	}

	// 合并上下文中的 fields 和当前日志的 fields
	allFields := make([]zapcore.Field, 0, len(c.fields)+len(fields))
	allFields = append(allFields, c.fields...)
	allFields = append(allFields, fields...)

	// 将 KV 结构化为可读字符串（使用 MapObjectEncoder 性能最好，这里为了适配你的原版格式做精简拼接）
	kvBuilder := strings.Builder{}
	for _, f := range allFields {
		// 简单格式化，zap 原生字段直接取 Interface 或 Integer 等
		kvBuilder.WriteString(fmt.Sprintf("%s=%v ", f.Key, getFieldValue(f)))
	}

	srvNode := fmt.Sprintf("%s(%d)", flag.SrvName(flag.SrvType), flag.SvcIndex)
	timeStr := ent.Time.Format("2006-01-02 15:04:05.000")
	callerStr := ent.Caller.TrimmedPath()
	levelStr := ent.Level.CapitalString()
	stackStr := ent.Stack

	noticeMsg := fmt.Sprintf(noticeMsgTpl, c.IId, srvNode, timeStr, callerStr, levelStr, ent.Message, kvBuilder.String(), stackStr)

	// 【核心优化】：必须异步发送！绝不能让 notice.Add 的网络请求阻塞游戏主逻辑 Goroutine
	go func(msg, url, line string) {
		notice.Add(msg, url, line)
	}(noticeMsg, c.NotifyUrl, callerStr)

	return nil
}

// Sync 同步方法 (这里无需实现)
func (c *ErrorNotifierCore) Sync() error {
	return nil
}

// getFieldValue 简单的 field 提取辅助方法
func getFieldValue(f zapcore.Field) any {
	switch f.Type {
	case zapcore.Int64Type, zapcore.Int32Type, zapcore.Int16Type, zapcore.Int8Type, zapcore.Uint64Type, zapcore.Uint32Type:
		return f.Integer
	case zapcore.StringType:
		return f.String
	default:
		return f.Interface
	}
}
