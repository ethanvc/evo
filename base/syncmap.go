package base

import "sync"

type SyncMap[K comparable, V any] struct {
	m sync.Map
}

func (sm *SyncMap[K, V]) Store(k K, v V) {
	sm.m.Store(k, v)
}

func (sm *SyncMap[K, V]) Load(k K) V {
	gv, ok := sm.m.Load(k)
	if ok {
		return gv.(V)
	} else {
		return Zero[V]()
	}
}

func (sm *SyncMap[K, V]) LoadOrStore(k K, v V) V {
	actual, _ := sm.m.LoadOrStore(k, v)
	v, ok := actual.(V)
	if ok {
		return v
	} else {
		return Zero[V]()
	}
}

func (sm *SyncMap[K, V]) ClearAll() {
	sm.m.Range(func(k, _ any) bool {
		sm.m.Delete(k)
		return true
	})
}
