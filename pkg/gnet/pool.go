package gnet

import "sync"

const (
	defaultCap = 1024
	maxKeepCap = 64 * 1024
)

var pool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, defaultCap)
		return &b
	},
}

func Get() *[]byte {
	bp := pool.Get().(*[]byte)
	*bp = (*bp)[:0]
	return bp
}

func Put(bp *[]byte) {
	if bp == nil {
		return
	}
	if cap(*bp) > maxKeepCap {
		return
	}
	*bp = (*bp)[:0]
	pool.Put(bp)
}
