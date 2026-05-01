package json

import (
	"reflect"

	"github.com/ethanvc/evo/logjson/logjsonbase"
)

func parseLogjsonFieldOptions(sf reflect.StructField, out *fieldOptions) {
	tag, hasTag := sf.Tag.Lookup("logjson")
	if !hasTag {
		return
	}
	applyLogjsonTag(sf, tag, out)
}

func logjsonResolveFieldOptions(structType reflect.Type, fieldIndex int, out *fieldOptions) {
	if out.MD5 || out.Discard {
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
	applyLogjsonOptions(sf.Type, opts, out)
}

func applyLogjsonOptions(fieldType reflect.Type, opts logjsonbase.TagOptions, out *fieldOptions) {
	// Discard wins over any value-transforming option.
	out.Discard = opts.Discard
	if out.Discard {
		out.MD5 = false
		return
	}
	out.MD5 = opts.MD5 && logjsonSupportsMD5(fieldType)
}

func logjsonSupportsMD5(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.String:
		return true
	case reflect.Pointer:
		return t.Elem().Kind() == reflect.String
	case reflect.Slice:
		return t.Elem().Kind() == reflect.Uint8
	default:
		return false
	}
}
