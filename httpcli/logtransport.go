package httpcli

import (
	"net/http"
)

type LogTransport struct {
	next http.RoundTripper
}

func (t *LogTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = t.init(req)
	resp, err := t.next.RoundTrip(req)
	// here does not catch panic to let upper layer catch and report it
	t.report(resp, err)
	return resp, err
}

func (t *LogTransport) init(req *http.Request) *http.Request {
	return req
}

func (t *LogTransport) report(resp *http.Response, err error) {

}
