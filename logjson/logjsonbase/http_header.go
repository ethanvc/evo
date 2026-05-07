package logjsonbase

import (
	"maps"
	"net/http"
)

type LogJsonConfigT map[string]func(string) string

func GetHttpHeader(logJsonConfig LogJsonConfigT, header http.Header) http.Header {
	var resultHeader http.Header
	for key, values := range header {
		f, ok := logJsonConfig[key]
		if !ok {
			continue
		}
		if resultHeader == nil {
			resultHeader = maps.Clone(header)
		}
		if f == nil {
			delete(resultHeader, key)
			continue
		}
		newValues := make([]string, len(values))
		for _, value := range values {
			newValues = append(newValues, f(value))
		}
		resultHeader[key] = newValues
	}
	if resultHeader == nil {
		return header
	}
	return resultHeader
}

func ConvertToHttpLogJsonConfig(config map[string]string) LogJsonConfigT {
	logJsonConfig := make(LogJsonConfigT, len(config))
	for key, value := range config {
		opt := ParseTag(value)
		newKey := http.CanonicalHeaderKey(key)
		if opt.Discard {
			logJsonConfig[newKey] = nil
		} else if opt.MD5 {
			logJsonConfig[newKey] = LogMd5Str
		}
	}
	return logJsonConfig
}
