package httpmux

type Pattern = pattern

func (mux *ServeMux) MatchPattern(method, host, p string) (h Handler, patStr string, pattern *Pattern, matches []string) {
}
