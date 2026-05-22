package gerror

import (
	"errors"
	"fmt"
	"server/api/pb"
)

type CodeErr struct {
	Code pb.ErrorCode // 错误码
	Err  error        // 原始错误（含堆栈/上下文）
}

// Msg 安全获取错误码对应的字符串描述
func (e *CodeErr) Msg() string {
	if name, ok := pb.ErrorCode_name[int32(e.Code)]; ok {
		return name
	}
	return "UnknownError"
}

func (e *CodeErr) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("code=%d msg=%s", e.Code, e.Msg())
	}
	return fmt.Sprintf("code=%d msg=%s: %v", e.Code, e.Msg(), e.Err)
}

func (e *CodeErr) Unwrap() error { return e.Err }

// Format 使得 CodeErr 兼容 %+v，打印出完整的堆栈信息
func (e *CodeErr) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			// %+v 输出带堆栈的格式
			fmt.Fprintf(s, "code=%d msg=%s\n%+v", e.Code, e.Msg(), e.Err)
			return
		}
		fallthrough
	case 's':
		fmt.Fprint(s, e.Error())
	case 'q':
		fmt.Fprintf(s, "%q", e.Error())
	}
}

// CodeOf 提取 code
func CodeOf(err error) (pb.ErrorCode, bool) {
	if err == nil {
		return pb.ErrorCode_Internal, false // 或返回你定义的 Success Code
	}

	var ce *CodeErr
	if errors.As(err, &ce) {
		return ce.Code, true
	}
	return pb.ErrorCode_Internal, false
}
