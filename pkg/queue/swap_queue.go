package queue

import (
	"server/pkg/gerror"
	"sync"
	"sync/atomic"
)

var (
	ErrQueueFull   = gerror.New("queue full")
	ErrQueueClosed = gerror.New("queue closed")
)

type SwapQueue[T any] struct {
	mtx     sync.Mutex
	write   []T
	read    []T
	sig     chan struct{}
	maxSize int
	size    int
	closed  atomic.Bool
}

func NewSwapQueue[T any](size int, maxSize int) *SwapQueue[T] {
	return &SwapQueue[T]{
		write:   make([]T, 0, size),
		read:    make([]T, 0, size),
		sig:     make(chan struct{}, 1), // 必须是容量为 1 的缓冲通道
		maxSize: maxSize,
		size:    size,
	}
}

// Push 支持多生产者
func (s *SwapQueue[T]) Push(data T) error {
	err := s.pushData(data)
	s.Wake()

	return err
}

func (s *SwapQueue[T]) pushData(data T) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if s.closed.Load() {
		return ErrQueueClosed
	}

	if len(s.write) >= s.maxSize {
		return ErrQueueFull
	}

	s.write = append(s.write, data)

	return nil
}

func (s *SwapQueue[T]) Wake() {
	// 唤醒消费者：如果通道满了(已有信号)，直接丢弃，消费者醒着自然会去读
	select {
	case s.sig <- struct{}{}:
	default:
	}
}

func (s *SwapQueue[T]) Close() {
	if s.closed.CompareAndSwap(false, true) {
		s.Wake()
	}
}

func (s *SwapQueue[T]) Sig() <-chan struct{} {
	return s.sig
}

// Range 仅单消费者
func (s *SwapQueue[T]) Range(f func(T)) {
	s.mtx.Lock()
	if len(s.write) == 0 {
		s.mtx.Unlock()
		return
	}

	s.write, s.read = s.read[:0], s.write
	s.mtx.Unlock()

	for i := 0; i < len(s.read); i++ {
		f(s.read[i])
	}

	clear(s.read)

	if cap(s.read) > s.size {
		s.read = make([]T, 0, s.size)
	}
}
