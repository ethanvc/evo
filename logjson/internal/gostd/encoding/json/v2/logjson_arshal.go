package json

import (
	"crypto/md5"
	"encoding/hex"
	"reflect"
	"strconv"

	"github.com/ethanvc/evo/logjson/internal/gostd/encoding/json/internal/jsonopts"
	"github.com/ethanvc/evo/logjson/internal/gostd/encoding/json/jsontext"
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
			// "len=" (4) + max int64 digits (20) + "," (1) + md5 hex (32) = 57
			var buf [57]byte
			data := getBytes(va)
			b := append(buf[:0], "len="...)
			b = strconv.AppendInt(b, int64(len(data)), 10)
			b = append(b, ',')
			h := md5.Sum(data)
			b = hex.AppendEncode(b, h[:])
			return enc.WriteToken(jsontext.String(string(b)))
		},
		unmarshal:  origFncs.unmarshal,
		nonDefault: true,
	}
	f.fncs = wrapped
}
