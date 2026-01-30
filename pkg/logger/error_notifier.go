package logger

import (
	"fmt"
	"github.com/tidwall/gjson"
	"server/pkg/flag"
	"server/pkg/notice"
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

// ErrorNotifierV1 错误日志报警, 实现 io.Writer
type ErrorNotifierV1 struct {
	IId       string
	NotifyUrl string
}

func (h *ErrorNotifierV1) Write(p []byte) (n int, _ error) {
	defer func() { n = len(p) }()

	if h.NotifyUrl == "" {
		return
	}

	jsonRet := gjson.ParseBytes(p)

	ts := jsonRet.Get("@time").String()
	line := jsonRet.Get("@line").String()
	lv := jsonRet.Get("@level").String()
	msg := jsonRet.Get("@msg").String()
	stack := jsonRet.Get("stack").String()

	var fieldContent string
	jsonRet.ForEach(func(key, value gjson.Result) bool {
		k := key.String()
		switch k {
		case "@time", "@line", "@level", "@msg", "stack", FieldColor:
		default:
			var val string
			if value.Type == gjson.JSON {
				val = value.String()
			} else {
				val = quoteIfNeeded(value.String())
			}
			fieldContent += fmt.Sprintf("%s=%s ", quoteIfNeeded(k), val)
		}
		return true
	})

	srvNode := fmt.Sprintf("%s(%d)", flag.SrvName(flag.SrvType), flag.SvcIndex)
	noticeMsg := fmt.Sprintf(noticeMsgTpl, h.IId, srvNode, ts, line, lv, msg, fieldContent, stack)
	notice.Add(noticeMsg, h.NotifyUrl, line)

	return n, nil
}
