package httpcli

import "context"

var DefaultSerializer = &JsonSerializer{}

var sDefaultClient = &Client{}

func GetDefault() *Client {
	return sDefaultClient
}

func Do(ctx context.Context, url string, req, resp any, opts *Options) error {
	return GetDefault().Do(ctx, url, req, resp, opts)
}

func DoType[Resp any](ctx context.Context, url string, req any, opts *Options) (*Resp, error) {
	var resp Resp
	err := Do(ctx, url, req, &resp, opts)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
