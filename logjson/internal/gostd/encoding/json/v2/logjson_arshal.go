package json

import (
	"reflect"

	"github.com/ethanvc/evo/logjson/internal/gostd/encoding/json/internal/jsonopts"
	"github.com/ethanvc/evo/logjson/internal/gostd/encoding/json/jsontext"
	"github.com/ethanvc/evo/logjson/logjsonbase"
)

func logjsonWrapArshaler(f *structField) {
	if !f.MD5 {
		return
	}

	var getBytes func(addressableValue) []byte
	switch f.typ.Kind() {
	case reflect.String:
		getBytes = func(va addressableValue) []byte { return []byte(va.String()) }
	case reflect.Slice:
		getBytes = func(va addressableValue) []byte { return va.Bytes() }
	default:
		return
	}

	origFncs := f.fncs
	wrapped := &arshaler{
		marshal: func(enc *jsontext.Encoder, va addressableValue, mo *jsonopts.Struct) error {
			return enc.WriteToken(jsontext.String(logjsonbase.LogMd5(getBytes(va))))
		},
		unmarshal:  origFncs.unmarshal,
		nonDefault: true,
	}
	f.fncs = wrapped
}
