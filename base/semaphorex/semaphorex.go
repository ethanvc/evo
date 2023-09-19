package semaphorex

import "golang.org/x/sync/semaphore"

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
		s.cur -= n - s.size
		if s.cur < 0 {
			s.mu.Unlock()
			panic("semaphore: released more than held")
		}
		s.notifyWaiters()
	} else {
		s.adjust += s.size - n
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
