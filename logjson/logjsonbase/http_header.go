package logjsonbase

import (
	"maps"
	"net/http"
)

func GetHttpHeader(logJsonConfig map[string]func(string) string, header http.Header) http.Header {
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
