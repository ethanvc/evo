package json

import (
	"reflect"

	"github.com/ethanvc/evo/logjson/logjsonbase"
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
	opts := logjsonbase.ParseTag(tag)
	if !opts.MD5 {
		return
	}
	k := sf.Type.Kind()
	if k == reflect.String || (k == reflect.Slice && sf.Type.Elem().Kind() == reflect.Uint8) {
		out.md5 = true
	}
}
