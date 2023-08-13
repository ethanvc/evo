package evojson

import "github.com/ethanvc/evo/evojson/internal/json"

type ExtConfiger = json.ExtConfiger
type Field = json.Field
type EncoderFunc = json.EncoderFunc
type EncodeState = json.EncodeState
type EncOpts = json.EncOpts
type Wrapper = json.Wrapper

func NewExtConfiger() *ExtConfiger {
	return json.NewExtConfiger()
}

func NewWrapper(configer *ExtConfiger, v any) Wrapper {
	return json.NewWrapper(configer, v)
}

func Marshal(v any, configer *ExtConfiger) ([]byte, error) {
	return json.Marshal(NewWrapper(configer, v))
}
