package httpcli

import (
	"net/http"

	"github.com/ethanvc/evo/bevo"
	"github.com/ethanvc/evo/obs"
)

type LogTransport struct {
	next http.RoundTripper
}

func (t *LogTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = t.initSpan(req)
	reader := bevo.NewReader(req.Body)
	req.Body = reader
	reqBody, _ := reader.Peek(1024 * 5)
	resp, err := t.next.RoundTrip(req)
	// here does not catch panic to let upper layer catch and report it
	t.report(req, reqBody, resp, err)
	return resp, err
}

func (t *LogTransport) initSpan(req *http.Request) *http.Request {
	ctx, span := obs.WithSpan(req.Context(), &obs.SpanConfig{
		Method: req.URL.Host + req.URL.Path,
	})
	req = req.WithContext(ctx)
	span.SetAttr("http.url", req.URL.String())
	span.SetAttr("http.method", req.Method)
	span.SetAttr("http.host", req.Host)
	span.SetAttr("http.scheme", req.URL.Scheme)
	span.SetAttr("http.header", req.Header)
	return req
}

func (t *LogTransport) report(req *http.Request, reqBody []byte, resp *http.Response, err error) {
	obsCtx := obs.GetObsContext(req.Context())
	if err != nil {
		obsCtx.AccessLogReport(req.Context(), err, reqBody, resp, nil)
		return
	}
	obsCtx.AccessLogReport(req.Context(), nil, reqBody, resp, nil)
}
