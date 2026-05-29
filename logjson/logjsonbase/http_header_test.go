package logjsonbase

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetHttpHeader(t *testing.T) {
	t.Run("discard", func(t *testing.T) {
		header := http.Header{
			"Authorization": []string{"secret"},
			"X-Trace-Id":    []string{"trace-id"},
		}

		got := GetHttpHeader(HeaderLogConfig{"Authorization": nil}, header)
		require.Equal(t, 1, len(got))
		assert.Equal(t, "trace-id", got.Get("X-Trace-Id"))
	})

	t.Run("all unmatched returns original header", func(t *testing.T) {
		header := http.Header{
			"X-Trace-Id": []string{"trace-id"},
		}

		got := GetHttpHeader(HeaderLogConfig{
			"Authorization": nil,
			"X-Body":        LogMd5Str,
		}, header)
		require.True(t, SameMap(got, header))
	})
}

func SameMap[K comparable, V any](a, b map[K]V) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}

	return reflect.ValueOf(a).Pointer() == reflect.ValueOf(b).Pointer()
}
