package httpclient

import (
	"context"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestSingleAttempt_Basic(t *testing.T) {
	sa := NewSingleAttempt(context.Background(), http.MethodPost, "http://www.baidu.com")
	err := sa.Do(nil, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, sa.Response.StatusCode)
}
