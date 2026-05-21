package httpmux

import "net/http"

type Pattern = pattern

func (mux *ServeMux) MatchPattern(method, host, p string) (pattern *Pattern, matches []string) {
	req, _ := http.NewRequest(method, host+p, nil)
	_, _, pattern, matches = mux.findHandler(req)
	return pattern, matches
}
