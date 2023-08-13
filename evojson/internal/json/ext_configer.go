// base on json 1.20.6
package json

import (
	"sync"
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

func getConfiger(confgier ...*ExtConfiger) *ExtConfiger {
	for _, c := range confgier {
		if c != nil {
			return c
		}
	}
	return jsonDefaultConfiger
}

var jsonDefaultConfiger = NewExtConfiger()

type Wrapper struct {
	configer *ExtConfiger
	v        any
}

func NewWrapper(configer *ExtConfiger, v any) Wrapper {
	if configer == nil {
		configer = jsonDefaultConfiger
	}
	return Wrapper{
		configer: configer,
		v:        v,
	}
}

func (w Wrapper) MarshalJSON() ([]byte, error) {
	return Marshal(w)
}

func decodeWrapper(v any) (any, *ExtConfiger) {
	wrapper, ok := v.(Wrapper)
	if ok {
		return wrapper.v, wrapper.configer
	}
	return v, jsonDefaultConfiger
}
