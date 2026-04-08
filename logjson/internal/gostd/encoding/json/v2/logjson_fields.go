package json

import (
	"reflect"
	"strings"
)

type logjsonFieldOptions struct {
	md5 bool
}

func parseLogjsonFieldOptions(sf reflect.StructField, out *fieldOptions) {
	tag, hasTag := sf.Tag.Lookup("logjson")
	if !hasTag {
		return
	}
	for _, opt := range strings.Split(tag, ",") {
		opt = strings.TrimSpace(opt)
		switch opt {
		case "md5":
			k := sf.Type.Kind()
			if k == reflect.String || (k == reflect.Slice && sf.Type.Elem().Kind() == reflect.Uint8) {
				out.md5 = true
			}
		}
	}
}
