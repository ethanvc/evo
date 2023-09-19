package semaphorex

import (
	"context"
	"golang.org/x/sync/semaphore"
)

type _ = semaphore.Weighted // reference to copy code here

func (s *Weighted) SetSize(n int64) int64 {
	if n < 0 {
		n = 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	oldSize := s.size
	s.size = n
	if s.size > oldSize {
		s.notifyWaiters()
	}
	return oldSize
}

func (s *Weighted) GetSize() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.size
}

func (s *Weighted) GetCurrent() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cur
}

func (s *Weighted) Go(c context.Context, f func()) (err error) {
	err = s.Acquire(c, 1)
	if err != nil {
		return err
	}
	go s.goWrapper(f)
	return
}

func (s *Weighted) Wait() {
	s.SetSize(1)
	s.Acquire(context.Background(), 1)
	s.SetSize(0)
	s.Release(1)
}

func (s *Weighted) GetWaitRoutines() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.waiters.Len()
}

func (s *Weighted) goWrapper(f func()) {
	defer s.Release(1)
	f()
}
