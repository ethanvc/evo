package xsync

import "sync"

// WaitGroup same as conc.WaitGroup, but remove catch error function
type WaitGroup struct {
	sync.WaitGroup
}

func (wg *WaitGroup) Go(f func()) {
	wg.Add(1)
	go wg.goWrapper(f)
}

func (wg *WaitGroup) goWrapper(f func()) {
	defer wg.Done()
	f()
}
