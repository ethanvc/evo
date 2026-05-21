package httpmux

type Pattern = pattern

func (mux *ServeMux) MatchPattern(method, host, p string) (pattern *Pattern, matches []string) {
	n, matches := mux.findHandler(method, host, p)
	if n == nil {
		return nil, matches
	}
	return n.pattern, matches
}
