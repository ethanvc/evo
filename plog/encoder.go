package plog

import (
	"strings"
	"sync/atomic"

	"github.com/ethanvc/evo/base"
	"github.com/ethanvc/evo/evojson"
)

type Encoder struct {
	globalEncoder base.SyncMap[string, evojson.EncoderFunc]
	configer      atomic.Pointer[evojson.ExtConfiger]
}

func NewEncoder() *Encoder {
	enc := &Encoder{}
	enc.resetConfiger()
	return enc
}

func (enc *Encoder) Set(key string, f evojson.EncoderFunc) {
	enc.globalEncoder.Store(key, f)
	enc.resetConfiger()
}

func (enc *Encoder) ClearAll() {
	enc.globalEncoder.ClearAll()
	enc.resetConfiger()
}

func (enc *Encoder) resetConfiger() {
	configer := evojson.NewExtConfiger()
	configer.CustomGetEncoder = enc.GetEncoder
	enc.configer.Store(configer)
}

func (en *Encoder) GetEncoder(configer *evojson.ExtConfiger, f *evojson.Field) evojson.EncoderFunc {
	lt := parseLogTag(f.StructField.Tag.Get("evolog"))
	if lt.Ignore {
		return Ignore
	}
	if lt.Md5 {
		return Md5
	}
	ef := en.globalEncoder.Load(f.Name)
	if ef != nil {
		return ef
	}
	return nil
}

var defaultEncoder atomic.Pointer[Encoder]

func init() {
	defaultEncoder.Store(NewEncoder())
}

func DefaultEncoder() *Encoder {
	return defaultEncoder.Load()
}

type logTag struct {
	Ignore bool
	Md5    bool
}

func parseLogTag(tag string) logTag {
	lt := logTag{}
	kvs := strings.Split(tag, ";")
	for _, kv := range kvs {
		k, _, _ := strings.Cut(kv, ":")
		switch k {
		case "md5":
			lt.Md5 = true
		case "ignore":
			lt.Ignore = true
		}
	}
	return lt
}
