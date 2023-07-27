package base

import "sync"

type SyncMap[K, V any] struct {
	m sync.Map
}

func (sm *SyncMap[K, V]) Store(k K, v V) {
	sm.m.Store(k, v)
}

func (sm *SyncMap[K, V]) Load(k K) V {
	var d V
	gv, ok := sm.m.Load(k)
	if ok {
		return gv.(V)
	} else {
		return d
	}
}

func (sm *SyncMap[K, V]) ClearAll() {
	sm.m.Range(func(k, _ any) bool {
		sm.m.Delete(k)
		return true
	})
}
