package httpcli

import (
	"context"
	"net/url"

	"github.com/ethanvc/evo/obs"
)

type LogInterceptor struct {
}

func NewLogInterceptor() *LogInterceptor {
	return &LogInterceptor{}
}

func (i *LogInterceptor) Intercept(ctx context.Context, url string, req any, resp any, opts *Options, next Next) (*CliResp, error) {
	ctx, ok := i.initSpan(ctx, url, opts)
	cliResp, err := next.Next(ctx, url, req, resp, opts)
	if ok {
		i.report(ctx, url, req, resp, opts, cliResp, err)
	}
	return cliResp, err
}

func (i *LogInterceptor) initSpan(ctx context.Context, urlStr string, opts *Options) (context.Context, bool) {
	parsedUrl, err := url.Parse(urlStr)
	if err != nil {
		obs.ErrReport(ctx, "ParseUrlErr", "err", err, "url", urlStr)
		return ctx, false
	}
	ctx, _ = obs.WithSpan(ctx, &obs.SpanConfig{
		Method: parsedUrl.Host + parsedUrl.Path,
	})
	return ctx, true
}

func (i *LogInterceptor) report(ctx context.Context, url string, req any, resp any, opts *Options, cliResp *CliResp, err error) {
	obsCtx := obs.GetObsContext(ctx)
	span := obsCtx.GetSpan()
	span.SetAttr("http.url", url)
	span.SetAttr("http.method", opts.GetMethod())
	span.SetAttr("http.header", opts.Header)
	if cliResp != nil && cliResp.Response != nil {
		httpResp := cliResp.Response
		span.SetAttr("http.status_code", httpResp.StatusCode)
		span.SetAttr("http.resp_header", httpResp.Header)
	}
	obsCtx.AccessLogReport(ctx, err, req, resp, nil)
}
