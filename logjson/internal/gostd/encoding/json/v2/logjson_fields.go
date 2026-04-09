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
	applyLogjsonTag(sf, tag, out)
}

func logjsonResolveFieldOptions(structType reflect.Type, fieldIndex int, out *fieldOptions) {
	if out.md5 {
		return
	}
	tag := logjsonResolveProtoTag(structType, fieldIndex)
	if tag == "" {
		return
	}
	applyLogjsonTag(structType.Field(fieldIndex), tag, out)
}

func applyLogjsonTag(sf reflect.StructField, tag string, out *fieldOptions) {
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
