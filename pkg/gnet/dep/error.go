package dep

import "server/pkg/gerror"

var (
	ErrClosed = gerror.New("msgq closed")
	ErrArg    = gerror.New("msgq invalid argument")
)
