package queue

import (
	"fmt"
	"sync"
)

type SwapQueue[T any] struct {
	mtx     sync.Mutex
	write   []T
	read    []T
	sig     chan struct{}
	maxSize int
	size    int
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

func (s *SwapQueue[T]) PushAndWake(data T) error {
	err := s.Push(data)
	s.Wake()

	return err
}

func (s *SwapQueue[T]) Push(data T) error {
	s.mtx.Lock()

	if len(s.write) >= s.maxSize {
		s.mtx.Unlock()
		return fmt.Errorf("queue is full, now=%d, max=%d", len(s.write), s.maxSize)
	}

	s.write = append(s.write, data)
	s.mtx.Unlock() // 写入完毕，立刻解锁

	return nil
}

func (s *SwapQueue[T]) Wake() {
	// 唤醒消费者：如果通道满了(已有信号)，直接丢弃，消费者醒着自然会去读
	select {
	case s.sig <- struct{}{}:
	default:
	}
}

func (s *SwapQueue[T]) Sig() <-chan struct{} {
	return s.sig
}

// Range 只能在一个消费者协程中调用
func (s *SwapQueue[T]) Range(f func(T) bool) {
	s.mtx.Lock()
	if len(s.write) == 0 {
		s.mtx.Unlock()
		return
	}

	s.write, s.read = s.read[:0], s.write
	s.mtx.Unlock()

	for i := 0; i < len(s.read); i++ {
		if !f(s.read[i]) {
			break
		}
	}

	clear(s.read)

	if cap(s.read) > s.size {
		s.read = make([]T, 0, s.size)
	}
}
