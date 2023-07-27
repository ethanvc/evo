package evolog

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"reflect"

	"github.com/ethanvc/evo/evojson"
)

func Ignore(e *evojson.EncodeState, v reflect.Value, opts evojson.EncOpts) {
	e.WriteString(`""`)
}

func Md5(e *evojson.EncodeState, v reflect.Value, opts evojson.EncOpts) {
	md5val := ""
	contentLen := 0
	if v.Kind() == reflect.String {
		s := v.String()
		contentLen = len(s)
		md5val = calcMd5([]byte(s))
	} else if v.Kind() == reflect.Slice {
		if v.IsNil() {
			e.WriteString("null")
			return
		}
		s := v.Bytes()
		md5val = calcMd5(s)
		contentLen = len(s)
	}
	e.WriteString(fmt.Sprintf(`"%s(%d)"`, md5val, contentLen))
}

func calcMd5(v []byte) string {
	hash := md5.New()
	hash.Write(v)
	return hex.EncodeToString(hash.Sum(nil))
}
