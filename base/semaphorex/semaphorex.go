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
	// if previous adjust not effected, cancel it
	s.adjust = 0
	oldSize := s.size
	if n == s.size {
		return oldSize
	}
	if n > s.size {
		s.size = n
		s.notifyWaiters()
	} else {
		s.adjust = s.size - n
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

func (s *Weighted) goWrapper(f func()) {
	defer s.Release(1)
	f()
}
