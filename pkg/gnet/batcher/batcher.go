package batcher

type batcherState int32

const (
	BStateRunning batcherState = iota
	BStateClosing
	BStateStopped
)

// FlushFunc 是由具体的 NATS / JetStream 实现的发送回调
type FlushFunc func(data []byte, count int)
