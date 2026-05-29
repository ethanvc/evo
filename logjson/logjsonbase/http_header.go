package logjsonbase

import (
	"maps"
	"net/http"
)

type HeaderLogConfig map[string]func(string) string

func GetHttpHeader(logConfig HeaderLogConfig, header http.Header) http.Header {
	if logConfig == nil {
		logConfig = defaultHeaderLogConfig
	}
	var resultHeader http.Header
	for key, values := range header {
		f, ok := logConfig[key]
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

func ConvertToHeaderLogConfig(config map[string]string) HeaderLogConfig {
	logJsonConfig := make(HeaderLogConfig, len(config))
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

var defaultHeaderLogConfig = HeaderLogConfig{
	"Authorization": nil,
	"Connection":    nil,
}
