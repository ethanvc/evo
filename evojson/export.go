package evojson

import "github.com/ethanvc/evo/evojson/internal/json"

type ExtConfiger = json.ExtConfiger
type Field = json.Field
type EncoderFunc = json.EncoderFunc
type EncodeState = json.EncodeState
type EncOpts = json.EncOpts

func NewExtConfiger() *ExtConfiger {
	return json.NewExtConfiger()
}

func Default() *ExtConfiger {
	return json.DefaultConfier.Load()
}

func SetDefault(configer *ExtConfiger) {
	json.DefaultConfier.Store(configer)
}

func Marshal(v any, configer ...*ExtConfiger) ([]byte, error) {
	return json.Marshal(v, configer...)
}
