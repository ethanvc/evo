// base on json 1.20.6
package json

import (
	"sync"
	"sync/atomic"
)

type ExtConfiger struct {
	fieldCache       sync.Map
	encoderCache     sync.Map
	CustomGetEncoder func(*ExtConfiger, *Field) EncoderFunc
}

func NewExtConfiger() *ExtConfiger {
	configer := &ExtConfiger{}
	return configer
}

func (configer *ExtConfiger) GetEncoder(f *Field) EncoderFunc {
	if configer.CustomGetEncoder != nil {
		return configer.CustomGetEncoder(configer, f)
	}
	return nil
}

var DefaultConfier atomic.Pointer[ExtConfiger]

func init() {
	DefaultConfier.Store(NewExtConfiger())
}

func GetConfiger(configer ...*ExtConfiger) *ExtConfiger {
	for _, c := range configer {
		if c != nil {
			return c
		}
	}
	return DefaultConfier.Load()
}
