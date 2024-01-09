package httpcli

import (
	"context"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestSingleAttempt_Basic(t *testing.T) {
	sa := NewSingleAttempt(context.Background(), http.MethodGet, "https://www.baidu.com", nil)
	buf, err := Do[string](sa, "")
	require.NoError(t, err)
	require.Contains(t, buf, "baidu")
}
