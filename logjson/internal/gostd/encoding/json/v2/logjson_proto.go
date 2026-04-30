package json

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/ethanvc/evo/logjson/logjsonbase"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var protoMessageType = reflect.TypeFor[proto.Message]()

// logjsonResolveProtoTag reads the logjson extension from a proto field descriptor.
// Returns "" on any failure (degrade over error).
func logjsonResolveProtoTag(structType reflect.Type, fieldIndex int) string {
	if !reflect.PointerTo(structType).Implements(protoMessageType) {
		return ""
	}
	sf := structType.Field(fieldIndex)
	protoTag := sf.Tag.Get("protobuf")
	if protoTag == "" {
		return ""
	}
	fieldNum := parseProtoFieldNumber(protoTag)
	if fieldNum == 0 {
		return ""
	}
	msg := reflect.New(structType).Interface().(proto.Message)
	fd := msg.ProtoReflect().Descriptor().Fields().ByNumber(protoreflect.FieldNumber(fieldNum))
	if fd == nil {
		return ""
	}
	opts := fd.Options()
	if opts == nil {
		return ""
	}
	if !proto.HasExtension(opts, logjsonbase.E_Logjson) {
		return ""
	}
	val, ok := proto.GetExtension(opts, logjsonbase.E_Logjson).(string)
	if !ok {
		return ""
	}
	return val
}

func parseProtoFieldNumber(tag string) int {
	parts := strings.Split(tag, ",")
	if len(parts) < 2 {
		return 0
	}
	n, err := strconv.Atoi(parts[1])
	if err != nil || n <= 0 {
		return 0
	}
	return n
}
