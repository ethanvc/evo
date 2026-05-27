package httpcli

import "context"

var DefaultSerializer = &JsonSerializer{}

var sDefaultClient = &Client{}

func GetDefault() *Client {
	return sDefaultClient
}

func Do(ctx context.Context, url string, req, resp any, opts *Options) (*CliResp, error) {
	return GetDefault().Do(ctx, url, req, resp, opts)
}
