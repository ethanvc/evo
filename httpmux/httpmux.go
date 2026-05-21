package httpmux

type Pattern = pattern

func (mux *ServeMux) MatchPattern(method, host, p string) (pattern *Pattern, matches []string) {
	_, _, pattern, matches = mux.findHandler(method, host, p, "")
	return pattern, matches
}
