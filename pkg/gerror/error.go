package gerror

import (
	"server/api/pb"

	crerr "github.com/cockroachdb/errors"
)

/*
产生点：NewCode / From
边界层：Wrap 一次（比如 DAO → service、service → handler）
*/

// NewCode 带业务 Code 和堆栈的错误
func NewCode(code pb.ErrorCode) error {
	return &CodeErr{
		Code: code,
		Err:  crerr.New(pb.ErrorCode_name[int32(code)]),
	}
}

func New(msg string) error {
	return crerr.New(msg)
}

// From 仅赋予 ErrorCode，保留原始错误链和堆栈
func From(code pb.ErrorCode, err error) error {
	if err == nil {
		return nil
	}
	return &CodeErr{
		Code: code,
		Err:  crerr.WithStack(err),
	}
}

func WithStack(err error) error {
	if err == nil {
		return nil
	}
	return crerr.WithStack(err) // 堆栈在 %+v 中可见
}

// -----------边界层添加语义--------------

// Wrap 跨边界/关键节点加自定义语义
func Wrap(err error, msg string) error {
	if err == nil {
		return nil
	}
	return crerr.Wrap(err, msg)
}

func Wrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return crerr.Wrapf(err, format, args...)
}

// WrapCode 跨边界附加 ErrorCode 和自定义上下文语义
func WrapCode(code pb.ErrorCode, err error, msg string) error {
	if err == nil {
		return nil
	}
	return &CodeErr{
		Code: code,
		// 使用传入的 msg 增强上下文，而不是重复的 code name
		Err: crerr.Wrap(err, msg),
	}
}
