package httpcli

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Sign(t *testing.T) {
	ctx := context.Background()
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		_ = body
		_, _ = w.Write([]byte(r.Header.Get("X-Signature")))
	}))
	defer svr.Close()
	cli := &Client{
		Interceptors: []Interceptor{sign},
	}
	signVar := ""
	err := cli.Do(ctx, svr.URL, "TEST", &signVar, nil)
	require.NoError(t, err)
	require.Equal(t, "29263cf66af2f73a34bbcdf8bae8edd6e24a7e34c42a235f98ec59a4e331977e", signVar)
}

func sign(ctx context.Context, url string, req, resp any, opts *Options, next Next) error {
	newReq, err := json.Marshal(req)
	if err != nil {
		return err
	}
	if opts.Header == nil {
		opts.Header = http.Header{}
	}
	h := hmac.New(sha256.New, []byte("123"))
	h.Write(newReq)
	opts.Header.Set("Content-Type", "application/json")
	opts.Header.Set("X-Signature", hex.EncodeToString(h.Sum(nil)))
	return next.Next(ctx, url, newReq, resp, opts)
}
