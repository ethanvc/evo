package base

import "encoding/json"

func AnyToJson(v any) []byte {
	buf, _ := json.Marshal(v)
	return buf
}

func AnyToJsonString(v any) string {
	return string(AnyToJson(v))
}
